package workflows

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

// BulkSeparator adalah separator multi-ID di URL mikhmonv3 (analisis §4.1
// "Bulk delete (removehotspotusers) menggunakan separator `~`").
const BulkSeparator = "~"

// ParseBulkIDs men-parse input "id1~id2~id3" menjadi slice. Whitespace
// di-trim, entry kosong di-skip.
func ParseBulkIDs(raw string) []string {
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, BulkSeparator)
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

// BulkDeleteUsersErr menggabungkan kegagalan per-ID supaya caller dapat
// melaporkan parsial.
type BulkDeleteUsersErr struct {
	Failed map[string]error
}

func (e *BulkDeleteUsersErr) Error() string {
	parts := make([]string, 0, len(e.Failed))
	for id, err := range e.Failed {
		parts = append(parts, fmt.Sprintf("%s: %v", id, err))
	}
	return "workflows.BulkDeleteUsers: " + strings.Join(parts, "; ")
}

// BulkDeleteUsers loop DeleteUser untuk tiap ID. Tidak fail-fast —
// continue ke ID berikutnya kalau satu gagal, lalu return aggregated
// error.
func BulkDeleteUsers(ctx context.Context, c *Clients, ids []string) error {
	failed := map[string]error{}
	for _, id := range ids {
		if err := DeleteUser(ctx, c, id); err != nil {
			failed[id] = err
		}
	}
	if len(failed) == 0 {
		return nil
	}
	return &BulkDeleteUsersErr{Failed: failed}
}

// BulkDeleteUsersFromString shortcut: ParseBulkIDs + BulkDeleteUsers.
func BulkDeleteUsersFromString(ctx context.Context, c *Clients, raw string) error {
	ids := ParseBulkIDs(raw)
	if len(ids) == 0 {
		return errors.New("workflows.BulkDeleteUsers: no ids")
	}
	return BulkDeleteUsers(ctx, c, ids)
}
