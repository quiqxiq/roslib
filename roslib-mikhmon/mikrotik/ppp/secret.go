package ppp

import (
	"context"

	"github.com/quiqxiq/roslib"
	"github.com/quiqxiq/roslib-mikhmon/domain"
	"github.com/quiqxiq/roslib-mikhmon/mikrotik"
)

const secretPath = "/ppp/secret"

// SecretList → /ppp/secret/print (analisis §1.12 — inferred).
func (c *Client) SecretList(ctx context.Context) ([]domain.PPPSecret, error) {
	reply, err := c.dev.Path(secretPath).Print().Exec(ctx)
	if err != nil {
		return nil, err
	}
	return sentencesToSecrets(reply.Rows), nil
}

// SecretByName → /ppp/secret/print ?name=<name>.
func (c *Client) SecretByName(ctx context.Context, name string) (domain.PPPSecret, error) {
	if name == "" {
		return domain.PPPSecret{}, mikrotik.ErrInvalidArgument
	}
	reply, err := c.dev.Path(secretPath).Print().Where("name", name).Exec(ctx)
	if err != nil {
		return domain.PPPSecret{}, err
	}
	if len(reply.Rows) == 0 {
		return domain.PPPSecret{}, mikrotik.ErrNotFound
	}
	return sentenceToSecret(reply.Rows[0]), nil
}

// SecretAddArgs adalah parameter SecretAdd.
type SecretAddArgs struct {
	Name       string // wajib
	Password   string
	Service    string // pppoe, l2tp, pptp, ovpn, etc
	Profile    string
	LocalAddr  string
	RemoteAddr string
	Disabled   *bool
	Comment    string
}

// SecretAdd → /ppp/secret/add (analisis §1.12).
func (c *Client) SecretAdd(ctx context.Context, a SecretAddArgs) (string, error) {
	if a.Name == "" {
		return "", mikrotik.ErrInvalidArgument
	}
	pairs := []roslib.Pair{roslib.NewPair("name", a.Name)}
	if a.Password != "" {
		pairs = append(pairs, roslib.NewPair("password", a.Password))
	}
	if a.Service != "" {
		pairs = append(pairs, roslib.NewPair("service", a.Service))
	}
	if a.Profile != "" {
		pairs = append(pairs, roslib.NewPair("profile", a.Profile))
	}
	if a.LocalAddr != "" {
		pairs = append(pairs, roslib.NewPair("local-address", a.LocalAddr))
	}
	if a.RemoteAddr != "" {
		pairs = append(pairs, roslib.NewPair("remote-address", a.RemoteAddr))
	}
	if a.Disabled != nil {
		pairs = append(pairs, roslib.NewPair("disabled", mikrotik.BoolWord(*a.Disabled)))
	}
	if a.Comment != "" {
		pairs = append(pairs, roslib.NewPair("comment", a.Comment))
	}
	reply, err := c.dev.Path(secretPath).Add(ctx, pairs...)
	if err != nil {
		return "", err
	}
	if reply.Done == nil {
		return "", nil
	}
	return reply.Done.Map["ret"], nil
}

// SecretSetArgs adalah parameter SecretSet.
type SecretSetArgs struct {
	ID         string // wajib
	Name       string
	Password   string
	Service    string
	Profile    string
	LocalAddr  string
	RemoteAddr string
	Disabled   *bool
	Comment    *string
}

// SecretSet → /ppp/secret/set (analisis §1.12).
func (c *Client) SecretSet(ctx context.Context, a SecretSetArgs) error {
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
	if a.Service != "" {
		pairs = append(pairs, roslib.NewPair("service", a.Service))
	}
	if a.Profile != "" {
		pairs = append(pairs, roslib.NewPair("profile", a.Profile))
	}
	if a.LocalAddr != "" {
		pairs = append(pairs, roslib.NewPair("local-address", a.LocalAddr))
	}
	if a.RemoteAddr != "" {
		pairs = append(pairs, roslib.NewPair("remote-address", a.RemoteAddr))
	}
	if a.Disabled != nil {
		pairs = append(pairs, roslib.NewPair("disabled", mikrotik.BoolWord(*a.Disabled)))
	}
	if a.Comment != nil {
		pairs = append(pairs, roslib.NewPair("comment", *a.Comment))
	}
	_, err := c.dev.Path(secretPath).Set(ctx, a.ID, pairs...)
	return err
}

// SecretSetDisabled toggle disabled (analisis §1.12).
func (c *Client) SecretSetDisabled(ctx context.Context, id string, disabled bool) error {
	if id == "" {
		return mikrotik.ErrInvalidArgument
	}
	_, err := c.dev.Path(secretPath).Set(ctx, id, roslib.NewPair("disabled", mikrotik.BoolWord(disabled)))
	return err
}

// SecretRemove → /ppp/secret/remove (analisis §1.12).
func (c *Client) SecretRemove(ctx context.Context, id string) error {
	if id == "" {
		return mikrotik.ErrInvalidArgument
	}
	_, err := c.dev.Path(secretPath).Remove(ctx, id)
	return err
}

func sentenceToSecret(s *roslib.Sentence) domain.PPPSecret {
	return domain.PPPSecret{
		ID:         s.Get(".id"),
		Name:       s.Get("name"),
		Password:   s.Get("password"),
		Service:    s.Get("service"),
		Profile:    s.Get("profile"),
		LocalAddr:  s.Get("local-address"),
		RemoteAddr: s.Get("remote-address"),
		Disabled:   s.BoolOr("disabled", false),
		Comment:    s.Get("comment"),
	}
}

func sentencesToSecrets(rows []*roslib.Sentence) []domain.PPPSecret {
	out := make([]domain.PPPSecret, 0, len(rows))
	for _, r := range rows {
		out = append(out, sentenceToSecret(r))
	}
	return out
}
