package syslog

import "github.com/quiqxiq/roslib"

// LogStream → /log/print follow (analisis §1.5). FollowOnly variant —
// emit hanya event log baru, tanpa snapshot historikal awal.
// topics opsional (mis. "hotspot,info,debug"); kosong = semua topics.
//
// Stop dengan StopLogStream(id) atau dev.UnregisterStream(id).
func (c *Client) LogStream(id, topics string, h func(*roslib.Sentence)) error {
	pb := c.dev.Path("/log").Print()
	if topics != "" {
		pb = pb.Where("topics", topics)
	}
	return pb.FollowOnly().Stream(id, h)
}

// LogStreamFollow varian dengan snapshot awal lalu emit delta.
// Cocok kalau caller butuh state log historikal dulu sebelum live.
func (c *Client) LogStreamFollow(id, topics string, h func(*roslib.Sentence)) error {
	pb := c.dev.Path("/log").Print()
	if topics != "" {
		pb = pb.Where("topics", topics)
	}
	return pb.Follow().Stream(id, h)
}

// StopLogStream menghentikan listener dengan ID tersebut.
// Return true bila listener ada dan dihapus.
func (c *Client) StopLogStream(id string) bool {
	return c.dev.UnregisterStream(id)
}
