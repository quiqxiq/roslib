package network

import (
	"time"

	"github.com/quiqxiq/roslib"
)

// InterfaceTrafficStream → /interface/monitor-traffic interface=<name>
// (analisis §1.11). Inherent streaming — router emit reply berkala sampai
// /cancel. Stop dengan dev.UnregisterStream(id) atau StopStream(id).
func (c *Client) InterfaceTrafficStream(id, iface string, h func(*roslib.Sentence)) error {
	return c.dev.Path("/interface/monitor-traffic").
		With("interface", iface).
		Stream(id, h)
}

// InterfaceStatsStream → /interface/print stats interval=<d>.
// Counter byte/packet per interface, update tiap interval.
func (c *Client) InterfaceStatsStream(id string, interval time.Duration, h func(*roslib.Sentence)) error {
	return c.dev.Path("/interface").Print().Stats().
		Interval(interval).Stream(id, h)
}

// QueueStatsStream → /queue/simple/print stats interval=<d> (analisis §1.10).
// Counter per queue (bytes, packets, rate).
func (c *Client) QueueStatsStream(id string, interval time.Duration, h func(*roslib.Sentence)) error {
	return c.dev.Path("/queue/simple").Print().Stats().
		Interval(interval).Stream(id, h)
}

// StopStream menghentikan listener dengan ID tersebut.
// Return true bila listener ada dan dihapus.
func (c *Client) StopStream(id string) bool {
	return c.dev.UnregisterStream(id)
}
