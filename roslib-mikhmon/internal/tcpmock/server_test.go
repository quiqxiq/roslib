package tcpmock

import (
	"bufio"
	"bytes"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEncodeDecode_RoundTrip(t *testing.T) {
	cases := [][]string{
		{},
		{"hello"},
		{"/ip/hotspot/user/print"},
		{"=name=foo bar baz"},
		{string(make([]byte, 200))},     // 2-byte prefix path
		{string(make([]byte, 0x4001))},  // 3-byte prefix path
		{`=comment="quoted \"text\""`},  // escape chars in payload
	}
	for _, want := range cases {
		var buf bytes.Buffer
		require.NoError(t, EncodeSentence(&buf, want))
		got, err := DecodeSentence(bufio.NewReader(&buf))
		require.NoError(t, err)
		if len(want) == 0 {
			assert.Empty(t, got)
		} else {
			assert.Equal(t, want, got)
		}
	}
}

func TestServer_RecordsAndScriptsReply(t *testing.T) {
	srv, err := Start()
	require.NoError(t, err)
	defer srv.Close()

	srv.Script([]string{"!done", "=ret=*42"})

	conn, err := net.DialTimeout("tcp", srv.Addr(), 2*time.Second)
	require.NoError(t, err)
	defer conn.Close()

	require.NoError(t, EncodeSentence(conn, []string{"/ip/hotspot/user/print", "?name=foo"}))

	r := bufio.NewReader(conn)
	got, err := DecodeSentence(r)
	require.NoError(t, err)
	assert.Equal(t, []string{"!done", "=ret=*42"}, got)

	// give server a moment to record
	time.Sleep(20 * time.Millisecond)
	rcv := srv.Received()
	require.Len(t, rcv, 1)
	assert.Equal(t, []string{"/ip/hotspot/user/print", "?name=foo"}, rcv[0])
}
