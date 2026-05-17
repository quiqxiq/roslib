package network

import (
	"context"

	"github.com/quiqxiq/roslib"
	"github.com/quiqxiq/roslib-mikhmon/domain"
	"github.com/quiqxiq/roslib-mikhmon/mikrotik"
)

const queuePath = "/queue/simple"

// QueueSimpleList → /queue/simple/print (analisis §1.10).
func (c *Client) QueueSimpleList(ctx context.Context) ([]domain.QueueSimple, error) {
	reply, err := c.dev.Path(queuePath).Print().Exec(ctx)
	if err != nil {
		return nil, err
	}
	return sentencesToQueues(reply.Rows), nil
}

// QueueSimpleStatic → /queue/simple/print ?dynamic=false (analisis §1.10).
//
// Mikhmon memakai filter ini untuk dropdown parent-queue di profile
// (hanya queue yang user-defined, bukan auto-generated).
func (c *Client) QueueSimpleStatic(ctx context.Context) ([]domain.QueueSimple, error) {
	reply, err := c.dev.Path(queuePath).Print().Where("dynamic", "false").Exec(ctx)
	if err != nil {
		return nil, err
	}
	return sentencesToQueues(reply.Rows), nil
}

// QueueSimpleByName → /queue/simple/print ?name=<name> (analisis §1.10).
func (c *Client) QueueSimpleByName(ctx context.Context, name string) ([]domain.QueueSimple, error) {
	if name == "" {
		return nil, mikrotik.ErrInvalidArgument
	}
	reply, err := c.dev.Path(queuePath).Print().Where("name", name).Exec(ctx)
	if err != nil {
		return nil, err
	}
	return sentencesToQueues(reply.Rows), nil
}

// QueueSimpleRemove → /queue/simple/remove (analisis §1.10).
func (c *Client) QueueSimpleRemove(ctx context.Context, id string) error {
	if id == "" {
		return mikrotik.ErrInvalidArgument
	}
	_, err := c.dev.Path(queuePath).Remove(ctx, id)
	return err
}

func sentencesToQueues(rows []*roslib.Sentence) []domain.QueueSimple {
	out := make([]domain.QueueSimple, 0, len(rows))
	for _, s := range rows {
		out = append(out, domain.QueueSimple{
			ID:         s.Get(".id"),
			Name:       s.Get("name"),
			Target:     s.Get("target"),
			MaxLimit:   s.Get("max-limit"),
			BurstLimit: s.Get("burst-limit"),
			Parent:     s.Get("parent"),
			Disabled:   s.BoolOr("disabled", false),
			Dynamic:    s.BoolOr("dynamic", false),
			Comment:    s.Get("comment"),
		})
	}
	return out
}
