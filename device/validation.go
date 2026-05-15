package device

import (
	"github.com/quiqxiq/roslib/capability"
	"github.com/quiqxiq/roslib/poll"
	"github.com/quiqxiq/roslib/query"
	"github.com/quiqxiq/roslib/stream"
)

// validatePollConfig cek capability registry untuk poll config.
//
//   - Lookup command word (Path + "/" + Args[0]) — harus terdaftar.
//   - Class harus bukan ClassStreaming (poll bukan listener).
//   - Args/Pairs/Where harus dikenal sebagai arg yang valid.
//
// Hasil non-nil hanya kalau Strict=true. Strict=false → log-warn lalu nil.
func (d *RouterDevice) validatePollConfig(cfg poll.Config) error {
	if d.opts.Registry == nil {
		return nil
	}
	action := "print"
	flagArgs := []string(nil)
	if len(cfg.Args) > 0 {
		action = cfg.Args[0]
		flagArgs = cfg.Args[1:]
	}
	word := cfg.Path + "/" + action
	return d.validateWord(word, "use .Stream() for streaming commands",
		[]capability.Class{capability.ClassOneShot, capability.ClassStreamablePrint, capability.ClassMutation},
		flagArgs, cfg.Pairs, cfg.Where)
}

// validateStreamSpec cek capability registry untuk stream spec.
//
//   - Lookup spec.Word — harus terdaftar.
//   - Class harus Streaming ATAU StreamablePrint.
func (d *RouterDevice) validateStreamSpec(spec stream.Spec) error {
	if d.opts.Registry == nil {
		return nil
	}
	return d.validateWord(spec.Word, "use .Exec() for non-streaming commands",
		[]capability.Class{capability.ClassStreaming, capability.ClassStreamablePrint},
		spec.Args, spec.Pairs, spec.Where)
}

func (d *RouterDevice) validateWord(word, hint string, allowed []capability.Class,
	flags []string, pairs []query.Pair, wheres []query.WherePair) error {
	if _, err := d.opts.Registry.RequireClass(word, hint, allowed...); err != nil {
		return d.handleErr(err)
	}
	names := collectArgNames(flags, pairs, wheres)
	if err := d.opts.Registry.ValidateArgs(word, names); err != nil {
		return d.handleErr(err)
	}
	return nil
}

func (d *RouterDevice) handleErr(err error) error {
	if d.opts.StrictCapability {
		return err
	}
	d.log.WithError(err).WithField("strict", false).Warn("capability validation skipped")
	return nil
}

// collectArgNames merangkum semua nama arg yang muncul di flag bebas
// (kata seperti "detail" atau "follow"), pair keys, dan where keys.
// Flag bebas yang mengandung "=" (mis. "interval=1s") di-split, ambil prefix
// sebagai nama arg.
func collectArgNames(flags []string, pairs []query.Pair, wheres []query.WherePair) []string {
	names := make([]string, 0, len(flags)+len(pairs)+len(wheres))
	for _, f := range flags {
		name := f
		for i := 0; i < len(f); i++ {
			if f[i] == '=' {
				name = f[:i]
				break
			}
		}
		if name == "" {
			continue
		}
		names = append(names, name)
	}
	for _, p := range pairs {
		names = append(names, p.Key)
	}
	for _, w := range wheres {
		names = append(names, w.Key)
	}
	return names
}

