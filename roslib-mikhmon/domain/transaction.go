package domain

// TransactionRecord adalah hasil parse nama /system/script transaksi.
// Format nama script: "<date>-|-<time>-|-<user>-|-<price>-|-<ip>-|-<mac>-|-<validity>-|-<profile>-|-<comment>".
// Field comment script-nya sendiri selalu "mikhmon" (sebagai filter di /system/script/print).
//
// Cross-ref: analisis §3.1 (tabel "Konvensi field pada nama script transaksi").
type TransactionRecord struct {
	Date     string // jan/02/2006
	Time     string // HH:MM:SS
	User     string
	Price    string // disimpan string supaya tidak lossy untuk format custom
	IP       string
	MAC      string
	Validity string
	Profile  string
	Comment  string

	// Owner = "<bulan><tahun>" — disimpan terpisah karena owner bukan bagian dari name.
	Owner string

	// Source = "<date>" — disimpan terpisah, dipakai sebagai filter "today's report".
	Source string
}
