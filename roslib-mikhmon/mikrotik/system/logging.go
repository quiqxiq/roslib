package system

import (
	"context"

	"github.com/quiqxiq/roslib"
)

// LoggingByPrefix → /system/logging/print ?prefix=<p> (analisis §1.2).
// Mengembalikan jumlah row yang match (mikhmon hanya butuh "ada/tidak ada").
func (c *Client) LoggingByPrefix(ctx context.Context, prefix string) (int, error) {
	reply, err := c.dev.Path("/system/logging").Print().Where("prefix", prefix).Exec(ctx)
	if err != nil {
		return 0, err
	}
	return len(reply.Rows), nil
}

// LoggingAddHotspotDisk → /system/logging/add (analisis §1.2):
//
//	action=disk prefix="->" topics=hotspot,info,debug
//
// Dipakai mikhmon untuk auto-aktifkan logging hotspot ke disk saat
// dashboard dimuat.
func (c *Client) LoggingAddHotspotDisk(ctx context.Context, prefix string) error {
	if prefix == "" {
		prefix = "->"
	}
	_, err := c.dev.Path("/system/logging").Add(ctx,
		roslib.NewPair("action", "disk"),
		roslib.NewPair("prefix", prefix),
		roslib.NewPair("topics", "hotspot,info,debug"),
	)
	return err
}
