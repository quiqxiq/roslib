package hotspot

import "github.com/quiqxiq/roslib"

// ActiveStream → /ip/hotspot/active/print follow (analisis §1.8).
// Snapshot awal + emit delta (table watcher). Caller hentikan dengan
// StopActiveStream(id) atau dev.UnregisterStream(id).
func (c *Client) ActiveStream(id string, h func(*roslib.Sentence)) error {
	return c.dev.Path(activePath).Print().Follow().Stream(id, h)
}

// ActiveStreamFollowOnly hanya emit event baru, tanpa snapshot awal.
// Cocok untuk audit trail (kick / login event only).
func (c *Client) ActiveStreamFollowOnly(id string, h func(*roslib.Sentence)) error {
	return c.dev.Path(activePath).Print().FollowOnly().Stream(id, h)
}

// StopActiveStream menghentikan listener dengan ID tersebut.
// Return true bila listener ada dan dihapus.
func (c *Client) StopActiveStream(id string) bool {
	return c.dev.UnregisterStream(id)
}
