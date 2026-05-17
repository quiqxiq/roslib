// Package mikhmon adalah sub-module Go yang merepiklikasi semua RouterOS
// command + script dari aplikasi PHP `mikhmonv3` di atas library `roslib`.
//
// Iterasi pertama hanya mengimplementasi command-layer dan script-generator
// (lihat docs/ARCHITECTURE.md). REST + SSE handler menyusul iterasi
// berikutnya.
//
// Layer:
//
//   - domain/                    tipe value-object murni (no IO)
//   - mikrotik/                  thin wrapper command per modul RouterOS
//   - scripts/                   generator string RouterOS script
//   - workflows/                 cascade orchestration multi-step
//   - internal/rosfmt/           format & escape helper
//   - internal/tcpmock/          mock TCP server (hybrid test)
//   - internal/testutil/         helper integration test
//
// Struktur ini sengaja flat dan predictable supaya pair-programming dengan
// AI agent menghasilkan diff kecil dan konteks-minimal.
package mikhmon
