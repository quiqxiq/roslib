package rosfmt

import "strings"

// EscapeScriptString meng-escape string yang akan dimasukkan ke source
// /system/script atau ke field on-login profil hotspot. RouterOS script
// menggunakan backslash escape untuk karakter berikut:
//
//	\\  → backslash literal
//	\"  → quote literal di dalam string ber-quote
//	\$  → dollar literal (mencegah variable expansion)
//	\n  → newline literal di dalam string
//
// Implementasi ini paling konservatif — escape semua quote, backslash,
// dolar. Caller bertanggung jawab membungkus dengan tanda kutip kalau
// perlu.
func EscapeScriptString(s string) string {
	repl := strings.NewReplacer(
		`\`, `\\`,
		`"`, `\"`,
		`$`, `\$`,
		"\n", `\n`,
		"\r", `\r`,
	)
	return repl.Replace(s)
}

// QuoteScriptString = EscapeScriptString + bungkus dengan double-quote.
// Cocok untuk dipakai langsung sebagai value di sentence:
//
//	"=name=" + QuoteScriptString(name)  // ❌ jangan untuk API sentence
//
// Untuk API sentence pakai value mentah saja (tanpa escape) — protocol
// sudah length-prefix. Quote/escape hanya relevan untuk SCRIPT BODY.
func QuoteScriptString(s string) string {
	return `"` + EscapeScriptString(s) + `"`
}
