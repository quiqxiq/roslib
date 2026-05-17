package transaction

import (
	"errors"
	"strings"

	"github.com/quiqxiq/roslib-mikhmon/domain"
)

// Separator antar field di nama script transaksi.
const Separator = "-|-"

// FieldCount adalah jumlah field di nama script (analisis §3.1).
const FieldCount = 9

// ErrFieldCount dikembalikan Parse jika nama tidak punya 9 field.
var ErrFieldCount = errors.New("transaction: name does not have 9 fields")

// Format menghasilkan nama script dari TransactionRecord.
// Field comment paling akhir BUKAN comment script-nya (yang harus
// "mikhmon"); ini comment kasir/operator.
func Format(rec domain.TransactionRecord) string {
	parts := []string{
		rec.Date,
		rec.Time,
		rec.User,
		rec.Price,
		rec.IP,
		rec.MAC,
		rec.Validity,
		rec.Profile,
		rec.Comment,
	}
	return strings.Join(parts, Separator)
}

// Parse mengembalikan TransactionRecord dari nama script. Kalau format
// tidak sesuai (≠ 9 field), kembalikan ErrFieldCount.
//
// Field Owner & Source TIDAK di-isi — keduanya di-store di field
// /system/script lain yang berbeda dan harus di-set caller.
func Parse(name string) (domain.TransactionRecord, error) {
	parts := strings.Split(name, Separator)
	if len(parts) != FieldCount {
		return domain.TransactionRecord{}, ErrFieldCount
	}
	return domain.TransactionRecord{
		Date:     parts[0],
		Time:     parts[1],
		User:     parts[2],
		Price:    parts[3],
		IP:       parts[4],
		MAC:      parts[5],
		Validity: parts[6],
		Profile:  parts[7],
		Comment:  parts[8],
	}, nil
}
