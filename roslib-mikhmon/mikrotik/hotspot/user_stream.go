package hotspot

import (
	"time"

	"github.com/quiqxiq/roslib"
)

// UserBytesStream → /ip/hotspot/user/print bytes interval=<d>.
// Setiap tick router mengirim satu !re per user dengan field bytes-in, bytes-out.
func (c *Client) UserBytesStream(id string, interval time.Duration, h func(*roslib.Sentence)) error {
	return c.dev.Path(userPath).Print().Bytes().Interval(interval).Stream(id, h)
}

// UserPacketsStream → /ip/hotspot/user/print packets interval=<d>.
// Setiap tick router mengirim satu !re per user dengan field packets-in, packets-out.
func (c *Client) UserPacketsStream(id string, interval time.Duration, h func(*roslib.Sentence)) error {
	return c.dev.Path(userPath).Print().Packets().Interval(interval).Stream(id, h)
}

// StopUserStream menghentikan listener stream user dengan ID tersebut.
func (c *Client) StopUserStream(id string) bool {
	return c.dev.UnregisterStream(id)
}
