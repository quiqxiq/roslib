package system

import (
	"context"

	"github.com/quiqxiq/roslib-mikhmon/domain"
	"github.com/quiqxiq/roslib-mikhmon/mikrotik"
)

// Resource → /system/resource/print (analisis §1.1).
func (c *Client) Resource(ctx context.Context) (domain.SystemResource, error) {
	reply, err := c.dev.Path("/system/resource").Print().Exec(ctx)
	if err != nil {
		return domain.SystemResource{}, err
	}
	if len(reply.Rows) == 0 {
		return domain.SystemResource{}, mikrotik.ErrNotFound
	}
	s := reply.Rows[0]
	return domain.SystemResource{
		Uptime:           s.Get("uptime"),
		Version:          s.Get("version"),
		BoardName:        s.Get("board-name"),
		CPULoad:          int(s.IntOr("cpu-load", 0)),
		FreeMemory:       s.IntOr("free-memory", 0),
		TotalMemory:      s.IntOr("total-memory", 0),
		FreeHDDSpace:     s.IntOr("free-hdd-space", 0),
		TotalHDDSpace:    s.IntOr("total-hdd-space", 0),
		ArchitectureName: s.Get("architecture-name"),
	}, nil
}

// Routerboard → /system/routerboard/print (analisis §1.1).
func (c *Client) Routerboard(ctx context.Context) (domain.SystemRouterboard, error) {
	reply, err := c.dev.Path("/system/routerboard").Print().Exec(ctx)
	if err != nil {
		return domain.SystemRouterboard{}, err
	}
	if len(reply.Rows) == 0 {
		return domain.SystemRouterboard{}, mikrotik.ErrNotFound
	}
	s := reply.Rows[0]
	return domain.SystemRouterboard{
		Routerboard:     s.BoolOr("routerboard", false),
		Model:           s.Get("model"),
		SerialNumber:    s.Get("serial-number"),
		FirmwareType:    s.Get("firmware-type"),
		FactoryFirmware: s.Get("factory-firmware"),
		CurrentFirmware: s.Get("current-firmware"),
		UpgradeFirmware: s.Get("upgrade-firmware"),
	}, nil
}

// Clock → /system/clock/print (analisis §1.1).
func (c *Client) Clock(ctx context.Context) (domain.SystemClock, error) {
	reply, err := c.dev.Path("/system/clock").Print().Exec(ctx)
	if err != nil {
		return domain.SystemClock{}, err
	}
	if len(reply.Rows) == 0 {
		return domain.SystemClock{}, mikrotik.ErrNotFound
	}
	s := reply.Rows[0]
	return domain.SystemClock{
		Time:         s.Get("time"),
		Date:         s.Get("date"),
		TimeZoneName: s.Get("time-zone-name"),
		GMTOffset:    s.Get("gmt-offset"),
		DSTActive:    s.BoolOr("dst-active", false),
	}, nil
}
