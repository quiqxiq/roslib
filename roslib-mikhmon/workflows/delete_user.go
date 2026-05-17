package workflows

import (
	"context"
	"errors"
	"fmt"

	"github.com/quiqxiq/roslib-mikhmon/mikrotik"
)

// DeleteUser melakukan cascade hapus hotspot user mengikuti analisis §4.1:
//
//	1. /ip/hotspot/user/print  (?.id=$uid)        → dapat nama
//	2. /system/script/print    (?name=$name)      → script transaksi user
//	3. /system/scheduler/print (?name=$name)      → scheduler expiry user
//	4. /system/script/remove   (.id=$scr)         → hapus tiap script
//	5. /system/scheduler/remove(.id=$sch)         → hapus tiap scheduler
//	6. /ip/hotspot/user/remove (.id=$uid)         → hapus user
//
// Kalau user sudah tidak ada (lookup return ErrNotFound), workflow
// tetap mencoba step 2-6 tanpa nama (skip filter terkait nama).
func DeleteUser(ctx context.Context, c *Clients, userID string) error {
	if userID == "" {
		return mikrotik.ErrInvalidArgument
	}

	user, err := c.Hotspot.UserByID(ctx, userID)
	switch {
	case err == nil:
		// lanjut cleanup script + scheduler by name
		if cerr := cleanupByName(ctx, c, user.Name); cerr != nil {
			return cerr
		}
	case errors.Is(err, mikrotik.ErrNotFound):
		// user sudah ke-delete duluan — tidak ada yang bisa di-cleanup
	default:
		return fmt.Errorf("workflows.DeleteUser: lookup: %w", err)
	}

	if err := c.Hotspot.UserRemove(ctx, userID); err != nil {
		if errors.Is(err, mikrotik.ErrNotFound) {
			return nil
		}
		return fmt.Errorf("workflows.DeleteUser: remove user: %w", err)
	}
	return nil
}

// cleanupByName menjalankan step 2-5: hapus semua script & scheduler
// yang nama-nya sama dengan username.
func cleanupByName(ctx context.Context, c *Clients, name string) error {
	if name == "" {
		return nil
	}
	scripts, err := c.System.ScriptByName(ctx, name)
	if err != nil {
		return fmt.Errorf("script lookup: %w", err)
	}
	schedulers, err := c.System.SchedulerByName(ctx, name)
	if err != nil {
		return fmt.Errorf("scheduler lookup: %w", err)
	}
	for _, s := range scripts {
		if rerr := c.System.ScriptRemove(ctx, s.ID); rerr != nil {
			return fmt.Errorf("script remove %q: %w", s.ID, rerr)
		}
	}
	for _, s := range schedulers {
		if rerr := c.System.SchedulerRemove(ctx, s.ID); rerr != nil {
			return fmt.Errorf("scheduler remove %q: %w", s.ID, rerr)
		}
	}
	return nil
}
