package cache

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

// KeyOf membuat cache key kanonik dari deviceID + sentence RouterOS.
//
// Bentuk: "roslib:<deviceID>:<sha256-hex>". DeviceID di-prefix supaya
// cache yang dibagi antar router (fleet) tidak konflik — sentence sama
// dari dua router berbeda akan menghasilkan key berbeda.
//
// deviceID kosong → key tanpa prefix device ("roslib::<hash>") tetap
// deterministik tapi tanpa pemisahan fleet (cocok untuk single-router).
//
// Urutan kata di sentence penting — Print().Stats().Where(x,y) vs
// Print().Where(x,y).Stats() hash berbeda. Caller yang ingin equivalence
// boleh sort flag dulu.
func KeyOf(deviceID string, sentence []string) string {
	joined := strings.Join(sentence, "\x1f")
	sum := sha256.Sum256([]byte(joined))
	return "roslib:" + deviceID + ":" + hex.EncodeToString(sum[:])
}

// PathFromSentence mengekstrak path RouterOS dari sentence yang dibuat
// query.BuildSentence dengan asumsi sentence terakhir adalah aksi (mis.
// "print", "add", "remove"). Sentence pertama berbentuk "/ip/address/print"
// → kembalikan "/ip/address".
//
// Aturan: potong segmen terakhir dari first word kalau ada minimal 2 segmen.
// Kalau hanya 1 segmen ("/x"), kembalikan apa adanya.
//
// CATATAN: untuk inherent-streaming (mis. "/interface/monitor-traffic"),
// helper ini akan strip jadi "/interface" — TIDAK cocok dipakai sebagai
// cache path untuk command tersebut. Fungsi ini hanya dipakai sebagai
// helper publik; ExecCached internal pakai PathBuilder.path langsung
// (selalu correct), bukan PathFromSentence.
func PathFromSentence(sentence []string) string {
	if len(sentence) == 0 {
		return ""
	}
	first := sentence[0]
	idx := strings.LastIndex(first, "/")
	if idx <= 0 {
		return first
	}
	return first[:idx]
}
