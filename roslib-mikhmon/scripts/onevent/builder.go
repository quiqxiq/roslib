package onevent

import (
	"fmt"
	"strings"

	"github.com/quiqxiq/roslib-mikhmon/domain"
)

// Options adalah parameter generate body on-event.
//
// Cross-ref: analisis §3.2.
type Options struct {
	ProfileName string             // wajib — nama profil hotspot yang di-monitor
	Mode        domain.ExpiredMode // ModeRemove*/ModeNotice* (None tidak masuk akal di sini)
}

// Build menghasilkan body script untuk dipasang ke
// /system/scheduler.on-event. Caller harus pastikan Mode != ModeNone.
func Build(o Options) string {
	action := actionForMode(o.Mode)
	const tpl = `:local dateint do={
  :local montharray ("jan","feb","mar","apr","may","jun","jul","aug","sep","oct","nov","dec");
  :local days  [:pick $d 4 6];
  :local month [:pick $d 0 3];
  :local year  [:pick $d 7 11];
  :local monthint ([:find $montharray $month]);
  :local month ($monthint + 1);
  :if ([len $month] = 1) do={
    :local zero ("0");
    :return [:tonum ("$year$zero$month$days")];
  } else={
    :return [:tonum ("$year$month$days")];
  }
};
:local timeint do={
  :local hours   [:pick $t 0 2];
  :local minutes [:pick $t 3 5];
  :return ($hours * 60 + $minutes);
};
:local date    [/system clock get date];
:local time    [/system clock get time];
:local today   [$dateint d=$date];
:local curtime [$timeint t=$time];
:foreach i in [/ip hotspot user find where profile="<PROFILE>"] do={
  :local comment [/ip hotspot user get $i comment];
  :local name    [/ip hotspot user get $i name];
  :local gettime [:pic $comment 12 20];
  :if ([:pic $comment 3] = "/" and [:pic $comment 6] = "/") do={
    :local expd [$dateint d=$comment];
    :local expt [$timeint t=$gettime];
    :if (
      ($expd < $today and $expt < $curtime) or
      ($expd < $today and $expt > $curtime) or
      ($expd = $today and $expt < $curtime)
    ) do={
      <ACTION>;
      [/ip hotspot active remove [find where user=$name]];
    };
  };
};
`
	out := strings.ReplaceAll(tpl, "<PROFILE>", o.ProfileName)
	out = strings.ReplaceAll(out, "<ACTION>", action)
	return out
}

// actionForMode menerjemahkan ExpiredMode ke statement RouterOS yang
// di-substitusi ke placeholder <ACTION>.
//
//   - rem / remc → /ip hotspot user remove $i
//   - ntf / ntfc → /ip hotspot user set limit-uptime=1s $i
func actionForMode(m domain.ExpiredMode) string {
	switch m {
	case domain.ModeRemove, domain.ModeRemoveRecord:
		return "/ip hotspot user remove $i"
	case domain.ModeNotice, domain.ModeNoticeRecord:
		return "/ip hotspot user set limit-uptime=1s $i"
	}
	return fmt.Sprintf(":log warning \"onevent: unsupported mode %q\"", m)
}
