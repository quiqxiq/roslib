// Package system membungkus path /system/* RouterOS yang dipakai mikhmon:
// identity, resource, routerboard, clock, reboot/shutdown, logging,
// script, scheduler, log.
//
// Cross-ref: analisis §1.1, §1.2, §1.3, §1.4, §1.5.
package system

import "github.com/quiqxiq/roslib"

// Client adalah handle untuk semua command /system/*.
type Client struct {
	dev *roslib.Device
}

// New membuat Client baru.
func New(dev *roslib.Device) *Client { return &Client{dev: dev} }
