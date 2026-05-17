package ppp

import "github.com/quiqxiq/roslib"

// ActiveStream → /ppp/active/print follow (analisis §1.12).
// Snapshot awal + emit delta. Caller hentikan dengan StopActiveStream(id).
func (c *Client) ActiveStream(id string, h func(*roslib.Sentence)) error {
	return c.dev.Path(activePath).Print().Follow().Stream(id, h)
}

// ActiveStreamFollowOnly hanya emit event baru.
func (c *Client) ActiveStreamFollowOnly(id string, h func(*roslib.Sentence)) error {
	return c.dev.Path(activePath).Print().FollowOnly().Stream(id, h)
}

// StopActiveStream menghentikan listener dengan ID tersebut.
// Return true bila listener ada dan dihapus.
func (c *Client) StopActiveStream(id string) bool {
	return c.dev.UnregisterStream(id)
}
