package network

import (
	"context"

	"github.com/quiqxiq/roslib-mikhmon/domain"
)

// InterfaceList → /interface/print (analisis §1.11).
func (c *Client) InterfaceList(ctx context.Context) ([]domain.Interface, error) {
	reply, err := c.dev.Path("/interface").Print().Exec(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]domain.Interface, 0, len(reply.Rows))
	for _, s := range reply.Rows {
		out = append(out, domain.Interface{
			ID:       s.Get(".id"),
			Name:     s.Get("name"),
			Type:     s.Get("type"),
			Running:  s.BoolOr("running", false),
			Disabled: s.BoolOr("disabled", false),
			Comment:  s.Get("comment"),
		})
	}
	return out, nil
}

// MonitorTrafficSnapshot dihapus pada refactor — pakai InterfaceTrafficStream
// di iface_stream.go untuk continuous monitoring. Kalau butuh one-shot
// snapshot, panggil dev langsung:
//
//	dev.Path("/interface/monitor-traffic").
//	    With("interface", iface).With("once", "").
//	    Print().Exec(ctx)
