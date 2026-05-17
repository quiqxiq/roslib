// Package scripts (dan sub-paket onlogin, onevent, transaction, quickprint)
// adalah generator RouterOS script string yang dipakai mikhmon.
//
// Karakteristik:
//
//   - Pure function — input parameter, output string. Tidak ada IO.
//   - Mudah di-golden-test (perbandingan byte-for-byte vs file fixture).
//   - Caller (workflows/ atau handler REST) bertanggung jawab inject hasil
//     ke field on-login profil hotspot, on-event scheduler, dst.
//
// Lihat AGENTS.md untuk update flow golden file.
package scripts
