package rosfmt

import (
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode"
)

// ParseDuration parse format durasi RouterOS ("1w2d3h4m5s", "30d", "1h30m")
// menjadi time.Duration. Tanpa unit dianggap detik.
func ParseDuration(v string) (time.Duration, error) {
	v = strings.TrimSpace(v)
	if v == "" {
		return 0, fmt.Errorf("rosfmt: empty duration")
	}
	if _, err := strconv.ParseFloat(v, 64); err == nil {
		secs, _ := strconv.ParseFloat(v, 64)
		return time.Duration(secs * float64(time.Second)), nil
	}
	var total time.Duration
	i := 0
	for i < len(v) {
		j := i
		for j < len(v) && (unicode.IsDigit(rune(v[j])) || v[j] == '.') {
			j++
		}
		if j == i {
			return 0, fmt.Errorf("rosfmt: bad duration %q", v)
		}
		num, err := strconv.ParseFloat(v[i:j], 64)
		if err != nil {
			return 0, fmt.Errorf("rosfmt: bad number %q in duration", v[i:j])
		}
		k := j
		for k < len(v) && unicode.IsLetter(rune(v[k])) {
			k++
		}
		unit := v[j:k]
		mul, ok := durationUnit(unit)
		if !ok {
			return 0, fmt.Errorf("rosfmt: unknown unit %q in duration %q", unit, v)
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
	}
	return 0, false
}

// IsValidDuration cek apakah string bisa di-parse oleh ParseDuration.
func IsValidDuration(v string) bool {
	_, err := ParseDuration(v)
	return err == nil
}
