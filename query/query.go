// Package query mendefinisikan tipe-tipe shared untuk membangun
// RouterOS API sentence: pasangan key/value parameter dan filter where.
package query

import "strings"

// Pair adalah named parameter untuk command RouterOS.
// Saat di-encode menjadi sentence, hasilnya berformat "=Key=Value".
type Pair struct {
	Key   string
	Value string
}

// NewPair adalah helper konstruktor singkat.
func NewPair(key, value string) Pair {
	return Pair{Key: key, Value: value}
}

// Word mengembalikan representasi sentence untuk Pair ini.
func (p Pair) Word() string {
	return "=" + p.Key + "=" + p.Value
}

// WhereOp adalah operator perbandingan untuk filter where RouterOS.
type WhereOp string

const (
	OpEqual    WhereOp = "="
	OpNotEqual WhereOp = "-"
	OpGreater  WhereOp = ">"
	OpLess     WhereOp = "<"
)

// WherePair adalah filter ?key=value (atau operator lain) untuk Print.
// RouterOS mendukung beberapa operator dengan prefix khusus.
type WherePair struct {
	Key   string
	Value string
	Op    WhereOp
}

// Where membuat WherePair dengan operator =.
func Where(key, value string) WherePair {
	return WherePair{Key: key, Value: value, Op: OpEqual}
}

// WhereNot membuat WherePair dengan operator !=.
func WhereNot(key, value string) WherePair {
	return WherePair{Key: key, Value: value, Op: OpNotEqual}
}

// Word mengembalikan representasi sentence untuk WherePair.
//
// Format RouterOS:
//
//	?=key=value   → exact match
//	?-key=value   → not equal (sebenarnya dipakai jarang; kebanyakan
//	                user pakai operator ! di posisi terpisah, tapi prefix
//	                di sini menjaga API tetap deklaratif).
//	?>key=value, ?<key=value untuk komparasi numerik.
func (w WherePair) Word() string {
	var b strings.Builder
	b.WriteString("?")
	if w.Op != OpEqual {
		b.WriteString(string(w.Op))
	}
	b.WriteString(w.Key)
	b.WriteString("=")
	b.WriteString(w.Value)
	return b.String()
}

// BuildSentence menggabungkan command (e.g. "/ip/address/print") dengan
// daftar flag (kata bebas), Pair (=k=v), dan WherePair (?k=v) menjadi
// urutan kata sentence yang siap dikirim ke routeros.Client.RunArgsContext.
func BuildSentence(command string, flags []string, pairs []Pair, wheres []WherePair) []string {
	out := make([]string, 0, 1+len(flags)+len(pairs)+len(wheres))
	out = append(out, command)
	for _, f := range flags {
		if f != "" {
			out = append(out, f)
		}
	}
	for _, p := range pairs {
		out = append(out, p.Word())
	}
	for _, w := range wheres {
		out = append(out, w.Word())
	}
	return out
}
