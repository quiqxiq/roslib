package hotspot

import (
	"context"

	"github.com/quiqxiq/roslib"
	"github.com/quiqxiq/roslib-mikhmon/domain"
	"github.com/quiqxiq/roslib-mikhmon/mikrotik"
)

const bindingPath = "/ip/hotspot/ip-binding"

// BindingType menjelaskan jenis IP binding di RouterOS.
type BindingType string

const (
	BindingTypeRegular  BindingType = "regular"
	BindingTypeBypassed BindingType = "bypassed"
	BindingTypeBlocked  BindingType = "blocked"
)

// BindingList → /ip/hotspot/ip-binding/print (analisis §1.9).
func (c *Client) BindingList(ctx context.Context) ([]domain.HotspotBinding, error) {
	reply, err := c.dev.Path(bindingPath).Print().Exec(ctx)
	if err != nil {
		return nil, err
	}
	return sentencesToBindings(reply.Rows), nil
}

// BindingByID → /ip/hotspot/ip-binding/print ?.id=<id>.
func (c *Client) BindingByID(ctx context.Context, id string) (domain.HotspotBinding, error) {
	if id == "" {
		return domain.HotspotBinding{}, mikrotik.ErrInvalidArgument
	}
	reply, err := c.dev.Path(bindingPath).Print().Where(".id", id).Exec(ctx)
	if err != nil {
		return domain.HotspotBinding{}, err
	}
	if len(reply.Rows) == 0 {
		return domain.HotspotBinding{}, mikrotik.ErrNotFound
	}
	return sentenceToBinding(reply.Rows[0]), nil
}

// BindingCount → /ip/hotspot/ip-binding/print count-only="" (analisis §1.9).
func (c *Client) BindingCount(ctx context.Context) (int, error) {
	reply, err := c.dev.Path(bindingPath).Print().Count().Exec(ctx)
	if err != nil {
		return 0, err
	}
	return mikrotik.AtoiOr(reply.Raw.Done.Map["ret"], 0), nil
}

// BindingSetType → /ip/hotspot/ip-binding/set =type=... (analisis §1.9).
func (c *Client) BindingSetType(ctx context.Context, id string, t BindingType) error {
	if id == "" || t == "" {
		return mikrotik.ErrInvalidArgument
	}
	_, err := c.dev.Path(bindingPath).Set(ctx, id, roslib.NewPair("type", string(t)))
	return err
}

// BindingSetDisabled → /ip/hotspot/ip-binding/set =disabled=... (analisis §1.9).
func (c *Client) BindingSetDisabled(ctx context.Context, id string, disabled bool) error {
	if id == "" {
		return mikrotik.ErrInvalidArgument
	}
	_, err := c.dev.Path(bindingPath).Set(ctx, id, roslib.NewPair("disabled", mikrotik.BoolWord(disabled)))
	return err
}

// BindingRemove → /ip/hotspot/ip-binding/remove (analisis §1.9).
func (c *Client) BindingRemove(ctx context.Context, id string) error {
	if id == "" {
		return mikrotik.ErrInvalidArgument
	}
	_, err := c.dev.Path(bindingPath).Remove(ctx, id)
	return err
}

func sentenceToBinding(s *roslib.Sentence) domain.HotspotBinding {
	return domain.HotspotBinding{
		ID:         s.Get(".id"),
		MACAddress: s.Get("mac-address"),
		Address:    s.Get("address"),
		ToAddress:  s.Get("to-address"),
		Server:     s.Get("server"),
		Type:       s.Get("type"),
		Disabled:   s.BoolOr("disabled", false),
		Comment:    s.Get("comment"),
	}
}

func sentencesToBindings(rows []*roslib.Sentence) []domain.HotspotBinding {
	out := make([]domain.HotspotBinding, 0, len(rows))
	for _, s := range rows {
		out = append(out, sentenceToBinding(s))
	}
	return out
}
