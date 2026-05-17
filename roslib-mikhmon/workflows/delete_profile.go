package workflows

import (
	"context"
	"errors"
	"fmt"

	"github.com/quiqxiq/roslib-mikhmon/mikrotik"
)

// DeleteProfile melakukan cascade hapus user-profile mengikuti analisis §4.3:
//
//	1. /system/scheduler/print  (?name=$profname)  → cari monitor scheduler
//	2. /ip/hotspot/user/profile/remove (.id)        → hapus profil
//	3. /system/scheduler/remove (.id)               → hapus monitor scheduler
//
// Caller mengirim profile ID + nama (untuk filter scheduler). Kalau
// nama tidak diketahui, caller bisa lookup dulu via
// hot.ProfileByID(id).Name.
func DeleteProfile(ctx context.Context, c *Clients, profileID, profileName string) error {
	if profileID == "" {
		return mikrotik.ErrInvalidArgument
	}

	var schedulerIDs []string
	if profileName != "" {
		schedulers, err := c.System.SchedulerByName(ctx, profileName)
		if err != nil {
			return fmt.Errorf("workflows.DeleteProfile: scheduler lookup: %w", err)
		}
		for _, s := range schedulers {
			schedulerIDs = append(schedulerIDs, s.ID)
		}
	}

	if rerr := c.Hotspot.ProfileRemove(ctx, profileID); rerr != nil {
		if !errors.Is(rerr, mikrotik.ErrNotFound) {
			return fmt.Errorf("workflows.DeleteProfile: remove profile: %w", rerr)
		}
	}

	for _, id := range schedulerIDs {
		if rerr := c.System.SchedulerRemove(ctx, id); rerr != nil {
			return fmt.Errorf("workflows.DeleteProfile: remove scheduler %q: %w", id, rerr)
		}
	}
	return nil
}
