package rosfmt

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Months adalah daftar bulan format pendek RouterOS.
var Months = []string{"jan", "feb", "mar", "apr", "may", "jun", "jul", "aug", "sep", "oct", "nov", "dec"}

// MonthIndex melaporkan index 0..11 untuk nama bulan pendek (case-insensitive).
// Mengembalikan -1 kalau tidak dikenal.
func MonthIndex(name string) int {
	n := strings.ToLower(strings.TrimSpace(name))
	for i, m := range Months {
		if m == n {
			return i
		}
	}
	return -1
}

// DateInt mengubah string tanggal RouterOS "jan/02/2006" menjadi
// integer YYYYMMDD untuk perbandingan numerik (mengikuti algoritma
// `:dateint` di on-event scheduler script — analisis §3.2).
//
// Mengembalikan error jika format tidak dikenal.
func DateInt(date string) (int, error) {
	date = strings.TrimSpace(date)
	if len(date) < 11 {
		return 0, fmt.Errorf("rosfmt: date too short %q", date)
	}
	mon := date[0:3]
	day := date[4:6]
	year := date[7:11]

	mi := MonthIndex(mon)
	if mi < 0 {
		return 0, fmt.Errorf("rosfmt: unknown month %q", mon)
	}
	dayN, err := strconv.Atoi(day)
	if err != nil {
		return 0, fmt.Errorf("rosfmt: bad day %q: %w", day, err)
	}
	yearN, err := strconv.Atoi(year)
	if err != nil {
		return 0, fmt.Errorf("rosfmt: bad year %q: %w", year, err)
	}
	return yearN*10000 + (mi+1)*100 + dayN, nil
}

// TimeMinutes mengkonversi "HH:MM:SS" ke total menit (detik diabaikan,
// mengikuti algoritma `:timeint` di on-event script).
func TimeMinutes(clock string) (int, error) {
	clock = strings.TrimSpace(clock)
	parts := strings.Split(clock, ":")
	if len(parts) < 2 {
		return 0, fmt.Errorf("rosfmt: bad clock %q", clock)
	}
	h, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, fmt.Errorf("rosfmt: bad hour %q: %w", parts[0], err)
	}
	m, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, fmt.Errorf("rosfmt: bad minute %q: %w", parts[1], err)
	}
	return h*60 + m, nil
}

// FormatDate mengubah time.Time menjadi format RouterOS "jan/02/2006"
// (lower-case bulan).
func FormatDate(t time.Time) string {
	return strings.ToLower(t.Format("Jan/02/2006"))
}

// ParseDate kebalikan dari FormatDate. Lokasi dipakai sebagai fallback
// saat parsing (default: time.Local).
func ParseDate(s string, loc *time.Location) (time.Time, error) {
	if loc == nil {
		loc = time.Local
	}
	t, err := time.ParseInLocation("Jan/02/2006", strings.ToLower(strings.TrimSpace(s)), loc)
	if err != nil {
		return time.Time{}, err
	}
	return t, nil
}

// CurrentMonthOwner mengembalikan string "<bulan><tahun>" mengikuti
// konvensi Mikhmon (mis. `Jan2025`) untuk field owner /system/script.
// Bulan = 3 huruf TitleCase mengikuti format Go default.
func CurrentMonthOwner(t time.Time) string {
	return t.Format("Jan2006")
}
