package expiry

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/quiqxiq/roslib-mikhmon/mikrotik/hotspot"
	"github.com/quiqxiq/roslib-mikhmon/service/devmgr"
	"github.com/quiqxiq/roslib-mikhmon/store"
	"github.com/quiqxiq/roslib-mikhmon/store/model"
	"github.com/quiqxiq/roslib-mikhmon/workflows"
	"github.com/sirupsen/logrus"
)

// Service menjalankan satu goroutine checker per device yang memeriksa
// expiry user berdasarkan comment field (format mikhmon).
type Service struct {
	devMgr   *devmgr.Manager
	devices  store.DeviceStore
	profiles store.ProfileConfigStore
	txStore  store.TransactionStore
	log      *logrus.Logger

	mu      sync.Mutex
	cancels map[string]context.CancelFunc // key = device slug
	rootCtx context.Context
}

func New(
	mgr *devmgr.Manager,
	devices store.DeviceStore,
	profiles store.ProfileConfigStore,
	txStore store.TransactionStore,
	log *logrus.Logger,
) *Service {
	return &Service{
		devMgr:   mgr,
		devices:  devices,
		profiles: profiles,
		txStore:  txStore,
		log:      log,
		cancels:  make(map[string]context.CancelFunc),
	}
}

// Start menjalankan per-device goroutine checker. Non-blocking; semua goroutine
// dihentikan saat ctx dibatalkan.
func (s *Service) Start(ctx context.Context) error {
	s.rootCtx = ctx
	devs, err := s.devices.List(ctx)
	if err != nil {
		return err
	}
	s.mu.Lock()
	for _, d := range devs {
		s.startLocked(d)
	}
	s.mu.Unlock()
	return nil
}

// StartDevice menambah checker untuk device baru yang ditambah setelah Start().
// Idempotent — jika checker sudah jalan, tidak membuat duplikat.
func (s *Service) StartDevice(d model.MikrotikDevice) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.cancels[d.Slug]; exists {
		return
	}
	s.startLocked(d)
}

// StopDevice menghentikan checker untuk device yang dihapus.
func (s *Service) StopDevice(slug string) {
	s.mu.Lock()
	cancel, ok := s.cancels[slug]
	delete(s.cancels, slug)
	s.mu.Unlock()
	if ok {
		cancel()
	}
}

func (s *Service) startLocked(d model.MikrotikDevice) {
	ctx, cancel := context.WithCancel(s.rootCtx)
	s.cancels[d.Slug] = cancel
	go s.runChecker(ctx, d)
}

// runChecker adalah loop expiry checker untuk satu device.
func (s *Service) runChecker(ctx context.Context, d model.MikrotikDevice) {
	interval, err := time.ParseDuration(d.ExpiryCheckInterval)
	if err != nil || interval <= 0 {
		interval = 2 * time.Minute
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	log := s.log.WithField("device", d.Slug)
	log.Info("expiry: checker started")

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := s.checkDevice(ctx, d); err != nil {
				log.WithError(err).Warn("expiry: check failed")
			}
		}
	}
}

// checkDevice memeriksa semua user di satu device dan mengeksekusi aksi expired.
func (s *Service) checkDevice(ctx context.Context, d model.MikrotikDevice) error {
	cs, err := s.devMgr.Get(d.Slug)
	if err != nil {
		return err
	}

	users, err := cs.Hot.UserList(ctx)
	if err != nil {
		return fmt.Errorf("userlist: %w", err)
	}

	now := time.Now()
	for _, u := range users {
		expiry, ok := ParseExpiry(u.Comment)
		if !ok {
			continue
		}
		if now.Before(expiry) {
			continue
		}

		// User expired — ambil konfigurasi mode dari DB
		cfg, err := s.profiles.Get(ctx, d.ID, u.Profile)
		if err != nil {
			s.log.WithError(err).Warnf("expiry: cannot get profile config for %s", u.Profile)
			continue
		}

		if err := s.executeExpiry(ctx, cs, d, u.ID, u.Name, u.Profile, u.MACAddress, cfg); err != nil {
			s.log.WithError(err).Warnf("expiry: action failed for user %s", u.Name)
		}
	}
	return nil
}

// executeExpiry menjalankan aksi expiry sesuai mode di config.
func (s *Service) executeExpiry(
	ctx context.Context,
	cs *devmgr.ClientSet,
	d model.MikrotikDevice,
	userID, userName, profile, mac string,
	cfg model.HotspotProfileConfig,
) error {
	switch cfg.ExpiryMode {
	case "0":
		return nil

	case "rem", "remc":
		if err := workflows.DeleteUser(ctx, cs.WF, userID); err != nil {
			return err
		}

	case "ntf", "ntfc":
		// Set limit-uptime first so user cannot re-authenticate
		if err := cs.Hot.UserSet(ctx, hotspot.UserSetArgs{
			ID:          userID,
			LimitUptime: "1s",
		}); err != nil {
			return err
		}
		// Kick any active sessions (best-effort — don't fail on error)
		if actives, aerr := cs.Hot.ActiveList(ctx); aerr == nil {
			for _, a := range actives {
				if a.User == userName {
					_ = cs.Hot.ActiveRemove(ctx, a.ID)
				}
			}
		}

	default:
		return nil
	}

	// Catat transaksi jika mode berakhiran "c"
	if strings.HasSuffix(cfg.ExpiryMode, "c") {
		now := time.Now()
		tx := &model.Transaction{
			DeviceID:  d.ID,
			SaleDate:  now.Format("jan/02/2006"),
			SaleTime:  now.Format("15:04:05"),
			SaleMonth: now.Format("jan2006"),
			Username:  userName,
			Price:     cfg.Price,
			SellPrice: cfg.SellPrice,
			MAC:       mac,
			Validity:  cfg.Validity,
			Profile:   profile,
			Comment:   "expiry-auto",
		}
		if err := s.txStore.Create(ctx, tx); err != nil {
			s.log.WithError(err).Warn("expiry: failed to record transaction")
		}
	}
	return nil
}
