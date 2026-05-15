package decode

import (
	"errors"
	"strconv"
	"strings"
	"time"
	"unicode"
)

// ErrEmpty dikembalikan helper typed accessor bila value untuk key kosong/tidak ada.
var ErrEmpty = errors.New("decode: empty value")

// Bool menerjemahkan value RouterOS ("true"/"false"/"yes"/"no") ke bool.
// Default return false + error kalau bukan salah satu bentuk di atas.
func (s *Sentence) Bool(key string) (bool, error) {
	v := strings.TrimSpace(s.Get(key))
	if v == "" {
		return false, ErrEmpty
	}
	switch strings.ToLower(v) {
	case "true", "yes":
		return true, nil
	case "false", "no":
		return false, nil
	}
	return false, &valueError{key: key, raw: v, kind: "bool"}
}

// BoolOr mengembalikan Bool(key), atau def jika error.
func (s *Sentence) BoolOr(key string, def bool) bool {
	v, err := s.Bool(key)
	if err != nil {
		return def
	}
	return v
}

// Int menerjemahkan value RouterOS ke int64. RouterOS sering mengirim angka
// sebagai string desimal, tapi beberapa field punya satuan (mis "1024K") —
// untuk yang bersatuan gunakan Bytes() / Duration().
func (s *Sentence) Int(key string) (int64, error) {
	v := strings.TrimSpace(s.Get(key))
	if v == "" {
		return 0, ErrEmpty
	}
	n, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return 0, &valueError{key: key, raw: v, kind: "int"}
	}
	return n, nil
}

// IntOr mengembalikan Int(key), atau def jika error.
func (s *Sentence) IntOr(key string, def int64) int64 {
	v, err := s.Int(key)
	if err != nil {
		return def
	}
	return v
}

// Float menerjemahkan value RouterOS ke float64.
func (s *Sentence) Float(key string) (float64, error) {
	v := strings.TrimSpace(s.Get(key))
	if v == "" {
		return 0, ErrEmpty
	}
	f, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return 0, &valueError{key: key, raw: v, kind: "float"}
	}
	return f, nil
}

// Duration menerjemahkan format durasi RouterOS ke time.Duration.
// RouterOS memakai format seperti "1w2d3h4m5s" atau "00:01:23.456".
//
// Yang didukung:
//   - Satuan: w (week), d (day), h, m, s, ms, us, ns. Tanpa unit dianggap detik.
//   - Format HH:MM:SS[.fraction].
//   - String kosong → ErrEmpty.
func (s *Sentence) Duration(key string) (time.Duration, error) {
	v := strings.TrimSpace(s.Get(key))
	if v == "" {
		return 0, ErrEmpty
	}

	if strings.Contains(v, ":") {
		return parseClockDuration(v)
	}
	return parseUnitDuration(v)
}

// DurationOr mengembalikan Duration(key), atau def jika error.
func (s *Sentence) DurationOr(key string, def time.Duration) time.Duration {
	d, err := s.Duration(key)
	if err != nil {
		return def
	}
	return d
}

// Bytes menerjemahkan field ukuran (e.g. "1024", "1K", "2M", "3G") ke int64.
// RouterOS umumnya pakai SI sederhana: K=1024, M=1024*1024, G=1024^3.
func (s *Sentence) Bytes(key string) (int64, error) {
	v := strings.TrimSpace(s.Get(key))
	if v == "" {
		return 0, ErrEmpty
	}

	// Pisah numeric prefix dan suffix.
	i := 0
	for i < len(v) && (unicode.IsDigit(rune(v[i])) || v[i] == '.' || v[i] == '-') {
		i++
	}
	numPart, unitPart := v[:i], strings.ToUpper(strings.TrimSpace(v[i:]))

	n, err := strconv.ParseFloat(numPart, 64)
	if err != nil {
		return 0, &valueError{key: key, raw: v, kind: "bytes"}
	}

	mult := int64(1)
	switch unitPart {
	case "", "B":
		mult = 1
	case "K", "KB", "KIB":
		mult = 1 << 10
	case "M", "MB", "MIB":
		mult = 1 << 20
	case "G", "GB", "GIB":
		mult = 1 << 30
	case "T", "TB", "TIB":
		mult = 1 << 40
	default:
		return 0, &valueError{key: key, raw: v, kind: "bytes"}
	}
	return int64(n * float64(mult)), nil
}

// BytesOr mengembalikan Bytes(key), atau def jika error.
func (s *Sentence) BytesOr(key string, def int64) int64 {
	v, err := s.Bytes(key)
	if err != nil {
		return def
	}
	return v
}

// Time menerjemahkan field tanggal RouterOS ke time.Time.
// RouterOS biasanya mengirim "Jan/02/2006 15:04:05" atau ISO 8601.
func (s *Sentence) Time(key string) (time.Time, error) {
	v := strings.TrimSpace(s.Get(key))
	if v == "" {
		return time.Time{}, ErrEmpty
	}
	formats := []string{
		"Jan/02/2006 15:04:05",
		"2006-01-02 15:04:05",
		time.RFC3339,
	}
	for _, f := range formats {
		if t, err := time.Parse(f, v); err == nil {
			return t, nil
		}
	}
	return time.Time{}, &valueError{key: key, raw: v, kind: "time"}
}

// ──────────────── internal helpers ────────────────

type valueError struct {
	key  string
	raw  string
	kind string
}

func (e *valueError) Error() string {
	return "decode " + e.kind + " for key=" + e.key + ": invalid value " + strconv.Quote(e.raw)
}

func parseClockDuration(v string) (time.Duration, error) {
	parts := strings.Split(v, ":")
	if len(parts) < 2 || len(parts) > 3 {
		return 0, &valueError{raw: v, kind: "duration"}
	}
	var total time.Duration
	multipliers := []time.Duration{time.Hour, time.Minute, time.Second}
	offset := 3 - len(parts)
	for i, p := range parts {
		mul := multipliers[i+offset]
		if mul == time.Second {
			// Bisa ada fraksi.
			f, err := strconv.ParseFloat(p, 64)
			if err != nil {
				return 0, &valueError{raw: v, kind: "duration"}
			}
			total += time.Duration(f * float64(time.Second))
			continue
		}
		n, err := strconv.Atoi(p)
		if err != nil {
			return 0, &valueError{raw: v, kind: "duration"}
		}
		total += time.Duration(n) * mul
	}
	return total, nil
}

func parseUnitDuration(v string) (time.Duration, error) {
	// Format: kombinasi <angka><unit>... contoh "1w2d3h4m5s".
	// Tanpa unit sama sekali → asumsi detik.
	if _, err := strconv.ParseFloat(v, 64); err == nil {
		secs, _ := strconv.ParseFloat(v, 64)
		return time.Duration(secs * float64(time.Second)), nil
	}

	var total time.Duration
	i := 0
	for i < len(v) {
		// Baca angka.
		j := i
		for j < len(v) && (unicode.IsDigit(rune(v[j])) || v[j] == '.') {
			j++
		}
		if j == i {
			return 0, &valueError{raw: v, kind: "duration"}
		}
		num, err := strconv.ParseFloat(v[i:j], 64)
		if err != nil {
			return 0, &valueError{raw: v, kind: "duration"}
		}
		// Baca unit.
		k := j
		for k < len(v) && unicode.IsLetter(rune(v[k])) {
			k++
		}
		unit := v[j:k]

		mul, ok := durationUnit(unit)
		if !ok {
			return 0, &valueError{raw: v, kind: "duration"}
		}
		total += time.Duration(num * float64(mul))
		i = k
	}
	return total, nil
}

func durationUnit(unit string) (time.Duration, bool) {
	switch unit {
	case "w":
		return 7 * 24 * time.Hour, true
	case "d":
		return 24 * time.Hour, true
	case "h":
		return time.Hour, true
	case "m":
		return time.Minute, true
	case "s":
		return time.Second, true
	case "ms":
		return time.Millisecond, true
	case "us", "µs":
		return time.Microsecond, true
	case "ns":
		return time.Nanosecond, true
	}
	return 0, false
}
