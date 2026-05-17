package workflows

import (
	"context"
	"errors"
	"fmt"

	"github.com/quiqxiq/roslib-mikhmon/mikrotik"
)

// KickActive memutus sesi aktif user mengikuti analisis §4.4:
//
//	1. /ip/hotspot/active/print (?.id=$uid)       → dapat nama user
//	2. /ip/hotspot/cookie/print (?user=$name)     → cari cookie milik user
//	3. /ip/hotspot/cookie/remove (.id)            → hapus cookie (tiap match)
//	4. /ip/hotspot/active/remove (.id)            → kick sesi
//
// Cookie dihapus DULU sebelum active remove supaya user tidak auto re-login
// pakai cookie yang sama.
func KickActive(ctx context.Context, c *Clients, activeID string) error {
	if activeID == "" {
		return mikrotik.ErrInvalidArgument
	}
	active, err := c.Hotspot.ActiveByID(ctx, activeID)
	if err != nil {
		if errors.Is(err, mikrotik.ErrNotFound) {
			return nil // sudah keburu logout
		}
		return fmt.Errorf("workflows.KickActive: lookup: %w", err)
	}
	if active.User != "" {
		cookies, cerr := c.Hotspot.CookieByUser(ctx, active.User)
		if cerr != nil && !errors.Is(cerr, mikrotik.ErrNotFound) {
			return fmt.Errorf("workflows.KickActive: cookie lookup: %w", cerr)
		}
		for _, ck := range cookies {
			if rerr := c.Hotspot.CookieRemove(ctx, ck.ID); rerr != nil {
				return fmt.Errorf("workflows.KickActive: cookie remove %q: %w", ck.ID, rerr)
			}
		}
	}
	if rerr := c.Hotspot.ActiveRemove(ctx, activeID); rerr != nil {
		return fmt.Errorf("workflows.KickActive: active remove: %w", rerr)
	}
	return nil
}
