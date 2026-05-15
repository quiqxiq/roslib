package builder

import (
	"github.com/quiqxiq/roslib/capability"
	"github.com/quiqxiq/roslib/query"
)

// validatePrint cek Class command bukan Streaming + args dikenal.
func validatePrint(exec Executor, sentence []string, flags []string,
	pairs []query.Pair, wheres []query.WherePair) error {
	return validate(exec, sentence[0],
		"use .Stream() for streaming commands",
		[]capability.Class{capability.ClassOneShot, capability.ClassStreamablePrint, capability.ClassMutation},
		flags, pairs, wheres)
}

// validateMutation cek Class command Mutation + args dikenal.
func validateMutation(exec Executor, sentence []string, flags []string,
	pairs []query.Pair, wheres []query.WherePair) error {
	return validate(exec, sentence[0],
		"non-mutation command — periksa registry",
		[]capability.Class{capability.ClassMutation},
		flags, pairs, wheres)
}

// validateRun cek args dikenal saja (class apa pun OK — Run generik).
func validateRun(exec Executor, sentence []string, flags []string,
	pairs []query.Pair, wheres []query.WherePair) error {
	reg := exec.Registry()
	if reg == nil {
		return nil
	}
	names := collectNames(flags, pairs, wheres)
	if _, err := reg.Lookup(sentence[0]); err != nil {
		return wrapErr(exec, err)
	}
	if err := reg.ValidateArgs(sentence[0], names); err != nil {
		return wrapErr(exec, err)
	}
	return nil
}

func validate(exec Executor, word, hint string, allowed []capability.Class,
	flags []string, pairs []query.Pair, wheres []query.WherePair) error {
	reg := exec.Registry()
	if reg == nil {
		return nil
	}
	if _, err := reg.RequireClass(word, hint, allowed...); err != nil {
		return wrapErr(exec, err)
	}
	names := collectNames(flags, pairs, wheres)
	if err := reg.ValidateArgs(word, names); err != nil {
		return wrapErr(exec, err)
	}
	return nil
}

func wrapErr(exec Executor, err error) error {
	if exec.Strict() {
		return err
	}
	if log := exec.Logger(); log != nil {
		log.WithError(err).WithField("strict", false).
			Warn("capability validation skipped")
	}
	return nil
}

func collectNames(flags []string, pairs []query.Pair, wheres []query.WherePair) []string {
	names := make([]string, 0, len(flags)+len(pairs)+len(wheres))
	for _, f := range flags {
		name := f
		for i := 0; i < len(f); i++ {
			if f[i] == '=' {
				name = f[:i]
				break
			}
		}
		if name != "" {
			names = append(names, name)
		}
	}
	for _, p := range pairs {
		names = append(names, p.Key)
	}
	for _, w := range wheres {
		names = append(names, w.Key)
	}
	return names
}
