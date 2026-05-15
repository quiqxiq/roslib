package builder

import (
	"context"
	"encoding/json"
	"time"

	"github.com/go-routeros/routeros/v3"
	"github.com/go-routeros/routeros/v3/proto"
	"github.com/quiqxiq/roslib/cache"
	"github.com/quiqxiq/roslib/decode"
	"github.com/quiqxiq/roslib/query"
)

// PrintBuilder mengakumulasi opsi Print sebelum di-exec atau diubah menjadi
// stream listener.
type PrintBuilder struct {
	exec  Executor
	path  string
	flags []string
	where []query.WherePair
	pairs []query.Pair
}

// Detail menambahkan flag "detail" — RouterOS akan mengirim seluruh field.
func (p *PrintBuilder) Detail() *PrintBuilder {
	p.flags = append(p.flags, "detail")
	return p
}

// Stats menambahkan flag "stats" — counter/statistik (mis. di /interface).
func (p *PrintBuilder) Stats() *PrintBuilder {
	p.flags = append(p.flags, "stats")
	return p
}

// Count menambahkan flag "count-only" — RouterOS hanya kirim jumlah baris.
func (p *PrintBuilder) Count() *PrintBuilder {
	p.flags = append(p.flags, "count-only")
	return p
}

// Flag menambahkan kata flag bebas (raw word) ke sentence.
func (p *PrintBuilder) Flag(word string) *PrintBuilder {
	p.flags = append(p.flags, word)
	return p
}

// Where menambahkan filter "?key=value".
func (p *PrintBuilder) Where(key, value string) *PrintBuilder {
	p.where = append(p.where, query.Where(key, value))
	return p
}

// WherePair menambahkan filter lengkap (mis. operator non-equal).
func (p *PrintBuilder) WherePair(w query.WherePair) *PrintBuilder {
	p.where = append(p.where, w)
	return p
}

// With menambahkan named parameter "=key=value".
func (p *PrintBuilder) With(key, value string) *PrintBuilder {
	p.pairs = append(p.pairs, query.NewPair(key, value))
	return p
}

// command membangun sentence final.
func (p *PrintBuilder) command() []string {
	return query.BuildSentence(p.path+"/print", p.flags, p.pairs, p.where)
}

// Reply membungkus hasil print sebagai slice *decode.Sentence yang
// langsung bisa dipakai handler typed accessor.
type Reply struct {
	Raw  *routeros.Reply
	Rows []*decode.Sentence
}

// Exec mengirim print sekali dan tunggu reply (snapshot, bukan follow).
func (p *PrintBuilder) Exec(ctx context.Context) (*Reply, error) {
	sentence := p.command()
	if err := validatePrint(p.exec, sentence, p.flags, p.pairs, p.where); err != nil {
		return nil, err
	}
	raw, err := p.exec.RunCommand(ctx, sentence)
	if err != nil {
		return nil, err
	}
	return wrapReply(raw), nil
}

// ExecCached identik dengan Exec, tapi cek cache di awal dan simpan hasilnya
// dengan TTL tertentu. ttl=0 → pakai default dari Executor.CacheTTL().
//
// Cache key dari sentence canonical (sha256). Encoding JSON ke cache —
// debuggable lewat redis-cli `GET roslib:<hex>`.
//
// Kalau Cache adalah NoopCache, ExecCached berperilaku sama dengan Exec
// (selalu miss → selalu hit router).
func (p *PrintBuilder) ExecCached(ctx context.Context, ttl time.Duration) (*Reply, error) {
	sentence := p.command()
	if err := validatePrint(p.exec, sentence, p.flags, p.pairs, p.where); err != nil {
		return nil, err
	}
	c := p.exec.Cache()
	key := cache.KeyOf(sentence)

	if data, hit, err := c.Get(ctx, key); err == nil && hit {
		var rep cachedReply
		if jerr := json.Unmarshal(data, &rep); jerr == nil {
			return rep.toReply(), nil
		}
		// Kalau decode gagal, fall-through ke fetch.
	}

	raw, err := p.exec.RunCommand(ctx, sentence)
	if err != nil {
		return nil, err
	}
	reply := wrapReply(raw)

	if ttl <= 0 {
		ttl = p.exec.CacheTTL()
	}
	if encoded, jerr := json.Marshal(toCached(raw)); jerr == nil {
		_ = c.Set(ctx, key, encoded, ttl)
	}
	return reply, nil
}

func wrapReply(raw *routeros.Reply) *Reply {
	rows := make([]*decode.Sentence, 0, len(raw.Re))
	for _, re := range raw.Re {
		rows = append(rows, decode.Wrap(re))
	}
	return &Reply{Raw: raw, Rows: rows}
}

// cachedReply adalah bentuk serializable untuk *routeros.Reply.Re[*].Map.
// Kita tidak menyerialisasi tag/word — hanya field key/value yang penting
// untuk caller (semua handler hanya pakai .Map via decode.Sentence).
type cachedReply struct {
	Rows []map[string]string `json:"rows"`
}

func toCached(raw *routeros.Reply) cachedReply {
	out := cachedReply{Rows: make([]map[string]string, 0, len(raw.Re))}
	for _, re := range raw.Re {
		out.Rows = append(out.Rows, copyMap(re.Map))
	}
	return out
}

func (cr cachedReply) toReply() *Reply {
	rows := make([]*decode.Sentence, 0, len(cr.Rows))
	for _, m := range cr.Rows {
		sen := &proto.Sentence{Word: "!re", Map: copyMap(m)}
		rows = append(rows, decode.Wrap(sen))
	}
	return &Reply{Rows: rows}
}

func copyMap(m map[string]string) map[string]string {
	out := make(map[string]string, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}

// Follow membuat builder mode listen=follow (snapshot + delta).
func (p *PrintBuilder) Follow() *StreamBuilder {
	return &StreamBuilder{p: p, mode: modeFollow}
}

// FollowOnly membuat builder mode listen=follow-only (hanya event baru).
func (p *PrintBuilder) FollowOnly() *StreamBuilder {
	return &StreamBuilder{p: p, mode: modeFollowOnly}
}
