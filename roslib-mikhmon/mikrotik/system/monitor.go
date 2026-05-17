package system

import (
	"time"

	"github.com/quiqxiq/roslib"
)

// MonitorResource → poll /system/resource/print tiap interval (analisis §1.1).
// Cocok untuk dashboard CPU/RAM/uptime feed.
func (c *Client) MonitorResource(id string, interval time.Duration, h func(*roslib.Sentence)) error {
	return c.dev.RegisterPoll(roslib.PollConfig{
		ID:       id,
		Path:     "/system/resource",
		Args:     []string{"print"},
		Interval: interval,
		Handler:  h,
	})
}

// MonitorScheduler → poll /system/scheduler/print ?name=<name> tiap interval
// (analisis §1.4). Dipakai untuk mendeteksi expiry user — mikhmon membaca
// field next-run dari handler.
func (c *Client) MonitorScheduler(id, name string, interval time.Duration, h func(*roslib.Sentence)) error {
	return c.dev.RegisterPoll(roslib.PollConfig{
		ID:       id,
		Path:     schedulerPath,
		Args:     []string{"print"},
		Where:    []roslib.WherePair{roslib.Where("name", name)},
		Interval: interval,
		Handler:  h,
	})
}

// StopMonitor menghentikan poll dengan ID tersebut.
// Return true bila poll ada dan dihapus.
func (c *Client) StopMonitor(id string) bool {
	return c.dev.UnregisterPoll(id)
}
