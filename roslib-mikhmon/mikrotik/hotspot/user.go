package hotspot

import (
	"context"

	"github.com/quiqxiq/roslib"
	"github.com/quiqxiq/roslib-mikhmon/domain"
	"github.com/quiqxiq/roslib-mikhmon/mikrotik"
)

const userPath = "/ip/hotspot/user"

// UserList → /ip/hotspot/user/print (analisis §1.6).
func (c *Client) UserList(ctx context.Context) ([]domain.HotspotUser, error) {
	reply, err := c.dev.Path(userPath).Print().Exec(ctx)
	if err != nil {
		return nil, err
	}
	return sentencesToUsers(reply.Rows), nil
}

// UserCount → /ip/hotspot/user/print count-only="" (analisis §1.6).
func (c *Client) UserCount(ctx context.Context) (int, error) {
	reply, err := c.dev.Path(userPath).Print().Count().Exec(ctx)
	if err != nil {
		return 0, err
	}
	return mikrotik.AtoiOr(reply.Raw.Done.Map["ret"], 0), nil
}

// UserByName → /ip/hotspot/user/print ?name=<name>.
func (c *Client) UserByName(ctx context.Context, name string) (domain.HotspotUser, error) {
	if name == "" {
		return domain.HotspotUser{}, mikrotik.ErrInvalidArgument
	}
	reply, err := c.dev.Path(userPath).Print().Where("name", name).Exec(ctx)
	if err != nil {
		return domain.HotspotUser{}, err
	}
	if len(reply.Rows) == 0 {
		return domain.HotspotUser{}, mikrotik.ErrNotFound
	}
	return sentenceToUser(reply.Rows[0]), nil
}

// UserByID → /ip/hotspot/user/print ?.id=<id>.
func (c *Client) UserByID(ctx context.Context, id string) (domain.HotspotUser, error) {
	if id == "" {
		return domain.HotspotUser{}, mikrotik.ErrInvalidArgument
	}
	reply, err := c.dev.Path(userPath).Print().Where(".id", id).Exec(ctx)
	if err != nil {
		return domain.HotspotUser{}, err
	}
	if len(reply.Rows) == 0 {
		return domain.HotspotUser{}, mikrotik.ErrNotFound
	}
	return sentenceToUser(reply.Rows[0]), nil
}

// UserByProfile → /ip/hotspot/user/print ?profile=<name>.
func (c *Client) UserByProfile(ctx context.Context, profile string) ([]domain.HotspotUser, error) {
	if profile == "" {
		return nil, mikrotik.ErrInvalidArgument
	}
	reply, err := c.dev.Path(userPath).Print().Where("profile", profile).Exec(ctx)
	if err != nil {
		return nil, err
	}
	return sentencesToUsers(reply.Rows), nil
}

// UserByComment → /ip/hotspot/user/print ?comment=<value>.
func (c *Client) UserByComment(ctx context.Context, comment string) ([]domain.HotspotUser, error) {
	if comment == "" {
		return nil, mikrotik.ErrInvalidArgument
	}
	reply, err := c.dev.Path(userPath).Print().Where("comment", comment).Exec(ctx)
	if err != nil {
		return nil, err
	}
	return sentencesToUsers(reply.Rows), nil
}

// UserAddArgs adalah parameter UserAdd. Field kosong → tidak dikirim.
type UserAddArgs struct {
	Name            string // wajib
	Password        string
	Profile         string
	Server          string // "all" untuk semua
	Disabled        *bool
	LimitUptime     string
	LimitBytesTotal int64 // 0 = tidak dikirim
	LimitBytesIn    int64 // 0 = tidak dikirim (per-direction)
	LimitBytesOut   int64 // 0 = tidak dikirim (per-direction)
	Comment         string
}

// UserAdd → /ip/hotspot/user/add (analisis §1.6). Mengembalikan .id baru.
func (c *Client) UserAdd(ctx context.Context, a UserAddArgs) (string, error) {
	if a.Name == "" {
		return "", mikrotik.ErrInvalidArgument
	}
	pairs := []roslib.Pair{roslib.NewPair("name", a.Name)}
	if a.Password != "" {
		pairs = append(pairs, roslib.NewPair("password", a.Password))
	}
	if a.Profile != "" {
		pairs = append(pairs, roslib.NewPair("profile", a.Profile))
	}
	if a.Server != "" {
		pairs = append(pairs, roslib.NewPair("server", a.Server))
	}
	if a.Disabled != nil {
		pairs = append(pairs, roslib.NewPair("disabled", mikrotik.BoolWord(*a.Disabled)))
	}
	if a.LimitUptime != "" {
		pairs = append(pairs, roslib.NewPair("limit-uptime", a.LimitUptime))
	}
	if a.LimitBytesTotal > 0 {
		pairs = append(pairs, roslib.NewPair("limit-bytes-total", mikrotik.Itoa(a.LimitBytesTotal)))
	}
	if a.LimitBytesIn > 0 {
		pairs = append(pairs, roslib.NewPair("limit-bytes-in", mikrotik.Itoa(a.LimitBytesIn)))
	}
	if a.LimitBytesOut > 0 {
		pairs = append(pairs, roslib.NewPair("limit-bytes-out", mikrotik.Itoa(a.LimitBytesOut)))
	}
	if a.Comment != "" {
		pairs = append(pairs, roslib.NewPair("comment", a.Comment))
	}
	reply, err := c.dev.Path(userPath).Add(ctx, pairs...)
	if err != nil {
		return "", err
	}
	if reply.Done == nil {
		return "", nil
	}
	return reply.Done.Map["ret"], nil
}

// UserSetArgs adalah parameter UserSet (full update — mengikuti pemakaian
// di mikhmonv3 userbyname.php). Field kosong → tidak dikirim.
type UserSetArgs struct {
	ID              string // wajib
	Name            string
	Password        string
	Profile         string
	Server          string
	Disabled        *bool
	LimitUptime     string
	LimitBytesTotal *int64 // pointer supaya 0 tidak ambigu dengan unset
	LimitBytesIn    *int64
	LimitBytesOut   *int64
	Comment         *string
	MACAddress      *string
}

// UserSet → /ip/hotspot/user/set (analisis §1.6).
func (c *Client) UserSet(ctx context.Context, a UserSetArgs) error {
	if a.ID == "" {
		return mikrotik.ErrInvalidArgument
	}
	pairs := []roslib.Pair{}
	if a.Name != "" {
		pairs = append(pairs, roslib.NewPair("name", a.Name))
	}
	if a.Password != "" {
		pairs = append(pairs, roslib.NewPair("password", a.Password))
	}
	if a.Profile != "" {
		pairs = append(pairs, roslib.NewPair("profile", a.Profile))
	}
	if a.Server != "" {
		pairs = append(pairs, roslib.NewPair("server", a.Server))
	}
	if a.Disabled != nil {
		pairs = append(pairs, roslib.NewPair("disabled", mikrotik.BoolWord(*a.Disabled)))
	}
	if a.LimitUptime != "" {
		pairs = append(pairs, roslib.NewPair("limit-uptime", a.LimitUptime))
	}
	if a.LimitBytesTotal != nil {
		pairs = append(pairs, roslib.NewPair("limit-bytes-total", mikrotik.Itoa(*a.LimitBytesTotal)))
	}
	if a.LimitBytesIn != nil {
		pairs = append(pairs, roslib.NewPair("limit-bytes-in", mikrotik.Itoa(*a.LimitBytesIn)))
	}
	if a.LimitBytesOut != nil {
		pairs = append(pairs, roslib.NewPair("limit-bytes-out", mikrotik.Itoa(*a.LimitBytesOut)))
	}
	if a.Comment != nil {
		pairs = append(pairs, roslib.NewPair("comment", *a.Comment))
	}
	if a.MACAddress != nil {
		pairs = append(pairs, roslib.NewPair("mac-address", *a.MACAddress))
	}
	_, err := c.dev.Path(userPath).Set(ctx, a.ID, pairs...)
	return err
}

// UserSetDisabled toggle field disabled (analisis §1.6).
func (c *Client) UserSetDisabled(ctx context.Context, id string, disabled bool) error {
	if id == "" {
		return mikrotik.ErrInvalidArgument
	}
	_, err := c.dev.Path(userPath).Set(ctx, id, roslib.NewPair("disabled", mikrotik.BoolWord(disabled)))
	return err
}

// UserSetExpiry tulis comment user dengan expiry date (analisis §1.6,
// dipakai inline di on-login script).
func (c *Client) UserSetExpiry(ctx context.Context, id, expiry string) error {
	if id == "" {
		return mikrotik.ErrInvalidArgument
	}
	_, err := c.dev.Path(userPath).Set(ctx, id, roslib.NewPair("comment", expiry))
	return err
}

// UserSetMAC lock user ke MAC tertentu (analisis §1.6, varian Lock User).
func (c *Client) UserSetMAC(ctx context.Context, id, mac string) error {
	if id == "" {
		return mikrotik.ErrInvalidArgument
	}
	_, err := c.dev.Path(userPath).Set(ctx, id, roslib.NewPair("mac-address", mac))
	return err
}

// UserResetUsage set limit-uptime=0 dan kosongkan comment (analisis §1.6).
// Mikhmon memakai ini saat reset user ke kondisi awal.
func (c *Client) UserResetUsage(ctx context.Context, id string) error {
	if id == "" {
		return mikrotik.ErrInvalidArgument
	}
	_, err := c.dev.Path(userPath).Set(ctx, id,
		roslib.NewPair("limit-uptime", "0"),
		roslib.NewPair("comment", ""),
	)
	return err
}

// UserRemove → /ip/hotspot/user/remove (analisis §1.6).
func (c *Client) UserRemove(ctx context.Context, id string) error {
	if id == "" {
		return mikrotik.ErrInvalidArgument
	}
	_, err := c.dev.Path(userPath).Remove(ctx, id)
	return err
}

// UserResetCounters → /ip/hotspot/user/reset-counters (analisis §1.6).
func (c *Client) UserResetCounters(ctx context.Context, id string) error {
	if id == "" {
		return mikrotik.ErrInvalidArgument
	}
	_, err := c.dev.Path(userPath).Run(ctx, "reset-counters", roslib.NewPair("numbers", id))
	return err
}

// ───────── helpers ─────────

func sentenceToUser(s *roslib.Sentence) domain.HotspotUser {
	return domain.HotspotUser{
		ID:              s.Get(".id"),
		Name:            s.Get("name"),
		Password:        s.Get("password"),
		Profile:         s.Get("profile"),
		Server:          s.Get("server"),
		Disabled:        s.BoolOr("disabled", false),
		Comment:         s.Get("comment"),
		MACAddress:      s.Get("mac-address"),
		LimitUptime:     s.Get("limit-uptime"),
		LimitBytesTotal: s.IntOr("limit-bytes-total", 0),
		LimitBytesIn:    s.IntOr("limit-bytes-in", 0),
		LimitBytesOut:   s.IntOr("limit-bytes-out", 0),
		BytesIn:         s.IntOr("bytes-in", 0),
		BytesOut:        s.IntOr("bytes-out", 0),
		Uptime:          s.Get("uptime"),
	}
}

func sentencesToUsers(rows []*roslib.Sentence) []domain.HotspotUser {
	out := make([]domain.HotspotUser, 0, len(rows))
	for _, r := range rows {
		out = append(out, sentenceToUser(r))
	}
	return out
}
