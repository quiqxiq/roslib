package network

import (
	"context"

	"github.com/quiqxiq/roslib-mikhmon/domain"
	"github.com/quiqxiq/roslib-mikhmon/mikrotik"
)

// ARPByMAC → /ip/arp/print ?mac-address=<mac> (analisis §1.10).
func (c *Client) ARPByMAC(ctx context.Context, mac string) ([]domain.ARPEntry, error) {
	if mac == "" {
		return nil, mikrotik.ErrInvalidArgument
	}
	reply, err := c.dev.Path("/ip/arp").Print().Where("mac-address", mac).Exec(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]domain.ARPEntry, 0, len(reply.Rows))
	for _, s := range reply.Rows {
		out = append(out, domain.ARPEntry{
			ID:         s.Get(".id"),
			Address:    s.Get("address"),
			MACAddress: s.Get("mac-address"),
			Interface:  s.Get("interface"),
			Dynamic:    s.BoolOr("dynamic", false),
			Disabled:   s.BoolOr("disabled", false),
			Comment:    s.Get("comment"),
		})
	}
	return out, nil
}

// ARPRemove → /ip/arp/remove (analisis §1.10).
func (c *Client) ARPRemove(ctx context.Context, id string) error {
	if id == "" {
		return mikrotik.ErrInvalidArgument
	}
	_, err := c.dev.Path("/ip/arp").Remove(ctx, id)
	return err
}
