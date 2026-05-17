package network

import (
	"context"

	"github.com/quiqxiq/roslib"
	"github.com/quiqxiq/roslib-mikhmon/domain"
	"github.com/quiqxiq/roslib-mikhmon/mikrotik"
)

const dhcpPath = "/ip/dhcp-server/lease"

// DHCPLeaseList → /ip/dhcp-server/lease/print (analisis §1.10).
func (c *Client) DHCPLeaseList(ctx context.Context) ([]domain.DHCPLease, error) {
	reply, err := c.dev.Path(dhcpPath).Print().Exec(ctx)
	if err != nil {
		return nil, err
	}
	return sentencesToLeases(reply.Rows), nil
}

// DHCPLeaseCount → /ip/dhcp-server/lease/print count-only="" (analisis §1.10).
func (c *Client) DHCPLeaseCount(ctx context.Context) (int, error) {
	reply, err := c.dev.Path(dhcpPath).Print().Count().Exec(ctx)
	if err != nil {
		return 0, err
	}
	return mikrotik.AtoiOr(reply.Raw.Done.Map["ret"], 0), nil
}

// DHCPLeaseByMAC → /ip/dhcp-server/lease/print ?mac-address=<mac> (analisis §1.10).
func (c *Client) DHCPLeaseByMAC(ctx context.Context, mac string) ([]domain.DHCPLease, error) {
	if mac == "" {
		return nil, mikrotik.ErrInvalidArgument
	}
	reply, err := c.dev.Path(dhcpPath).Print().Where("mac-address", mac).Exec(ctx)
	if err != nil {
		return nil, err
	}
	return sentencesToLeases(reply.Rows), nil
}

// DHCPLeaseRemove → /ip/dhcp-server/lease/remove (analisis §1.10).
func (c *Client) DHCPLeaseRemove(ctx context.Context, id string) error {
	if id == "" {
		return mikrotik.ErrInvalidArgument
	}
	_, err := c.dev.Path(dhcpPath).Remove(ctx, id)
	return err
}

func sentencesToLeases(rows []*roslib.Sentence) []domain.DHCPLease {
	out := make([]domain.DHCPLease, 0, len(rows))
	for _, s := range rows {
		out = append(out, domain.DHCPLease{
			ID:         s.Get(".id"),
			Address:    s.Get("address"),
			MACAddress: s.Get("mac-address"),
			HostName:   s.Get("host-name"),
			Server:     s.Get("server"),
			Status:     s.Get("status"),
			Dynamic:    s.BoolOr("dynamic", false),
			Disabled:   s.BoolOr("disabled", false),
			Comment:    s.Get("comment"),
		})
	}
	return out
}
