package onlogin

import (
	"fmt"
	"strings"

	"github.com/quiqxiq/roslib-mikhmon/domain"
)

// Build menghasilkan body script on-login untuk parameter o. Output bukan
// di-quote — caller bertanggung jawab embed ke value `=on-login=...`
// secara apa adanya (RouterOS API protocol length-prefix word, jadi tidak
// butuh escape kuotasi level kawat).
//
// Cross-ref: analisis §3.1.
func Build(o Options) string {
	var b strings.Builder
	writeMetadata(&b, o)

	if o.Mode == domain.ModeNone {
		// Mode 0: hanya metadata + tidak ada blok logika. PHP tetap dapat
		// parse harga via metadata `:put`.
		return b.String()
	}

	writeExpiryBlock(&b, o)

	if o.LockMAC {
		writeLockMACBlock(&b)
	}

	if o.Mode.RecordsTransaction() {
		writeRecordBlock(&b, o)
	}

	return b.String()
}

// writeMetadata menulis baris `:put` yang dipakai PHP untuk parse balik
// konfigurasi profile.
//
// Format mode != 0:  ",<expmode>,<price>,<validity>,<sprice>,,<lock>,"
// Format mode == 0:  ",,<price>,,,noexp,<lock>,"
func writeMetadata(b *strings.Builder, o Options) {
	if o.Mode == domain.ModeNone {
		fmt.Fprintf(b, `:put (",,%d,,,noexp,%s,");`+"\n", o.Price, o.metadataLockToken())
		return
	}
	fmt.Fprintf(b,
		`:put (",%s,%d,%s,%d,,%s,");`+"\n",
		o.Mode, o.Price, o.Validity, o.SellPrice, o.metadataLockToken(),
	)
}

func writeExpiryBlock(b *strings.Builder, o Options) {
	const tpl = `{
  :local comment [/ip hotspot user get [/ip hotspot user find where name="$user"] comment];
  :local ucode [:pic $comment 0 2];
  :if ($ucode = "vc" or $ucode = "up" or $comment = "") do={
    :local date  [/system clock get date];
    :local year  [:pick $date 7 11];
    :local month [:pick $date 0 3];
    /sys sch add name="$user" disable=no start-date=$date interval="<VALIDITY>";
    :delay 5s;
    :local exp    [/sys sch get [/sys sch find where name="$user"] next-run];
    :local getxp  [len $exp];
    :if ($getxp = 15) do={
      :local d [:pic $exp 0 6];
      :local t [:pic $exp 7 16];
      :local s ("/");
      :local exp ("$d$s$year $t");
      /ip hotspot user set comment="$exp" [find where name="$user"];
    };
    :if ($getxp = 8) do={
      /ip hotspot user set comment="$date $exp" [find where name="$user"];
    };
    :if ($getxp > 15) do={
      /ip hotspot user set comment="$exp" [find where name="$user"];
    };
    :delay 5s;
    /sys sch remove [find where name="$user"];
  }
}
`
	b.WriteString(strings.ReplaceAll(tpl, "<VALIDITY>", o.Validity))
}

func writeLockMACBlock(b *strings.Builder) {
	b.WriteString(`:local mac $"mac-address";
/ip hotspot user set mac-address=$mac [find where name=$user];
`)
}

// writeRecordBlock menulis snippet pembuatan /system/script transaksi.
// Nama script mengikuti format konvensi mikhmon (analisis §3.1):
//
//	<date>-|-<time>-|-<user>-|-<price>-|-<ip>-|-<mac>-|-<validity>-|-<profile>-|-<comment>
func writeRecordBlock(b *strings.Builder, o Options) {
	fmt.Fprintf(b, `:local mac $"mac-address";
:local time [/system clock get time];
/system script add name=("$date-|-$time-|-$user-|-%d-|-$"address"-|-$mac-|-%s-|-<PROFILE>-|-") owner=("<MONTHYEAR>") source=("$date") comment="mikhmon";
`, o.Price, o.Validity)
}

// PostProcessNamePlaceholders meng-substitusi placeholder `<PROFILE>` dan
// `<MONTHYEAR>` di hasil Build dengan nilai aktual. Dipisah dari Build()
// supaya generator murni (no time.Now). Dipanggil oleh workflows/ saat
// menempel script ke profile.
func PostProcessNamePlaceholders(script, profile, monthYear string) string {
	out := strings.ReplaceAll(script, "<PROFILE>", profile)
	out = strings.ReplaceAll(out, "<MONTHYEAR>", monthYear)
	return out
}
