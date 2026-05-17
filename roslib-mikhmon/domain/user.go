package domain

// HotspotUser adalah proyeksi tipe-aman dari row /ip/hotspot/user/print.
// Field opsional di-omit pakai pointer / zero-value (RouterOS sering tidak
// kirim field jika tidak relevan).
//
// Cross-ref: analisis §1.6.
type HotspotUser struct {
	ID              string // .id di RouterOS
	Name            string
	Password        string
	Profile         string
	Server          string // "all" untuk semua server hotspot
	Disabled        bool
	Comment         string // sering dipakai mikhmon untuk simpan expiry date
	MACAddress      string
	LimitUptime     string // format RouterOS: "1h", "1d", dll
	LimitBytesTotal int64  // 0 = unlimited
	LimitBytesIn    int64  // limit-bytes-in per-direction (0 = unlimited)
	LimitBytesOut   int64  // limit-bytes-out per-direction (0 = unlimited)

	// Counters runtime (read-only, di-isi saat list).
	BytesIn  int64
	BytesOut int64
	Uptime   string
}

// HotspotActive adalah row /ip/hotspot/active/print.
type HotspotActive struct {
	ID         string
	User       string
	Address    string
	MACAddress string
	Server     string
	LoginBy    string
	Uptime     string
	BytesIn    int64
	BytesOut   int64
}

// HotspotHost adalah row /ip/hotspot/host/print.
type HotspotHost struct {
	ID         string
	MACAddress string
	Address    string
	ToAddress  string
	Server     string
	Authorized bool
	Bypassed   bool
	Comment    string
}

// HotspotCookie adalah row /ip/hotspot/cookie/print.
type HotspotCookie struct {
	ID       string
	User     string
	Domain   string
	Expires  string
	MACEndor string
}

// HotspotBinding adalah row /ip/hotspot/ip-binding/print.
//
// Cross-ref: analisis §1.9.
type HotspotBinding struct {
	ID         string
	MACAddress string
	Address    string
	ToAddress  string
	Server     string
	Type       string // "regular", "bypassed", "blocked"
	Disabled   bool
	Comment    string
}
