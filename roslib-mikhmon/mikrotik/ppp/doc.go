// Package ppp membungkus path /ppp/* RouterOS yang dipakai mikhmon.
//
// Catatan: file PHP-nya hilang dari repo upstream — command di-infer dari
// routing index.php dan menu.php (analisis §1.12). Hanya
// /ppp/active/remove yang ter-konfirmasi via process/removepactive.php
// (file ini ADA di repo).
package ppp

import "github.com/quiqxiq/roslib"

// Client adalah handle untuk semua command /ppp/*.
type Client struct {
	dev *roslib.Device
}

// New membuat Client baru.
func New(dev *roslib.Device) *Client { return &Client{dev: dev} }
