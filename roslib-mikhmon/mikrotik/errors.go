package mikrotik

import "errors"

// ErrNotFound dikembalikan helper *ByName / *ByID kalau tidak ada row.
var ErrNotFound = errors.New("mikrotik: resource not found")

// ErrAmbiguous dikembalikan helper *ByName kalau hasil > 1 row dan
// caller minta single result.
var ErrAmbiguous = errors.New("mikrotik: result is ambiguous (multiple rows)")

// ErrInvalidArgument dikembalikan kalau parameter wajib kosong.
var ErrInvalidArgument = errors.New("mikrotik: invalid argument")
