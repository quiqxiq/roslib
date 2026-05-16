package builder

import (
	"context"

	"github.com/go-routeros/routeros/v3"
	"github.com/quiqxiq/roslib/query"
)

// Add menjalankan command "/path/add" dengan named pairs sebagai parameter.
func (b *PathBuilder) Add(ctx context.Context, pairs ...query.Pair) (*routeros.Reply, error) {
	sentence := query.BuildSentence(b.path+"/add", nil, pairs, nil)
	return b.exec.RunCommand(ctx, sentence)
}

// Set menjalankan "/path/set" untuk satu numbers (ID atau .id) ditambah pairs.
// numbers boleh kosong untuk path yang menerima set tanpa ID.
func (b *PathBuilder) Set(ctx context.Context, numbers string, pairs ...query.Pair) (*routeros.Reply, error) {
	all := append([]query.Pair(nil), pairs...)
	if numbers != "" {
		all = append([]query.Pair{query.NewPair("numbers", numbers)}, all...)
	}
	sentence := query.BuildSentence(b.path+"/set", nil, all, nil)
	return b.exec.RunCommand(ctx, sentence)
}

// Remove menjalankan "/path/remove" untuk numbers tertentu.
func (b *PathBuilder) Remove(ctx context.Context, numbers string) (*routeros.Reply, error) {
	pairs := []query.Pair{query.NewPair("numbers", numbers)}
	sentence := query.BuildSentence(b.path+"/remove", nil, pairs, nil)
	return b.exec.RunCommand(ctx, sentence)
}

// Enable menjalankan "/path/enable" untuk numbers tertentu.
func (b *PathBuilder) Enable(ctx context.Context, numbers string) (*routeros.Reply, error) {
	return b.toggle(ctx, "enable", numbers)
}

// Disable menjalankan "/path/disable" untuk numbers tertentu.
func (b *PathBuilder) Disable(ctx context.Context, numbers string) (*routeros.Reply, error) {
	return b.toggle(ctx, "disable", numbers)
}

func (b *PathBuilder) toggle(ctx context.Context, action, numbers string) (*routeros.Reply, error) {
	pairs := []query.Pair{query.NewPair("numbers", numbers)}
	sentence := query.BuildSentence(b.path+"/"+action, nil, pairs, nil)
	return b.exec.RunCommand(ctx, sentence)
}

// Run mengeksekusi sentence apapun pada path ini secara langsung.
// Berguna untuk command yang tidak punya helper khusus, mis. "ping" atau
// "monitor-once". Action adalah kata setelah path (tanpa "/" awal).
func (b *PathBuilder) Run(ctx context.Context, action string, pairs ...query.Pair) (*routeros.Reply, error) {
	sentence := query.BuildSentence(b.path+"/"+action, nil, pairs, nil)
	return b.exec.RunCommand(ctx, sentence)
}
