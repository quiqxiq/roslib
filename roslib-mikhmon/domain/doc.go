// Package domain berisi value-object murni untuk roslib-mikhmon.
// Tidak ada IO di paket ini — semua tipe deterministik dan mudah di-test.
//
// Tipe utama:
//
//   - HotspotUser, HotspotProfile           — proyeksi data user/profile dari RouterOS.
//   - VoucherSpec                           — parameter generate voucher batch.
//   - ExpiredMode (None/Rem/Ntf/RemC/NtfC)  — perilaku saat user expired.
//   - Charset                               — kategori karakter generator voucher.
//   - TransactionRecord                     — hasil parse nama script transaksi.
package domain
