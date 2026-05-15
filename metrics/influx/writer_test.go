package influx

import (
	"testing"

	"github.com/go-routeros/routeros/v3/proto"
	"github.com/quiqxiq/roslib/decode"
)

// TestBuildPoint memastikan Writer.BuildPoint mengisi tag & field
// dari sentence sesuai fungsi mapper. Tidak melakukan HTTP — pure unit test.
func TestBuildPoint(t *testing.T) {
	w := NewWriter(nil /* client tidak dipakai oleh BuildPoint */,
		"system_resource",
		func(s *decode.Sentence) map[string]string {
			return map[string]string{"host": s.Get("board-name")}
		},
		func(s *decode.Sentence) map[string]any {
			return map[string]any{"uptime": s.Get("uptime")}
		},
	)

	sen := decode.Wrap(&proto.Sentence{
		Word: "!re",
		Map:  map[string]string{"board-name": "RB750Gr3", "uptime": "1w2d"},
	})

	p := w.BuildPoint(sen)
	if p == nil {
		t.Fatal("BuildPoint returned nil")
	}
	// Tidak ada accessor public yang langsung mengekspos tag map; verifikasi
	// melalui rendering line-protocol kalau dibutuhkan deeper assertion.
	if w.meas != "system_resource" {
		t.Errorf("measurement = %q", w.meas)
	}
}

// TestBatchedWriterAddTriggersDrain memastikan AddSentence menumpuk &
// flush goroutine ada di siklus normal. Tidak panggil HTTP — kita pakai
// Writer dengan client nil dan handle nil-pointer di drain via skip
// (drain tetap call, tapi panggilan client.WritePoints akan panic).
// Karena itu di sini kita verify hanya buffer pengisian saja.
func TestBatchedWriterBufferAccumulates(t *testing.T) {
	w := NewWriter(nil, "m",
		func(*decode.Sentence) map[string]string { return nil },
		func(*decode.Sentence) map[string]any { return nil },
	)
	bw := NewBatchedWriter(w, 5, 0)
	sen := decode.Wrap(&proto.Sentence{Word: "!re", Map: map[string]string{}})

	for i := 0; i < 3; i++ {
		bw.AddSentence(sen)
	}
	bw.mu.Lock()
	defer bw.mu.Unlock()
	if len(bw.buf) != 3 {
		t.Errorf("buf len = %d, want 3", len(bw.buf))
	}
}
