package network

import (
	"context"
	"time"

	"github.com/quiqxiq/roslib"
	"github.com/quiqxiq/roslib-mikhmon/domain"
)

// IPPoolList → /ip/pool/print (analisis §1.10).
func (c *Client) IPPoolList(ctx context.Context) ([]domain.IPPool, error) {
	reply, err := c.dev.Path("/ip/pool").Print().Exec(ctx)
	if err != nil {
		return nil, err
	}
	return sentencesToPools(reply.Rows), nil
}

// PoolListCached → /ip/pool/print dengan TTL cache. IP pool nyaris
// immutable; cocok untuk dropdown profile yang refresh sering.
func (c *Client) PoolListCached(ctx context.Context, ttl time.Duration) ([]domain.IPPool, error) {
	reply, err := c.dev.Path("/ip/pool").Print().ExecCached(ctx, ttl)
	if err != nil {
		return nil, err
	}
	return sentencesToPools(reply.Rows), nil
}

func sentencesToPools(rows []*roslib.Sentence) []domain.IPPool {
	out := make([]domain.IPPool, 0, len(rows))
	for _, s := range rows {
		out = append(out, domain.IPPool{
			ID:     s.Get(".id"),
			Name:   s.Get("name"),
			Ranges: s.Get("ranges"),
		})
	}
	return out
}
