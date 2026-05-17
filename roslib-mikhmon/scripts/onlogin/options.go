package onlogin

import "github.com/quiqxiq/roslib-mikhmon/domain"

// Options mendeskripsikan parameter generate body script on-login.
//
// Cross-ref: analisis §3.1.
type Options struct {
	// Mode menentukan kelakuan script (analisis §5).
	Mode domain.ExpiredMode

	// Validity adalah durasi paket dalam format RouterOS, mis. "30d", "1h".
	// Diabaikan saat Mode = ModeNone.
	Validity string

	// Price (nilai numerik raw) — di-embed ke metadata `:put`.
	Price int

	// SellPrice (sprice) — harga jual ke end user.
	SellPrice int

	// LockMAC menambah blok lock user ke MAC saat login pertama.
	LockMAC bool
}

// metadataLockToken mengembalikan kata "Enable" / "Disable" yang diharapkan
// PHP saat parse metadata `:put`.
func (o Options) metadataLockToken() string {
	if o.LockMAC {
		return "Enable"
	}
	return "Disable"
}
