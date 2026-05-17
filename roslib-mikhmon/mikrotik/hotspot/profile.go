package hotspot

import (
	"context"
	"time"

	"github.com/quiqxiq/roslib"
	"github.com/quiqxiq/roslib-mikhmon/domain"
	"github.com/quiqxiq/roslib-mikhmon/mikrotik"
)

const profilePath = "/ip/hotspot/user/profile"

// ProfileList → /ip/hotspot/user/profile/print (analisis §1.7).
func (c *Client) ProfileList(ctx context.Context) ([]domain.HotspotProfile, error) {
	reply, err := c.dev.Path(profilePath).Print().Exec(ctx)
	if err != nil {
		return nil, err
	}
	return sentencesToProfiles(reply.Rows), nil
}

// ProfileListCached → /ip/hotspot/user/profile/print dengan TTL cache.
// Cocok untuk dropdown profile yang refresh sering tapi profile jarang
// berubah. Setelah mutation, caller panggil dev.InvalidateCache(ctx, profilePath).
func (c *Client) ProfileListCached(ctx context.Context, ttl time.Duration) ([]domain.HotspotProfile, error) {
	reply, err := c.dev.Path(profilePath).Print().ExecCached(ctx, ttl)
	if err != nil {
		return nil, err
	}
	return sentencesToProfiles(reply.Rows), nil
}

// ProfileByName → /ip/hotspot/user/profile/print ?name=<name>.
func (c *Client) ProfileByName(ctx context.Context, name string) (domain.HotspotProfile, error) {
	if name == "" {
		return domain.HotspotProfile{}, mikrotik.ErrInvalidArgument
	}
	reply, err := c.dev.Path(profilePath).Print().Where("name", name).Exec(ctx)
	if err != nil {
		return domain.HotspotProfile{}, err
	}
	if len(reply.Rows) == 0 {
		return domain.HotspotProfile{}, mikrotik.ErrNotFound
	}
	return sentenceToProfile(reply.Rows[0]), nil
}

// ProfileByID → /ip/hotspot/user/profile/print ?.id=<id>.
func (c *Client) ProfileByID(ctx context.Context, id string) (domain.HotspotProfile, error) {
	if id == "" {
		return domain.HotspotProfile{}, mikrotik.ErrInvalidArgument
	}
	reply, err := c.dev.Path(profilePath).Print().Where(".id", id).Exec(ctx)
	if err != nil {
		return domain.HotspotProfile{}, err
	}
	if len(reply.Rows) == 0 {
		return domain.HotspotProfile{}, mikrotik.ErrNotFound
	}
	return sentenceToProfile(reply.Rows[0]), nil
}

// ProfileAddArgs adalah parameter ProfileAdd.
type ProfileAddArgs struct {
	Name              string // wajib
	AddressPool       string
	RateLimit         string
	SharedUsers       int
	StatusAutorefresh string // mis. "1m"
	OnLogin           string // body script — biasanya hasil scripts/onlogin.Build*
	ParentQueue       string
}

// ProfileAdd → /ip/hotspot/user/profile/add (analisis §1.7).
// Mengembalikan .id baru.
func (c *Client) ProfileAdd(ctx context.Context, a ProfileAddArgs) (string, error) {
	if a.Name == "" {
		return "", mikrotik.ErrInvalidArgument
	}
	pairs := []roslib.Pair{roslib.NewPair("name", a.Name)}
	if a.AddressPool != "" {
		pairs = append(pairs, roslib.NewPair("address-pool", a.AddressPool))
	}
	if a.RateLimit != "" {
		pairs = append(pairs, roslib.NewPair("rate-limit", a.RateLimit))
	}
	if a.SharedUsers > 0 {
		pairs = append(pairs, roslib.NewPair("shared-users", mikrotik.Itoa(int64(a.SharedUsers))))
	}
	if a.StatusAutorefresh != "" {
		pairs = append(pairs, roslib.NewPair("status-autorefresh", a.StatusAutorefresh))
	}
	if a.OnLogin != "" {
		pairs = append(pairs, roslib.NewPair("on-login", a.OnLogin))
	}
	if a.ParentQueue != "" {
		pairs = append(pairs, roslib.NewPair("parent-queue", a.ParentQueue))
	}
	reply, err := c.dev.Path(profilePath).Add(ctx, pairs...)
	if err != nil {
		return "", err
	}
	if reply.Done == nil {
		return "", nil
	}
	return reply.Done.Map["ret"], nil
}

// ProfileSetArgs adalah parameter ProfileSet.
type ProfileSetArgs struct {
	ID                string // wajib
	Name              string
	AddressPool       string
	RateLimit         string
	SharedUsers       *int
	StatusAutorefresh string
	OnLogin           *string
	ParentQueue       string
}

// ProfileSet → /ip/hotspot/user/profile/set (analisis §1.7).
func (c *Client) ProfileSet(ctx context.Context, a ProfileSetArgs) error {
	if a.ID == "" {
		return mikrotik.ErrInvalidArgument
	}
	pairs := []roslib.Pair{}
	if a.Name != "" {
		pairs = append(pairs, roslib.NewPair("name", a.Name))
	}
	if a.AddressPool != "" {
		pairs = append(pairs, roslib.NewPair("address-pool", a.AddressPool))
	}
	if a.RateLimit != "" {
		pairs = append(pairs, roslib.NewPair("rate-limit", a.RateLimit))
	}
	if a.SharedUsers != nil {
		pairs = append(pairs, roslib.NewPair("shared-users", mikrotik.Itoa(int64(*a.SharedUsers))))
	}
	if a.StatusAutorefresh != "" {
		pairs = append(pairs, roslib.NewPair("status-autorefresh", a.StatusAutorefresh))
	}
	if a.OnLogin != nil {
		pairs = append(pairs, roslib.NewPair("on-login", *a.OnLogin))
	}
	if a.ParentQueue != "" {
		pairs = append(pairs, roslib.NewPair("parent-queue", a.ParentQueue))
	}
	_, err := c.dev.Path(profilePath).Set(ctx, a.ID, pairs...)
	return err
}

// ProfileRemove → /ip/hotspot/user/profile/remove (analisis §1.7).
func (c *Client) ProfileRemove(ctx context.Context, id string) error {
	if id == "" {
		return mikrotik.ErrInvalidArgument
	}
	_, err := c.dev.Path(profilePath).Remove(ctx, id)
	return err
}

// ───────── helpers ─────────

func sentenceToProfile(s *roslib.Sentence) domain.HotspotProfile {
	return domain.HotspotProfile{
		ID:                s.Get(".id"),
		Name:              s.Get("name"),
		AddressPool:       s.Get("address-pool"),
		RateLimit:         s.Get("rate-limit"),
		SharedUsers:       int(s.IntOr("shared-users", 1)),
		StatusAutorefresh: s.Get("status-autorefresh"),
		OnLogin:           s.Get("on-login"),
		ParentQueue:       s.Get("parent-queue"),
	}
}

func sentencesToProfiles(rows []*roslib.Sentence) []domain.HotspotProfile {
	out := make([]domain.HotspotProfile, 0, len(rows))
	for _, r := range rows {
		out = append(out, sentenceToProfile(r))
	}
	return out
}
