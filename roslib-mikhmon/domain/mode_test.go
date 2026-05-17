package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExpiredMode_Validity(t *testing.T) {
	cases := map[ExpiredMode]bool{
		ModeNone:         true,
		ModeRemove:       true,
		ModeNotice:       true,
		ModeRemoveRecord: true,
		ModeNoticeRecord: true,
		ExpiredMode("xyz"): false,
		ExpiredMode(""):    false,
	}
	for m, ok := range cases {
		assert.Equal(t, ok, m.IsValid(), "mode=%q", m)
	}
}

func TestExpiredMode_RecordsTransaction(t *testing.T) {
	cases := map[ExpiredMode]bool{
		ModeNone:         false,
		ModeRemove:       false,
		ModeNotice:       false,
		ModeRemoveRecord: true,
		ModeNoticeRecord: true,
	}
	for m, want := range cases {
		assert.Equal(t, want, m.RecordsTransaction(), "mode=%q", m)
	}
}

func TestExpiredMode_HasExpiry(t *testing.T) {
	assert.False(t, ModeNone.HasExpiry())
	assert.True(t, ModeRemove.HasExpiry())
	assert.True(t, ModeNoticeRecord.HasExpiry())
}

func TestParseExpiredMode_RoundTrip(t *testing.T) {
	for _, m := range []ExpiredMode{ModeNone, ModeRemove, ModeNotice, ModeRemoveRecord, ModeNoticeRecord} {
		got, err := ParseExpiredMode(string(m))
		require.NoError(t, err)
		assert.Equal(t, m, got)
	}
}

func TestParseExpiredMode_Invalid(t *testing.T) {
	_, err := ParseExpiredMode("foo")
	require.Error(t, err)
}

func TestCharset_Validity(t *testing.T) {
	for _, c := range []Charset{CharsetLower, CharsetUpper, CharsetMixed, CharsetNumber, CharsetLowNum, CharsetUpNum, CharsetMixNum} {
		assert.True(t, c.IsValid(), "charset=%q", c)
	}
	assert.False(t, Charset("nope").IsValid())
}
