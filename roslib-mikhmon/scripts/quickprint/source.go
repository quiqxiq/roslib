package quickprint

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/quiqxiq/roslib-mikhmon/domain"
)

// Config adalah representasi typed dari source QuickPrint.
type Config struct {
	Name      string
	Server    string
	UserMode  string // "up" | "vc"
	Length    int
	Prefix    string
	Charset   domain.Charset
	Profile   string
	TimeLimit string
	DataLimit int64
	Comment   string
	Validity  string
	Price     int
	SellPrice int
	LockMAC   bool // "Enable" / "Disable" di source
}

// ErrFieldCount dikembalikan Parse jika source tidak punya 13 field.
var ErrFieldCount = errors.New("quickprint: source does not have 13 fields")

// Format menyusun string source mengikuti analisis §7.
//
// Diawali dengan separator `#` (index 0 = string kosong) seperti format
// asli mikhmon, supaya backward-compatible.
func Format(c Config) string {
	priceField := fmt.Sprintf("%d_%d", c.Price, c.SellPrice)
	parts := []string{
		"", // index 0 (analisis §7 1-based)
		c.Name,
		c.Server,
		c.UserMode,
		strconv.Itoa(c.Length),
		c.Prefix,
		string(c.Charset),
		c.Profile,
		c.TimeLimit,
		strconv.FormatInt(c.DataLimit, 10),
		c.Comment,
		c.Validity,
		priceField,
		lockToken(c.LockMAC),
	}
	return strings.Join(parts, "#")
}

// Parse mengkonversi source string ke Config. Tolerant terhadap value
// numeric yang tidak valid (di-set 0).
func Parse(src string) (Config, error) {
	parts := strings.Split(src, "#")
	if len(parts) != 14 {
		return Config{}, ErrFieldCount
	}
	c := Config{
		Name:      parts[1],
		Server:    parts[2],
		UserMode:  parts[3],
		Length:    atoiOr(parts[4], 0),
		Prefix:    parts[5],
		Charset:   domain.Charset(parts[6]),
		Profile:   parts[7],
		TimeLimit: parts[8],
		DataLimit: atoi64Or(parts[9], 0),
		Comment:   parts[10],
		Validity:  parts[11],
	}
	if pp := strings.SplitN(parts[12], "_", 2); len(pp) == 2 {
		c.Price = atoiOr(pp[0], 0)
		c.SellPrice = atoiOr(pp[1], 0)
	} else {
		c.Price = atoiOr(parts[12], 0)
	}
	c.LockMAC = parts[13] == "Enable"
	return c, nil
}

func lockToken(b bool) string {
	if b {
		return "Enable"
	}
	return "Disable"
}

func atoiOr(s string, def int) int {
	n, err := strconv.Atoi(s)
	if err != nil {
		return def
	}
	return n
}

func atoi64Or(s string, def int64) int64 {
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return def
	}
	return n
}
