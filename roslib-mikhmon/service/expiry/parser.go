package expiry

import (
	"strings"
	"time"
)

// mikhmon date format: "jan/05/2025 15:04:05"
// RouterOS month names are lowercase 3-letter english abbreviations.
const mikhmonLayout = "jan/02/2006 15:04:05"

// ParseExpiry memparse field comment format mikhmon menjadi time.Time.
// Mengembalikan (time, true) jika valid, (zero, false) jika bukan expiry comment.
func ParseExpiry(comment string) (time.Time, bool) {
	comment = strings.TrimSpace(comment)
	if len(comment) < 19 {
		return time.Time{}, false
	}
	// Format: "jan/02/2006 15:04:05" — 19 karakter minimum
	// Ambil 19 karakter pertama untuk parse
	t, err := time.Parse(mikhmonLayout, comment[:19])
	if err != nil {
		return time.Time{}, false
	}
	return t, true
}
