package cache

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

// KeyOf membuat cache key kanonik dari sentence RouterOS.
//
// Bentuk: "roslib:<sha256-hex>". Hash dipakai supaya key tidak panjang
// dan tidak mengandung karakter yang merepotkan di backend cache.
//
// Urutan kata penting — flag dengan urutan berbeda akan menghasilkan key
// berbeda. Caller boleh menormalisasi urutan flag/pair sebelum panggil
// kalau menginginkan equivalence.
func KeyOf(sentence []string) string {
	joined := strings.Join(sentence, "\x1f")
	sum := sha256.Sum256([]byte(joined))
	return "roslib:" + hex.EncodeToString(sum[:])
}
