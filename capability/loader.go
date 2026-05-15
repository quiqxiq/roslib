package capability

import (
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sync"
)

//go:embed assets/mikrotik/routeros_7.20.8.json
var embeddedRegistryJSON []byte

// LoadOptions mengontrol asal data JSON registry.
//
// Urutan prioritas: Bytes > Path > embed default.
type LoadOptions struct {
	// Path adalah file JSON eksternal (override embed).
	Path string
	// Bytes adalah JSON di-memory (override Path & embed).
	Bytes []byte
}

// defaultRegistry adalah singleton hasil load embed; lazy init.
var defaultRegistry = sync.OnceValues(func() (*Registry, error) {
	return parse(embeddedRegistryJSON)
})

// Default mengembalikan registry hasil parse JSON embedded.
// Aman dipanggil concurrent; parse dijalankan maksimal sekali per binary.
func Default() (*Registry, error) {
	return defaultRegistry()
}

// Load membangun Registry sesuai LoadOptions.
//
// Tanpa opsi (Path & Bytes kosong) sama dengan Default().
func Load(opts LoadOptions) (*Registry, error) {
	if len(opts.Bytes) > 0 {
		return parse(opts.Bytes)
	}
	if opts.Path != "" {
		data, err := os.ReadFile(opts.Path)
		if err != nil {
			return nil, fmt.Errorf("capability: read %s: %w", opts.Path, err)
		}
		return parse(data)
	}
	return Default()
}

// rootDoc adalah top-level JSON: {"version": "...", "tree": {...}}
type rootDoc struct {
	Version string                 `json:"version"`
	Tree    map[string]interface{} `json:"tree"`
}

// parse walk JSON tree rekursif dan menghasilkan Registry flat.
func parse(data []byte) (*Registry, error) {
	var doc rootDoc
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("capability: parse JSON: %w", err)
	}
	if doc.Tree == nil {
		return nil, errors.New("capability: JSON missing 'tree' object")
	}
	r := &Registry{
		Version: doc.Version,
		Cmds:    make(map[string]*Command, 600),
	}
	for name, child := range doc.Tree {
		walk(r, "/"+name, child)
	}
	return r, nil
}

// walk merekursifkan satu node JSON.
//
// Setiap node adalah map[string]any dengan field "_type" ∈ {dir,cmd,arg}.
// - dir → recurse ke setiap child (kecuali key "_type")
// - cmd → emit Command, args = children dengan _type=arg
// - arg → tidak emit apa-apa di tingkat ini (sudah jadi arg dari parent cmd)
func walk(r *Registry, path string, node interface{}) {
	m, ok := node.(map[string]interface{})
	if !ok {
		return
	}
	typ, _ := m["_type"].(string)
	switch typ {
	case "dir", "path":
		// Keduanya berfungsi sebagai namespace yang berisi children
		// (sub-dir, sub-path, atau cmd). RouterOS JSON memakai "path"
		// untuk subset top-level (7 simpul) seperti "tool".
		for childName, childNode := range m {
			if childName == "_type" {
				continue
			}
			walk(r, path+"/"+childName, childNode)
		}
	case "cmd":
		emitCommand(r, path, m)
		// Cmd juga bisa punya nested children (sub-dir di bawah cmd, jarang
		// tapi terjadi mis. /tool/sniffer punya start/stop/print sebagai
		// sibling action). RouterOS JSON memodelkan ini bukan sebagai
		// nested di bawah cmd — sniffer adalah dir di parent. Jadi tidak
		// ada rekursi tambahan di branch cmd.
	}
}

func emitCommand(r *Registry, word string, node map[string]interface{}) {
	args := make(map[string]struct{}, len(node))
	for name, child := range node {
		if name == "_type" {
			continue
		}
		childMap, ok := child.(map[string]interface{})
		if !ok {
			continue
		}
		if t, _ := childMap["_type"].(string); t == "arg" {
			args[name] = struct{}{}
		}
	}
	// Action = segment terakhir setelah '/'.
	action := word
	for i := len(word) - 1; i >= 0; i-- {
		if word[i] == '/' {
			action = word[i+1:]
			break
		}
	}
	r.Cmds[word] = &Command{
		Word:   word,
		Action: action,
		Args:   args,
		Class:  classify(word, action, args),
	}
}
