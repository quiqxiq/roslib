package syslog

import (
	"context"

	"github.com/quiqxiq/roslib-mikhmon/domain"
	"github.com/quiqxiq/roslib-mikhmon/mikrotik"
)

// LogList → /log/print (analisis §1.5). Kembalikan semua entry log
// (caller bertanggung jawab batasi via filter).
func (c *Client) LogList(ctx context.Context) ([]domain.LogEntry, error) {
	reply, err := c.dev.Path("/log").Print().Exec(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]domain.LogEntry, 0, len(reply.Rows))
	for _, s := range reply.Rows {
		out = append(out, domain.LogEntry{
			ID:      s.Get(".id"),
			Time:    s.Get("time"),
			Topics:  s.Get("topics"),
			Message: s.Get("message"),
		})
	}
	return out, nil
}

// LogByTopics → /log/print ?topics=<topics> (analisis §1.5).
//
// Mikhmon dashboard pakai topics="hotspot,info,debug" untuk feed live log
// hotspot. Multi-topic dipisah dengan koma.
func (c *Client) LogByTopics(ctx context.Context, topics string) ([]domain.LogEntry, error) {
	if topics == "" {
		return nil, mikrotik.ErrInvalidArgument
	}
	reply, err := c.dev.Path("/log").Print().Where("topics", topics).Exec(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]domain.LogEntry, 0, len(reply.Rows))
	for _, s := range reply.Rows {
		out = append(out, domain.LogEntry{
			ID:      s.Get(".id"),
			Time:    s.Get("time"),
			Topics:  s.Get("topics"),
			Message: s.Get("message"),
		})
	}
	return out, nil
}
