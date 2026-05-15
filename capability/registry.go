// Package capability mendefinisikan registry command RouterOS hasil parse
// dari JSON yang di-embed (atau di-override path-nya). Registry dipakai
// builder untuk:
//
//   - Klasifikasi command (one-shot / streaming / streamable-print / mutation)
//     supaya routing otomatis ke connCommand atau connStream.
//   - Validasi argumen sebelum kirim ke router (cegah typo & combo invalid).
//
// Format JSON: lihat /capability/assets/mikrotik/routeros_7.20.8.json.
// Node punya field _type ∈ {"dir","cmd","arg"}. Walk rekursif menghasilkan
// satu Command per node _type=cmd dengan word = "/seg1/seg2/.../lastSeg".
package capability

import (
	"fmt"
	"strings"
)

// Class adalah klasifikasi runtime dari command RouterOS.
type Class int

const (
	// ClassOneShot adalah command sekali-jalan: print biasa, find, get,
	// monitor-once, export, dst.
	ClassOneShot Class = iota
	// ClassMutation mengubah state router: add/set/remove/enable/disable/
	// comment/move/edit/reset/reset-counters/unset.
	ClassMutation
	// ClassStreamablePrint adalah print yang punya flag follow/follow-only/
	// interval — bisa dipanggil .Exec (snapshot) atau .Stream (long-running).
	ClassStreamablePrint
	// ClassStreaming adalah command inherently streaming yang tidak butuh
	// flag follow: monitor-traffic, ping, torch, sniffer, dll.
	ClassStreaming
)

// String mengembalikan nama Class untuk logging/debug.
func (c Class) String() string {
	switch c {
	case ClassOneShot:
		return "OneShot"
	case ClassMutation:
		return "Mutation"
	case ClassStreamablePrint:
		return "StreamablePrint"
	case ClassStreaming:
		return "Streaming"
	}
	return fmt.Sprintf("Class(%d)", int(c))
}

// Command adalah satu entry registry: satu kombinasi path+action di
// RouterOS yang valid, lengkap dengan daftar arg & klasifikasinya.
type Command struct {
	// Word adalah first word sentence yang dikirim ke router,
	// e.g. "/interface/monitor-traffic" atau "/ip/address/print".
	Word string

	// Action adalah segment terakhir dari Word.
	Action string

	// Args adalah set nama arg valid (lookup O(1)).
	Args map[string]struct{}

	// Class menentukan routing dan validasi.
	Class Class
}

// HasArg melaporkan apakah arg dengan nama tersebut valid untuk command ini.
func (c *Command) HasArg(name string) bool {
	if c == nil {
		return false
	}
	_, ok := c.Args[name]
	return ok
}

// Registry adalah index flat semua Command. Key = Command.Word.
type Registry struct {
	Version string
	Cmds    map[string]*Command
}

// ErrUnknownCommand dikembalikan Lookup bila word tidak terdaftar.
type ErrUnknownCommand struct{ Word string }

func (e *ErrUnknownCommand) Error() string {
	return "capability: unknown command word " + e.Word
}

// ErrUnknownArg dikembalikan ValidateArgs bila ada arg yang tidak dikenal.
type ErrUnknownArg struct {
	Word string
	Arg  string
}

func (e *ErrUnknownArg) Error() string {
	return "capability: arg " + e.Arg + " not valid for " + e.Word
}

// ErrInvalidClass dikembalikan saat command dipanggil lewat jalur yang
// salah — mis. Exec() di path streaming, atau Stream() di path one-shot.
type ErrInvalidClass struct {
	Word    string
	Got     Class
	Wanted  []Class
	Hint    string
}

func (e *ErrInvalidClass) Error() string {
	wanted := make([]string, 0, len(e.Wanted))
	for _, w := range e.Wanted {
		wanted = append(wanted, w.String())
	}
	hint := ""
	if e.Hint != "" {
		hint = " — " + e.Hint
	}
	return fmt.Sprintf("capability: %s has class %s, expected %s%s",
		e.Word, e.Got, strings.Join(wanted, "|"), hint)
}

// Lookup mengambil Command berdasarkan first word sentence.
func (r *Registry) Lookup(word string) (*Command, error) {
	if r == nil {
		return nil, &ErrUnknownCommand{Word: word}
	}
	c, ok := r.Cmds[word]
	if !ok {
		return nil, &ErrUnknownCommand{Word: word}
	}
	return c, nil
}

// ValidateArgs cek setiap nama arg dalam argNames apakah valid untuk Command
// dengan word tertentu. Argument "raw" flag (mis. "detail") dan named pair
// keys (mis. "address" dari "=address=10.0.0.1/24") sama-sama dilewatkan ke
// sini sebagai single list nama.
func (r *Registry) ValidateArgs(word string, argNames []string) error {
	cmd, err := r.Lookup(word)
	if err != nil {
		return err
	}
	for _, name := range argNames {
		if name == "" {
			continue
		}
		if !cmd.HasArg(name) {
			return &ErrUnknownArg{Word: word, Arg: name}
		}
	}
	return nil
}

// RequireClass memverifikasi Class command match salah satu dari allowed.
// Mengembalikan ErrInvalidClass dengan hint kalau tidak match.
func (r *Registry) RequireClass(word string, hint string, allowed ...Class) (*Command, error) {
	cmd, err := r.Lookup(word)
	if err != nil {
		return nil, err
	}
	for _, c := range allowed {
		if cmd.Class == c {
			return cmd, nil
		}
	}
	return cmd, &ErrInvalidClass{
		Word:   word,
		Got:    cmd.Class,
		Wanted: allowed,
		Hint:   hint,
	}
}
