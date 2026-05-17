// Package network membungkus path RouterOS yang dipakai mikhmon untuk
// queue, IP pool, ARP, DHCP server lease, dan interface — semuanya dipakai
// pada cascade IP-binding (analisis §1.10, §1.11, §4.2).
package network

import "github.com/quiqxiq/roslib"

// Client adalah handle untuk semua command network-level.
type Client struct {
	dev *roslib.Device
}

// New membuat Client baru.
func New(dev *roslib.Device) *Client { return &Client{dev: dev} }
