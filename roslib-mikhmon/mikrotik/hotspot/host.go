package hotspot

import (
	"context"

	"github.com/quiqxiq/roslib"
	"github.com/quiqxiq/roslib-mikhmon/domain"
	"github.com/quiqxiq/roslib-mikhmon/mikrotik"
)

const hostPath = "/ip/hotspot/host"

// HostList → /ip/hotspot/host/print (analisis §1.8).
func (c *Client) HostList(ctx context.Context) ([]domain.HotspotHost, error) {
	reply, err := c.dev.Path(hostPath).Print().Exec(ctx)
	if err != nil {
		return nil, err
	}
	return sentencesToHosts(reply.Rows), nil
}

// HostRemove → /ip/hotspot/host/remove (analisis §1.8).
func (c *Client) HostRemove(ctx context.Context, id string) error {
	if id == "" {
		return mikrotik.ErrInvalidArgument
	}
	_, err := c.dev.Path(hostPath).Remove(ctx, id)
	return err
}

// ───────── helpers ─────────

func sentencesToHosts(rows []*roslib.Sentence) []domain.HotspotHost {
	out := make([]domain.HotspotHost, 0, len(rows))
	for _, s := range rows {
		out = append(out, domain.HotspotHost{
			ID:         s.Get(".id"),
			MACAddress: s.Get("mac-address"),
			Address:    s.Get("address"),
			ToAddress:  s.Get("to-address"),
			Server:     s.Get("server"),
			Authorized: s.BoolOr("authorized", false),
			Bypassed:   s.BoolOr("bypassed", false),
			Comment:    s.Get("comment"),
		})
	}
	return out
}
