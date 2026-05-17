package ppp

import (
	"context"

	"github.com/quiqxiq/roslib"
	"github.com/quiqxiq/roslib-mikhmon/domain"
	"github.com/quiqxiq/roslib-mikhmon/mikrotik"
)

const profilePath = "/ppp/profile"

// ProfileList → /ppp/profile/print (analisis §1.12 — inferred).
func (c *Client) ProfileList(ctx context.Context) ([]domain.PPPProfile, error) {
	reply, err := c.dev.Path(profilePath).Print().Exec(ctx)
	if err != nil {
		return nil, err
	}
	return sentencesToProfiles(reply.Rows), nil
}

// ProfileByName → /ppp/profile/print ?name=<name>.
func (c *Client) ProfileByName(ctx context.Context, name string) (domain.PPPProfile, error) {
	if name == "" {
		return domain.PPPProfile{}, mikrotik.ErrInvalidArgument
	}
	reply, err := c.dev.Path(profilePath).Print().Where("name", name).Exec(ctx)
	if err != nil {
		return domain.PPPProfile{}, err
	}
	if len(reply.Rows) == 0 {
		return domain.PPPProfile{}, mikrotik.ErrNotFound
	}
	return sentenceToProfile(reply.Rows[0]), nil
}

// ProfileAddArgs adalah parameter ProfileAdd.
type ProfileAddArgs struct {
	Name       string // wajib
	LocalAddr  string
	RemoteAddr string
	RateLimit  string
	Comment    string
}

// ProfileAdd → /ppp/profile/add (analisis §1.12).
func (c *Client) ProfileAdd(ctx context.Context, a ProfileAddArgs) (string, error) {
	if a.Name == "" {
		return "", mikrotik.ErrInvalidArgument
	}
	pairs := []roslib.Pair{roslib.NewPair("name", a.Name)}
	if a.LocalAddr != "" {
		pairs = append(pairs, roslib.NewPair("local-address", a.LocalAddr))
	}
	if a.RemoteAddr != "" {
		pairs = append(pairs, roslib.NewPair("remote-address", a.RemoteAddr))
	}
	if a.RateLimit != "" {
		pairs = append(pairs, roslib.NewPair("rate-limit", a.RateLimit))
	}
	if a.Comment != "" {
		pairs = append(pairs, roslib.NewPair("comment", a.Comment))
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
	ID         string // wajib
	Name       string
	LocalAddr  string
	RemoteAddr string
	RateLimit  string
	Comment    *string
}

// ProfileSet → /ppp/profile/set (analisis §1.12).
func (c *Client) ProfileSet(ctx context.Context, a ProfileSetArgs) error {
	if a.ID == "" {
		return mikrotik.ErrInvalidArgument
	}
	pairs := []roslib.Pair{}
	if a.Name != "" {
		pairs = append(pairs, roslib.NewPair("name", a.Name))
	}
	if a.LocalAddr != "" {
		pairs = append(pairs, roslib.NewPair("local-address", a.LocalAddr))
	}
	if a.RemoteAddr != "" {
		pairs = append(pairs, roslib.NewPair("remote-address", a.RemoteAddr))
	}
	if a.RateLimit != "" {
		pairs = append(pairs, roslib.NewPair("rate-limit", a.RateLimit))
	}
	if a.Comment != nil {
		pairs = append(pairs, roslib.NewPair("comment", *a.Comment))
	}
	_, err := c.dev.Path(profilePath).Set(ctx, a.ID, pairs...)
	return err
}

// ProfileRemove → /ppp/profile/remove (analisis §1.12).
func (c *Client) ProfileRemove(ctx context.Context, id string) error {
	if id == "" {
		return mikrotik.ErrInvalidArgument
	}
	_, err := c.dev.Path(profilePath).Remove(ctx, id)
	return err
}

func sentenceToProfile(s *roslib.Sentence) domain.PPPProfile {
	return domain.PPPProfile{
		ID:         s.Get(".id"),
		Name:       s.Get("name"),
		LocalAddr:  s.Get("local-address"),
		RemoteAddr: s.Get("remote-address"),
		RateLimit:  s.Get("rate-limit"),
		Comment:    s.Get("comment"),
	}
}

func sentencesToProfiles(rows []*roslib.Sentence) []domain.PPPProfile {
	out := make([]domain.PPPProfile, 0, len(rows))
	for _, r := range rows {
		out = append(out, sentenceToProfile(r))
	}
	return out
}
