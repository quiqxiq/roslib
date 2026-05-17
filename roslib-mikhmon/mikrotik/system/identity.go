package system

import (
	"context"
	"time"

	"github.com/quiqxiq/roslib-mikhmon/domain"
	"github.com/quiqxiq/roslib-mikhmon/mikrotik"
)

// Identity → /system/identity/print (analisis §1.1).
func (c *Client) Identity(ctx context.Context) (domain.SystemIdentity, error) {
	reply, err := c.dev.Path("/system/identity").Print().Exec(ctx)
	if err != nil {
		return domain.SystemIdentity{}, err
	}
	if len(reply.Rows) == 0 {
		return domain.SystemIdentity{}, mikrotik.ErrNotFound
	}
	return domain.SystemIdentity{Name: reply.Rows[0].Get("name")}, nil
}

// IdentityCached → /system/identity/print dengan TTL cache. Identity nyaris
// immutable; cocok untuk dashboard header yang refresh sering.
func (c *Client) IdentityCached(ctx context.Context, ttl time.Duration) (domain.SystemIdentity, error) {
	reply, err := c.dev.Path("/system/identity").Print().ExecCached(ctx, ttl)
	if err != nil {
		return domain.SystemIdentity{}, err
	}
	if len(reply.Rows) == 0 {
		return domain.SystemIdentity{}, mikrotik.ErrNotFound
	}
	return domain.SystemIdentity{Name: reply.Rows[0].Get("name")}, nil
}
