package hotspot

import (
	"context"

	"github.com/quiqxiq/roslib"
	"github.com/quiqxiq/roslib-mikhmon/domain"
	"github.com/quiqxiq/roslib-mikhmon/mikrotik"
)

const activePath = "/ip/hotspot/active"

// ActiveList → /ip/hotspot/active/print (analisis §1.8).
func (c *Client) ActiveList(ctx context.Context) ([]domain.HotspotActive, error) {
	reply, err := c.dev.Path(activePath).Print().Exec(ctx)
	if err != nil {
		return nil, err
	}
	return sentencesToActives(reply.Rows), nil
}

// ActiveCount → /ip/hotspot/active/print count-only="" (analisis §1.8).
func (c *Client) ActiveCount(ctx context.Context) (int, error) {
	reply, err := c.dev.Path(activePath).Print().Count().Exec(ctx)
	if err != nil {
		return 0, err
	}
	return mikrotik.AtoiOr(reply.Raw.Done.Map["ret"], 0), nil
}

// ActiveByServer → /ip/hotspot/active/print ?server=<name>.
func (c *Client) ActiveByServer(ctx context.Context, server string) ([]domain.HotspotActive, error) {
	if server == "" {
		return nil, mikrotik.ErrInvalidArgument
	}
	reply, err := c.dev.Path(activePath).Print().Where("server", server).Exec(ctx)
	if err != nil {
		return nil, err
	}
	return sentencesToActives(reply.Rows), nil
}

// ActiveCountByServer → /ip/hotspot/active/print count-only="" ?server=<name>.
func (c *Client) ActiveCountByServer(ctx context.Context, server string) (int, error) {
	if server == "" {
		return 0, mikrotik.ErrInvalidArgument
	}
	reply, err := c.dev.Path(activePath).Print().Count().Where("server", server).Exec(ctx)
	if err != nil {
		return 0, err
	}
	return mikrotik.AtoiOr(reply.Raw.Done.Map["ret"], 0), nil
}

// ActiveByID → /ip/hotspot/active/print ?.id=<id>.
func (c *Client) ActiveByID(ctx context.Context, id string) (domain.HotspotActive, error) {
	if id == "" {
		return domain.HotspotActive{}, mikrotik.ErrInvalidArgument
	}
	reply, err := c.dev.Path(activePath).Print().Where(".id", id).Exec(ctx)
	if err != nil {
		return domain.HotspotActive{}, err
	}
	if len(reply.Rows) == 0 {
		return domain.HotspotActive{}, mikrotik.ErrNotFound
	}
	return sentenceToActive(reply.Rows[0]), nil
}

// ActiveRemove → /ip/hotspot/active/remove (kick session — analisis §1.8).
func (c *Client) ActiveRemove(ctx context.Context, id string) error {
	if id == "" {
		return mikrotik.ErrInvalidArgument
	}
	_, err := c.dev.Path(activePath).Remove(ctx, id)
	return err
}

// ───────── helpers ─────────

func sentenceToActive(s *roslib.Sentence) domain.HotspotActive {
	return domain.HotspotActive{
		ID:         s.Get(".id"),
		User:       s.Get("user"),
		Address:    s.Get("address"),
		MACAddress: s.Get("mac-address"),
		Server:     s.Get("server"),
		LoginBy:    s.Get("login-by"),
		Uptime:     s.Get("uptime"),
		BytesIn:    s.IntOr("bytes-in", 0),
		BytesOut:   s.IntOr("bytes-out", 0),
	}
}

func sentencesToActives(rows []*roslib.Sentence) []domain.HotspotActive {
	out := make([]domain.HotspotActive, 0, len(rows))
	for _, r := range rows {
		out = append(out, sentenceToActive(r))
	}
	return out
}
