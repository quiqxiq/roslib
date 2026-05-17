// Package syslog membungkus path /log/* (analisis §1.5).
//
// Nama paket "syslog" karena "log" bentrok dengan stdlib log.
package syslog

import "github.com/quiqxiq/roslib"

// Client adalah handle untuk command /log/*.
type Client struct {
	dev *roslib.Device
}

// New membuat Client baru.
func New(dev *roslib.Device) *Client { return &Client{dev: dev} }
