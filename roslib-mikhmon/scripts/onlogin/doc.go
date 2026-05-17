// Package onlogin men-generate body script yang ditempel ke
// /ip/hotspot/user/profile.on-login (analisis §3.1).
//
// Tiga blok:
//
//   - metadata `:put` di awal — dipakai PHP untuk mem-parse balik config.
//   - kalkulasi expiry (mode != 0) atau no-op (mode == 0).
//   - opsional: lock MAC + record transaksi (mode remc/ntfc).
package onlogin
