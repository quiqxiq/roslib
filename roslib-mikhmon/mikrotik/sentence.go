package mikrotik

import "strconv"

// AtoiOr men-parse string desimal jadi int; return def kalau gagal.
// Dipakai sub-paket untuk decode field !done.ret hasil count-only="" atau
// .id setelah add.
func AtoiOr(s string, def int) int {
	if s == "" {
		return def
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return def
	}
	return n
}

// BoolWord render bool ke "yes"/"no" sesuai konvensi RouterOS API.
func BoolWord(b bool) string {
	if b {
		return "yes"
	}
	return "no"
}

// Itoa shortcut int64→string desimal.
func Itoa(n int64) string { return strconv.FormatInt(n, 10) }
