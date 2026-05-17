package onevent

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
		{name: "remove_1day", opts: Options{ProfileName: "1day", Mode: domain.ModeRemove}},
		{name: "notice_30day", opts: Options{ProfileName: "30day", Mode: domain.ModeNotice}},
		{name: "remove_record_voucher", opts: Options{ProfileName: "voucher-1h", Mode: domain.ModeRemoveRecord}},
		{name: "notice_record_voucher", opts: Options{ProfileName: "voucher-1h", Mode: domain.ModeNoticeRecord}},
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

func TestActionForMode(t *testing.T) {
	assert.Equal(t, "/ip hotspot user remove $i", actionForMode(domain.ModeRemove))
	assert.Equal(t, "/ip hotspot user remove $i", actionForMode(domain.ModeRemoveRecord))
	assert.Equal(t, "/ip hotspot user set limit-uptime=1s $i", actionForMode(domain.ModeNotice))
	assert.Equal(t, "/ip hotspot user set limit-uptime=1s $i", actionForMode(domain.ModeNoticeRecord))
}
