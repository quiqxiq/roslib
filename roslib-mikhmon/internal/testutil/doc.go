// Package testutil berisi helper untuk integration test (real router).
//
// Konvensi:
//
//   - RequireIntegration(t) wajib dipanggil di awal tiap test integration —
//     skip kalau ROSLIB_ROUTER_ADDRESS kosong.
//   - Resource yang dibuat selalu prefix `mikhmon-it-<test>-<rand>` agar
//     mudah cleanup manual kalau test crash.
package testutil
