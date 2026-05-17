package system

import (
	"context"

	"github.com/quiqxiq/roslib"
	"github.com/quiqxiq/roslib-mikhmon/domain"
	"github.com/quiqxiq/roslib-mikhmon/mikrotik"
)

// Konstanta filter comment yang dipakai mikhmon (analisis §1.3).
const (
	// CommentTransaction adalah comment marker untuk script transaksi
	// yang dibuat on-login (analisis §3.1).
	CommentTransaction = "mikhmon"

	// CommentQuickPrint adalah comment marker untuk template Quick Print.
	CommentQuickPrint = "QuickPrintMikhmon"
)

const scriptPath = "/system/script"

// ScriptList → /system/script/print (analisis §1.3).
func (c *Client) ScriptList(ctx context.Context) ([]domain.Script, error) {
	reply, err := c.dev.Path(scriptPath).Print().Exec(ctx)
	if err != nil {
		return nil, err
	}
	return sentencesToScripts(reply.Rows), nil
}

// ScriptByID → /system/script/print ?.id=<id>.
func (c *Client) ScriptByID(ctx context.Context, id string) (domain.Script, error) {
	if id == "" {
		return domain.Script{}, mikrotik.ErrInvalidArgument
	}
	reply, err := c.dev.Path(scriptPath).Print().Where(".id", id).Exec(ctx)
	if err != nil {
		return domain.Script{}, err
	}
	if len(reply.Rows) == 0 {
		return domain.Script{}, mikrotik.ErrNotFound
	}
	return sentenceToScript(reply.Rows[0]), nil
}

// ScriptByName → /system/script/print ?name=<name>. Mengembalikan list,
// karena nama script transaksi tidak unik secara prinsip.
func (c *Client) ScriptByName(ctx context.Context, name string) ([]domain.Script, error) {
	if name == "" {
		return nil, mikrotik.ErrInvalidArgument
	}
	reply, err := c.dev.Path(scriptPath).Print().Where("name", name).Exec(ctx)
	if err != nil {
		return nil, err
	}
	return sentencesToScripts(reply.Rows), nil
}

// ScriptByComment → /system/script/print ?comment=<value>.
//
// Dipakai untuk:
//   - filter ?comment=mikhmon → semua transaksi (analisis §1.3).
//   - filter ?comment=QuickPrintMikhmon → daftar Quick Print template.
func (c *Client) ScriptByComment(ctx context.Context, comment string) ([]domain.Script, error) {
	if comment == "" {
		return nil, mikrotik.ErrInvalidArgument
	}
	reply, err := c.dev.Path(scriptPath).Print().Where("comment", comment).Exec(ctx)
	if err != nil {
		return nil, err
	}
	return sentencesToScripts(reply.Rows), nil
}

// ScriptByOwner → /system/script/print ?owner=<value>.
//
// Mikhmon memakai owner = "<bulan><tahun>" (mis. "Jan2025") untuk
// laporan transaksi bulanan (analisis §1.3).
func (c *Client) ScriptByOwner(ctx context.Context, owner string) ([]domain.Script, error) {
	if owner == "" {
		return nil, mikrotik.ErrInvalidArgument
	}
	reply, err := c.dev.Path(scriptPath).Print().Where("owner", owner).Exec(ctx)
	if err != nil {
		return nil, err
	}
	return sentencesToScripts(reply.Rows), nil
}

// ScriptBySource → /system/script/print ?source=<value>.
//
// Mikhmon memakai source = "<date>" (mis. "jan/05/2025") untuk
// laporan transaksi hari ini (analisis §1.3).
func (c *Client) ScriptBySource(ctx context.Context, source string) ([]domain.Script, error) {
	if source == "" {
		return nil, mikrotik.ErrInvalidArgument
	}
	reply, err := c.dev.Path(scriptPath).Print().Where("source", source).Exec(ctx)
	if err != nil {
		return nil, err
	}
	return sentencesToScripts(reply.Rows), nil
}

// ScriptAddArgs adalah parameter ScriptAdd. Kosong → tidak dikirim.
type ScriptAddArgs struct {
	Name    string // wajib
	Source  string // wajib
	Comment string
	Owner   string
	Policy  string // mis. "read,write,policy,test"
}

// ScriptAdd → /system/script/add (analisis §1.3). Mengembalikan .id baru.
func (c *Client) ScriptAdd(ctx context.Context, a ScriptAddArgs) (string, error) {
	if a.Name == "" || a.Source == "" {
		return "", mikrotik.ErrInvalidArgument
	}
	pairs := []roslib.Pair{
		roslib.NewPair("name", a.Name),
		roslib.NewPair("source", a.Source),
	}
	if a.Comment != "" {
		pairs = append(pairs, roslib.NewPair("comment", a.Comment))
	}
	if a.Owner != "" {
		pairs = append(pairs, roslib.NewPair("owner", a.Owner))
	}
	if a.Policy != "" {
		pairs = append(pairs, roslib.NewPair("policy", a.Policy))
	}
	reply, err := c.dev.Path(scriptPath).Add(ctx, pairs...)
	if err != nil {
		return "", err
	}
	if reply.Done == nil {
		return "", nil
	}
	return reply.Done.Map["ret"], nil
}

// ScriptSetArgs adalah parameter ScriptSet. Field Kosong → tidak diubah.
type ScriptSetArgs struct {
	ID      string // wajib
	Name    string
	Source  string
	Comment string
	Owner   string
}

// ScriptSet → /system/script/set (analisis §1.3).
func (c *Client) ScriptSet(ctx context.Context, a ScriptSetArgs) error {
	if a.ID == "" {
		return mikrotik.ErrInvalidArgument
	}
	pairs := []roslib.Pair{}
	if a.Name != "" {
		pairs = append(pairs, roslib.NewPair("name", a.Name))
	}
	if a.Source != "" {
		pairs = append(pairs, roslib.NewPair("source", a.Source))
	}
	if a.Comment != "" {
		pairs = append(pairs, roslib.NewPair("comment", a.Comment))
	}
	if a.Owner != "" {
		pairs = append(pairs, roslib.NewPair("owner", a.Owner))
	}
	_, err := c.dev.Path(scriptPath).Set(ctx, a.ID, pairs...)
	return err
}

// ScriptRemove → /system/script/remove (analisis §1.3).
func (c *Client) ScriptRemove(ctx context.Context, id string) error {
	if id == "" {
		return mikrotik.ErrInvalidArgument
	}
	_, err := c.dev.Path(scriptPath).Remove(ctx, id)
	return err
}

// ───────────── helpers ─────────────

func sentenceToScript(s *roslib.Sentence) domain.Script {
	return domain.Script{
		ID:      s.Get(".id"),
		Name:    s.Get("name"),
		Owner:   s.Get("owner"),
		Source:  s.Get("source"),
		Comment: s.Get("comment"),
		Policy:  s.Get("policy"),
	}
}

func sentencesToScripts(rows []*roslib.Sentence) []domain.Script {
	out := make([]domain.Script, 0, len(rows))
	for _, r := range rows {
		out = append(out, sentenceToScript(r))
	}
	return out
}
