package system

import (
	"context"

	"github.com/quiqxiq/roslib"
	"github.com/quiqxiq/roslib-mikhmon/domain"
	"github.com/quiqxiq/roslib-mikhmon/mikrotik"
)

const schedulerPath = "/system/scheduler"

// SchedulerList → /system/scheduler/print (analisis §1.4).
func (c *Client) SchedulerList(ctx context.Context) ([]domain.Scheduler, error) {
	reply, err := c.dev.Path(schedulerPath).Print().Exec(ctx)
	if err != nil {
		return nil, err
	}
	return sentencesToSchedulers(reply.Rows), nil
}

// SchedulerCount → /system/scheduler/print count-only="" (analisis §1.4).
func (c *Client) SchedulerCount(ctx context.Context) (int, error) {
	reply, err := c.dev.Path(schedulerPath).Print().Count().Exec(ctx)
	if err != nil {
		return 0, err
	}
	return mikrotik.AtoiOr(reply.Raw.Done.Map["ret"], 0), nil
}

// SchedulerByName → /system/scheduler/print ?name=<name> (analisis §1.4).
//
// Mengembalikan list karena nama scheduler tidak dijamin unik di
// RouterOS — mikhmon mengandalkan konvensi (1 scheduler per profile).
func (c *Client) SchedulerByName(ctx context.Context, name string) ([]domain.Scheduler, error) {
	if name == "" {
		return nil, mikrotik.ErrInvalidArgument
	}
	reply, err := c.dev.Path(schedulerPath).Print().Where("name", name).Exec(ctx)
	if err != nil {
		return nil, err
	}
	return sentencesToSchedulers(reply.Rows), nil
}

// SchedulerAddArgs adalah parameter SchedulerAdd. Field kosong → tidak dikirim.
type SchedulerAddArgs struct {
	Name      string // wajib
	OnEvent   string // RouterOS script body
	StartDate string // mis. "jan/01/1970"
	StartTime string // mis. "00:02:30"
	Interval  string // mis. "00:02:00"
	Disabled  *bool  // nil = pakai default RouterOS
	Comment   string
}

// SchedulerAdd → /system/scheduler/add (analisis §1.4).
func (c *Client) SchedulerAdd(ctx context.Context, a SchedulerAddArgs) (string, error) {
	if a.Name == "" {
		return "", mikrotik.ErrInvalidArgument
	}
	pairs := []roslib.Pair{roslib.NewPair("name", a.Name)}
	if a.OnEvent != "" {
		pairs = append(pairs, roslib.NewPair("on-event", a.OnEvent))
	}
	if a.StartDate != "" {
		pairs = append(pairs, roslib.NewPair("start-date", a.StartDate))
	}
	if a.StartTime != "" {
		pairs = append(pairs, roslib.NewPair("start-time", a.StartTime))
	}
	if a.Interval != "" {
		pairs = append(pairs, roslib.NewPair("interval", a.Interval))
	}
	if a.Disabled != nil {
		pairs = append(pairs, roslib.NewPair("disabled", mikrotik.BoolWord(*a.Disabled)))
	}
	if a.Comment != "" {
		pairs = append(pairs, roslib.NewPair("comment", a.Comment))
	}
	reply, err := c.dev.Path(schedulerPath).Add(ctx, pairs...)
	if err != nil {
		return "", err
	}
	if reply.Done == nil {
		return "", nil
	}
	return reply.Done.Map["ret"], nil
}

// SchedulerSetArgs adalah parameter SchedulerSet.
type SchedulerSetArgs struct {
	ID       string // wajib
	Name     string
	OnEvent  string
	Interval string
	Disabled *bool
	Comment  string
}

// SchedulerSet → /system/scheduler/set (analisis §1.4).
func (c *Client) SchedulerSet(ctx context.Context, a SchedulerSetArgs) error {
	if a.ID == "" {
		return mikrotik.ErrInvalidArgument
	}
	pairs := []roslib.Pair{}
	if a.Name != "" {
		pairs = append(pairs, roslib.NewPair("name", a.Name))
	}
	if a.OnEvent != "" {
		pairs = append(pairs, roslib.NewPair("on-event", a.OnEvent))
	}
	if a.Interval != "" {
		pairs = append(pairs, roslib.NewPair("interval", a.Interval))
	}
	if a.Disabled != nil {
		pairs = append(pairs, roslib.NewPair("disabled", mikrotik.BoolWord(*a.Disabled)))
	}
	if a.Comment != "" {
		pairs = append(pairs, roslib.NewPair("comment", a.Comment))
	}
	_, err := c.dev.Path(schedulerPath).Set(ctx, a.ID, pairs...)
	return err
}

// SchedulerRemove → /system/scheduler/remove (analisis §1.4).
func (c *Client) SchedulerRemove(ctx context.Context, id string) error {
	if id == "" {
		return mikrotik.ErrInvalidArgument
	}
	_, err := c.dev.Path(schedulerPath).Remove(ctx, id)
	return err
}

// ───────────── helpers ─────────────

func sentenceToScheduler(s *roslib.Sentence) domain.Scheduler {
	return domain.Scheduler{
		ID:        s.Get(".id"),
		Name:      s.Get("name"),
		StartDate: s.Get("start-date"),
		StartTime: s.Get("start-time"),
		Interval:  s.Get("interval"),
		OnEvent:   s.Get("on-event"),
		NextRun:   s.Get("next-run"),
		Disabled:  s.BoolOr("disabled", false),
		Comment:   s.Get("comment"),
		RunCount:  int(s.IntOr("run-count", 0)),
	}
}

func sentencesToSchedulers(rows []*roslib.Sentence) []domain.Scheduler {
	out := make([]domain.Scheduler, 0, len(rows))
	for _, r := range rows {
		out = append(out, sentenceToScheduler(r))
	}
	return out
}
