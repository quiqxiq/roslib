package domain

import "fmt"

// ExpiredMode adalah perilaku yang dipasang di profil hotspot untuk
// menentukan apa yang terjadi pada user setelah comment expiry-nya
// tercapai oleh background scheduler. Mengikuti analisis §5.
type ExpiredMode string

const (
	// ModeNone tidak melakukan apa-apa saat expired (kode "0").
	ModeNone ExpiredMode = "0"
	// ModeRemove menghapus user saat expired (kode "rem").
	ModeRemove ExpiredMode = "rem"
	// ModeNotice set limit-uptime=1s saat expired (kode "ntf").
	ModeNotice ExpiredMode = "ntf"
	// ModeRemoveRecord = ModeRemove + record transaksi (kode "remc").
	ModeRemoveRecord ExpiredMode = "remc"
	// ModeNoticeRecord = ModeNotice + record transaksi (kode "ntfc").
	ModeNoticeRecord ExpiredMode = "ntfc"
)

// IsValid melaporkan apakah mode dikenal.
func (m ExpiredMode) IsValid() bool {
	switch m {
	case ModeNone, ModeRemove, ModeNotice, ModeRemoveRecord, ModeNoticeRecord:
		return true
	}
	return false
}

// RecordsTransaction true jika mode menyimpan transaksi sebagai
// /system/script (mode "remc" atau "ntfc").
func (m ExpiredMode) RecordsTransaction() bool {
	return m == ModeRemoveRecord || m == ModeNoticeRecord
}

// HasExpiry true jika mode mengaktifkan kalkulasi expiry (selain ModeNone).
func (m ExpiredMode) HasExpiry() bool { return m != ModeNone }

// ParseExpiredMode mengkonversi string ke ExpiredMode dan validasi.
func ParseExpiredMode(s string) (ExpiredMode, error) {
	m := ExpiredMode(s)
	if !m.IsValid() {
		return "", fmt.Errorf("domain: unknown expired mode %q", s)
	}
	return m, nil
}

// Charset adalah kategori karakter pembentuk voucher username/password.
// Mengikuti analisis §7 (kolom user mode di Quick Print).
type Charset string

const (
	CharsetLower  Charset = "lower"
	CharsetUpper  Charset = "upper"
	CharsetMixed  Charset = "mixed"
	CharsetNumber Charset = "number"
	CharsetLowNum Charset = "lower_number"
	CharsetUpNum  Charset = "upper_number"
	CharsetMixNum Charset = "mixed_number"
)

// IsValid melaporkan apakah charset dikenal.
func (c Charset) IsValid() bool {
	switch c {
	case CharsetLower, CharsetUpper, CharsetMixed,
		CharsetNumber, CharsetLowNum, CharsetUpNum, CharsetMixNum:
		return true
	}
	return false
}
