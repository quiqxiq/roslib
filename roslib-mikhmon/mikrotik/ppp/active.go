package ppp

import (
	"context"

	"github.com/quiqxiq/roslib-mikhmon/domain"
	"github.com/quiqxiq/roslib-mikhmon/mikrotik"
)

const activePath = "/ppp/active"

// ActiveList → /ppp/active/print (analisis §1.12 — inferred).
func (c *Client) ActiveList(ctx context.Context) ([]domain.PPPActive, error) {
	reply, err := c.dev.Path(activePath).Print().Exec(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]domain.PPPActive, 0, len(reply.Rows))
	for _, s := range reply.Rows {
		out = append(out, domain.PPPActive{
			ID:       s.Get(".id"),
			Name:     s.Get("name"),
			Service:  s.Get("service"),
			CallerID: s.Get("caller-id"),
			Address:  s.Get("address"),
			Uptime:   s.Get("uptime"),
		})
	}
	return out, nil
}

// ActiveRemove → /ppp/active/remove (analisis §1.12; ini SATU-SATUNYA
// PPP command yang file PHP-nya ADA di repo: process/removepactive.php).
func (c *Client) ActiveRemove(ctx context.Context, id string) error {
	if id == "" {
		return mikrotik.ErrInvalidArgument
	}
	_, err := c.dev.Path(activePath).Remove(ctx, id)
	return err
}

