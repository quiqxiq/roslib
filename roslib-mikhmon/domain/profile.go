package domain

// HotspotProfile adalah proyeksi /ip/hotspot/user/profile/print.
//
// Cross-ref: analisis §1.7.
type HotspotProfile struct {
	ID                string
	Name              string
	AddressPool       string
	RateLimit         string // mis. "1M/2M"
	SharedUsers       int    // default 1 di RouterOS kalau kosong
	StatusAutorefresh string // mis. "1m"
	OnLogin           string // RouterOS script — di-generate oleh paket scripts/onlogin
	ParentQueue       string
}

// VoucherSpec mendeskripsikan parameter generate voucher batch.
// Tidak terikat ke RouterOS row — murni input dari user.
//
// Cross-ref: analisis §7 (Quick Print fields).
type VoucherSpec struct {
	Server      string  // "all" atau nama server hotspot
	UserMode    string  // "up" (user/pass) atau "vc" (voucher code)
	Length      int     // panjang username/password
	Prefix      string  // prefix username
	Charset     Charset // kategori karakter
	Profile     string  // nama profile yang akan dipakai
	TimeLimit   string  // "1h", "1d", dll (limit-uptime)
	DataLimit   int64   // limit-bytes-total, 0=unlimited
	Comment     string  // comment default
	Validity    string  // mis. "30d" untuk kalkulasi expiry
	Price       int     // harga normal
	SellPrice   int     // harga jual (sprice)
	LockToMAC   bool    // bind voucher ke MAC pertama yg login
	BatchSize   int     // berapa voucher dibuat
}
