package onlogin

import (
	"flag"
	"os"
	"path/filepath"
	"testing"

	"github.com/quiqxiq/roslib-mikhmon/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var update = flag.Bool("update", false, "regenerate golden files")

func TestBuild_GoldenFiles(t *testing.T) {
	cases := []struct {
		name string
		opts Options
	}{
		{
			name: "mode_none",
			opts: Options{Mode: domain.ModeNone, Price: 5000},
		},
		{
			name: "mode_remove",
			opts: Options{Mode: domain.ModeRemove, Validity: "1d", Price: 5000, SellPrice: 4500},
		},
		{
			name: "mode_notice",
			opts: Options{Mode: domain.ModeNotice, Validity: "30d", Price: 25000, SellPrice: 20000},
		},
		{
			name: "mode_remove_record",
			opts: Options{Mode: domain.ModeRemoveRecord, Validity: "1d", Price: 5000, SellPrice: 4500},
		},
		{
			name: "mode_notice_record_lock",
			opts: Options{Mode: domain.ModeNoticeRecord, Validity: "30d", Price: 25000, SellPrice: 20000, LockMAC: true},
		},
		{
			name: "mode_remove_lock_only",
			opts: Options{Mode: domain.ModeRemove, Validity: "1d", Price: 5000, SellPrice: 4500, LockMAC: true},
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			got := Build(tt.opts)
			path := filepath.Join("testdata", "golden", tt.name+".txt")
			if *update {
				require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
				require.NoError(t, os.WriteFile(path, []byte(got), 0o644))
				return
			}
			want, err := os.ReadFile(path)
			require.NoError(t, err, "missing golden file (run with -update)")
			assert.Equal(t, string(want), got)
		})
	}
}

func TestPostProcessNamePlaceholders(t *testing.T) {
	in := `name=("$date-|-...-|-<PROFILE>-|-") owner=("<MONTHYEAR>")`
	got := PostProcessNamePlaceholders(in, "1day", "Jan2025")
	assert.Equal(t, `name=("$date-|-...-|-1day-|-") owner=("Jan2025")`, got)
}
