package quickprint

import (
	"testing"

	"github.com/quiqxiq/roslib-mikhmon/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFormatParse_RoundTrip(t *testing.T) {
	c := Config{
		Name:      "1day",
		Server:    "all",
		UserMode:  "vc",
		Length:    8,
		Prefix:    "user",
		Charset:   domain.CharsetMixed,
		Profile:   "1day-prof",
		TimeLimit: "1d",
		DataLimit: 1024 * 1024 * 100,
		Comment:   "kasir",
		Validity:  "30d",
		Price:     5000,
		SellPrice: 4500,
		LockMAC:   true,
	}
	src := Format(c)
	got, err := Parse(src)
	require.NoError(t, err)
	assert.Equal(t, c, got)
}

func TestParse_NoSellPriceField(t *testing.T) {
	// Format lama yang hanya simpan price tanpa _sprice.
	src := "#1day#all#vc#8#user#mixed#1day-prof#1d#0#kasir#30d#5000#Disable"
	c, err := Parse(src)
	require.NoError(t, err)
	assert.Equal(t, 5000, c.Price)
	assert.Equal(t, 0, c.SellPrice)
	assert.False(t, c.LockMAC)
}

func TestParse_BadFormat(t *testing.T) {
	for _, in := range []string{"", "#a#b#c", "noseps"} {
		_, err := Parse(in)
		assert.ErrorIs(t, err, ErrFieldCount, in)
	}
}

func TestFormat_PriceCombined(t *testing.T) {
	c := Config{Price: 5000, SellPrice: 4500, Length: 8, DataLimit: 0}
	src := Format(c)
	assert.Contains(t, src, "#5000_4500#")
}
