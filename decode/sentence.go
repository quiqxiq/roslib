// Package decode menyediakan wrapper typed di atas proto.Sentence dari go-routeros.
//
// RouterOS API mengembalikan field sebagai map[string]string. Sentence menyimpan
// referensi ke kalimat asli dan menyediakan helper untuk mengakses field
// dengan tipe yang sesuai (bool, durasi RouterOS, ukuran byte, waktu).
package decode

import (
	"github.com/go-routeros/routeros/v3/proto"
)

// Sentence membungkus *proto.Sentence dengan helper typed accessor.
// Field Raw selalu menunjuk ke kalimat asli — bisa dipakai langsung kalau
// caller butuh akses penuh ke Word/Tag/List.
type Sentence struct {
	Raw *proto.Sentence
}

// Wrap membungkus *proto.Sentence menjadi *Sentence siap pakai.
// Jika sen nil maka return juga nil — caller wajib cek.
func Wrap(sen *proto.Sentence) *Sentence {
	if sen == nil {
		return nil
	}
	return &Sentence{Raw: sen}
}

// Get mengembalikan nilai string apa adanya untuk key tertentu.
// Jika key tidak ada, return "" — sengaja tanpa "ok" bool agar pemakaian
// di handler ringkas. Pakai Has() bila perlu deteksi ketiadaan.
func (s *Sentence) Get(key string) string {
	if s == nil || s.Raw == nil {
		return ""
	}
	return s.Raw.Map[key]
}

// Has melaporkan apakah key ada di kalimat.
func (s *Sentence) Has(key string) bool {
	if s == nil || s.Raw == nil {
		return false
	}
	_, ok := s.Raw.Map[key]
	return ok
}

// Map mengembalikan reference ke map underlying.
// Jangan dimutasi — gunakan untuk iterasi/read saja.
func (s *Sentence) Map() map[string]string {
	if s == nil || s.Raw == nil {
		return nil
	}
	return s.Raw.Map
}

// Word mengembalikan tipe sentence (!re / !done / !trap / !fatal).
func (s *Sentence) Word() string {
	if s == nil || s.Raw == nil {
		return ""
	}
	return s.Raw.Word
}

// Tag mengembalikan tag RouterOS untuk kalimat ini.
func (s *Sentence) Tag() string {
	if s == nil || s.Raw == nil {
		return ""
	}
	return s.Raw.Tag
}
