// Package tcpmock menyediakan server TCP minimal yang berbicara protocol
// RouterOS API (length-prefix word encoding) untuk test encoding-level.
//
// Ruang lingkup sengaja dibatasi: server hanya dapat menerima 1 koneksi,
// merekam semua sentence yang masuk, dan men-jawab dengan reply yang
// sudah di-script. Tidak ada login challenge, tidak ada state tracking,
// tidak ada concurrency. Cukup untuk verifikasi:
//
//   - encoding karakter spesial (quote, backslash, dolar, multibyte)
//   - sentence builder mengirim word yang benar berdasarkan input domain
//
// Untuk integration end-to-end pakai `test/integration/` ke router asli.
package tcpmock
