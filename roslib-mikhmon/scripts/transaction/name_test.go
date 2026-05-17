package transaction

import (
	"testing"

	"github.com/quiqxiq/roslib-mikhmon/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFormatParse_RoundTrip(t *testing.T) {
	rec := domain.TransactionRecord{
		Date:     "jan/05/2025",
		Time:     "14:32:01",
		User:     "user001",
		Price:    "5000",
		IP:       "192.168.88.10",
		MAC:      "AA:BB:CC:DD:EE:FF",
		Validity: "1d",
		Profile:  "1day",
		Comment:  "kasir-andi",
	}
	name := Format(rec)
	assert.Equal(t, "jan/05/2025-|-14:32:01-|-user001-|-5000-|-192.168.88.10-|-AA:BB:CC:DD:EE:FF-|-1d-|-1day-|-kasir-andi", name)

	got, err := Parse(name)
	require.NoError(t, err)
	assert.Equal(t, rec, got)
}

func TestParse_BadFormat(t *testing.T) {
	for _, in := range []string{
		"",
		"foo",
		"a-|-b-|-c", // hanya 3 field
	} {
		_, err := Parse(in)
		assert.ErrorIs(t, err, ErrFieldCount, in)
	}
}

func TestParse_EmptyFieldsAllowed(t *testing.T) {
	// Nama bisa berisi field kosong (mis. comment kosong) — tetap valid.
	in := "jan/05/2025-|-14:32:01-|-u1-|-0-|--|--|--|--|-"
	got, err := Parse(in)
	require.NoError(t, err)
	assert.Equal(t, "u1", got.User)
	assert.Equal(t, "", got.Comment)
}
