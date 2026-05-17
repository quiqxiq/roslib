package tcpmock

import (
	"bufio"
	"errors"
	"fmt"
	"io"
)

// EncodeWord meng-encode satu word RouterOS API ke writer mengikuti aturan
// length-prefix variabel:
//
//	len <  0x80         → 1 byte
//	len <  0x4000       → 2 byte (high bit 0x80 di byte pertama)
//	len <  0x200000     → 3 byte (high bit 0xC0)
//	len <  0x10000000   → 4 byte (high bit 0xE0)
//	len >= 0x10000000   → 5 byte (0xF0 + length 32-bit big-endian)
func EncodeWord(w io.Writer, word string) error {
	n := len(word)
	var buf []byte
	switch {
	case n < 0x80:
		buf = []byte{byte(n)}
	case n < 0x4000:
		buf = []byte{byte(n>>8) | 0x80, byte(n)}
	case n < 0x200000:
		buf = []byte{byte(n>>16) | 0xC0, byte(n >> 8), byte(n)}
	case n < 0x10000000:
		buf = []byte{byte(n>>24) | 0xE0, byte(n >> 16), byte(n >> 8), byte(n)}
	default:
		buf = []byte{0xF0, byte(n >> 24), byte(n >> 16), byte(n >> 8), byte(n)}
	}
	if _, err := w.Write(buf); err != nil {
		return err
	}
	if n > 0 {
		if _, err := w.Write([]byte(word)); err != nil {
			return err
		}
	}
	return nil
}

// EncodeSentence meng-encode urutan word + word kosong sebagai terminator.
func EncodeSentence(w io.Writer, words []string) error {
	for _, word := range words {
		if err := EncodeWord(w, word); err != nil {
			return err
		}
	}
	return EncodeWord(w, "")
}

// DecodeWord membaca satu word sesuai aturan length-prefix.
// Mengembalikan io.EOF kalau koneksi tutup.
func DecodeWord(r *bufio.Reader) (string, error) {
	first, err := r.ReadByte()
	if err != nil {
		return "", err
	}
	var length int
	switch {
	case first&0x80 == 0:
		length = int(first)
	case first&0xC0 == 0x80:
		b1, err := r.ReadByte()
		if err != nil {
			return "", err
		}
		length = (int(first&0x3F) << 8) | int(b1)
	case first&0xE0 == 0xC0:
		b := make([]byte, 2)
		if _, err := io.ReadFull(r, b); err != nil {
			return "", err
		}
		length = (int(first&0x1F) << 16) | (int(b[0]) << 8) | int(b[1])
	case first&0xF0 == 0xE0:
		b := make([]byte, 3)
		if _, err := io.ReadFull(r, b); err != nil {
			return "", err
		}
		length = (int(first&0x0F) << 24) | (int(b[0]) << 16) | (int(b[1]) << 8) | int(b[2])
	case first == 0xF0:
		b := make([]byte, 4)
		if _, err := io.ReadFull(r, b); err != nil {
			return "", err
		}
		length = (int(b[0]) << 24) | (int(b[1]) << 16) | (int(b[2]) << 8) | int(b[3])
	default:
		return "", fmt.Errorf("tcpmock: unknown length prefix 0x%02x", first)
	}
	if length == 0 {
		return "", nil
	}
	buf := make([]byte, length)
	if _, err := io.ReadFull(r, buf); err != nil {
		return "", err
	}
	return string(buf), nil
}

// ErrEndOfSentence dikembalikan DecodeSentence jika terminator (length-0)
// muncul di awal — sentence kosong, koneksi tetap hidup.
var ErrEndOfSentence = errors.New("tcpmock: end of sentence")

// DecodeSentence membaca word sampai terminator (length-0). Kembalikan
// list word (tanpa terminator) atau io.EOF kalau koneksi tutup di
// pertengahan.
func DecodeSentence(r *bufio.Reader) ([]string, error) {
	var words []string
	for {
		w, err := DecodeWord(r)
		if err != nil {
			if err == io.EOF && len(words) == 0 {
				return nil, io.EOF
			}
			return words, err
		}
		if w == "" {
			return words, nil
		}
		words = append(words, w)
	}
}
