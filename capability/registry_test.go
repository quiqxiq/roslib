package capability

import (
	"testing"
)

// TestDefaultRegistryLoads memastikan JSON embedded ter-parse.
func TestDefaultRegistryLoads(t *testing.T) {
	r, err := Default()
	if err != nil {
		t.Fatalf("Default() error: %v", err)
	}
	if r.Version == "" {
		t.Error("expected non-empty Version")
	}
	if len(r.Cmds) < 500 {
		t.Errorf("expected ~541 commands, got %d", len(r.Cmds))
	}
	t.Logf("loaded RouterOS %s with %d commands", r.Version, len(r.Cmds))
}

// TestClassification memastikan command kunci diklasifikasi benar.
func TestClassification(t *testing.T) {
	r, err := Default()
	if err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		word string
		want Class
	}{
		// Inherent streaming (whitelist exact).
		{"/interface/monitor-traffic", ClassStreaming},
		{"/tool/ping", ClassStreaming},
		{"/tool/torch", ClassStreaming},
		{"/tool/flood-ping", ClassStreaming},
		{"/tool/traceroute", ClassStreaming},
		{"/tool/bandwidth-test", ClassStreaming},

		// Monitor sub-action di /interface/*.
		{"/interface/ethernet/monitor", ClassStreaming},

		// Print yang punya follow → streamable.
		{"/log/print", ClassStreamablePrint},
		{"/ip/address/print", ClassStreamablePrint},

		// Mutation.
		{"/ip/address/add", ClassMutation},
		{"/ip/address/remove", ClassMutation},
		{"/ip/address/set", ClassMutation},
		{"/ip/firewall/filter/enable", ClassMutation},
	}

	for _, tc := range cases {
		cmd, err := r.Lookup(tc.word)
		if err != nil {
			t.Errorf("Lookup(%q) error: %v", tc.word, err)
			continue
		}
		if cmd.Class != tc.want {
			t.Errorf("Lookup(%q).Class = %s, want %s", tc.word, cmd.Class, tc.want)
		}
	}
}

// TestValidateArgs cek penolakan arg tidak dikenal.
func TestValidateArgs(t *testing.T) {
	r, err := Default()
	if err != nil {
		t.Fatal(err)
	}

	// /ip/address/add punya arg "address" dan "interface".
	if err := r.ValidateArgs("/ip/address/add", []string{"address", "interface"}); err != nil {
		t.Errorf("expected valid: %v", err)
	}
	if err := r.ValidateArgs("/ip/address/add", []string{"addres"}); err == nil {
		t.Error("expected ErrUnknownArg for typo, got nil")
	}
}

// TestRequireClass cek error untuk path streaming dipakai non-stream.
func TestRequireClass(t *testing.T) {
	r, err := Default()
	if err != nil {
		t.Fatal(err)
	}

	// /interface/monitor-traffic adalah Streaming; minta OneShot harus error.
	_, err = r.RequireClass("/interface/monitor-traffic",
		"use .Stream() instead of .Exec()", ClassOneShot)
	if err == nil {
		t.Error("expected ErrInvalidClass, got nil")
	}

	// Minta Streaming pada path streaming harus OK.
	if _, err := r.RequireClass("/interface/monitor-traffic", "", ClassStreaming); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

// TestLoadOverride memastikan opsi Bytes meng-override embed.
func TestLoadOverride(t *testing.T) {
	tiny := []byte(`{"version":"test","tree":{"hello":{"_type":"cmd","world":{"_type":"arg"}}}}`)
	r, err := Load(LoadOptions{Bytes: tiny})
	if err != nil {
		t.Fatal(err)
	}
	if r.Version != "test" {
		t.Errorf("Version = %q, want test", r.Version)
	}
	if _, err := r.Lookup("/hello"); err != nil {
		t.Errorf("Lookup(/hello): %v", err)
	}
}
