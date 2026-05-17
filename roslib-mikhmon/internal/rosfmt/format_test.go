package rosfmt

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDateInt(t *testing.T) {
	tests := []struct {
		in   string
		want int
	}{
		{"jan/05/2025", 20250105},
		{"feb/29/2024", 20240229},
		{"dec/31/2099", 20991231},
		{"oct/01/1970", 19701001},
	}
	for _, tt := range tests {
		got, err := DateInt(tt.in)
		require.NoError(t, err, tt.in)
		assert.Equal(t, tt.want, got, tt.in)
	}
}

func TestDateInt_Errors(t *testing.T) {
	for _, in := range []string{"", "xyz/01/2025", "jan/xx/2025", "short"} {
		_, err := DateInt(in)
		assert.Error(t, err, in)
	}
}

func TestTimeMinutes(t *testing.T) {
	tests := []struct {
		in   string
		want int
	}{
		{"00:00:00", 0},
		{"01:30:00", 90},
		{"23:59:59", 23*60 + 59},
		{"12:00", 12 * 60},
	}
	for _, tt := range tests {
		got, err := TimeMinutes(tt.in)
		require.NoError(t, err, tt.in)
		assert.Equal(t, tt.want, got, tt.in)
	}
}

func TestFormatDate_RoundTrip(t *testing.T) {
	t0 := time.Date(2025, 1, 5, 0, 0, 0, 0, time.UTC)
	s := FormatDate(t0)
	assert.Equal(t, "jan/05/2025", s)
	parsed, err := ParseDate(s, time.UTC)
	require.NoError(t, err)
	assert.Equal(t, t0, parsed)
}

func TestCurrentMonthOwner(t *testing.T) {
	t0 := time.Date(2025, 3, 15, 0, 0, 0, 0, time.UTC)
	assert.Equal(t, "Mar2025", CurrentMonthOwner(t0))
}

func TestParseDuration(t *testing.T) {
	tests := []struct {
		in   string
		want time.Duration
	}{
		{"30s", 30 * time.Second},
		{"5m", 5 * time.Minute},
		{"1h", time.Hour},
		{"1d", 24 * time.Hour},
		{"1w", 7 * 24 * time.Hour},
		{"1w2d3h", 7*24*time.Hour + 2*24*time.Hour + 3*time.Hour},
		{"30", 30 * time.Second}, // tanpa unit
	}
	for _, tt := range tests {
		got, err := ParseDuration(tt.in)
		require.NoError(t, err, tt.in)
		assert.Equal(t, tt.want, got, tt.in)
	}
}

func TestParseDuration_Errors(t *testing.T) {
	for _, in := range []string{"", "abc", "1x", "1h2x"} {
		_, err := ParseDuration(in)
		assert.Error(t, err, in)
	}
}

func TestIsValidDuration(t *testing.T) {
	assert.True(t, IsValidDuration("1d"))
	assert.False(t, IsValidDuration("xyz"))
}

func TestEscapeScriptString(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{`hello`, `hello`},
		{`a"b`, `a\"b`},
		{`a\b`, `a\\b`},
		{`$user`, `\$user`},
		{"line1\nline2", `line1\nline2`},
		{`\"$x"`, `\\\"\$x\"`},
	}
	for _, tt := range tests {
		got := EscapeScriptString(tt.in)
		assert.Equal(t, tt.want, got, tt.in)
	}
}

func TestQuoteScriptString(t *testing.T) {
	assert.Equal(t, `"hello"`, QuoteScriptString("hello"))
	assert.Equal(t, `"a\"b"`, QuoteScriptString(`a"b`))
}
