// Package builder menyediakan fluent API di atas RouterDevice:
//
//	device.Path("/ip/address").Print().Where(...).Exec(ctx)
//	device.Path("/ip/address").Add(ctx, query.NewPair("address", "10.0.0.1/24"), ...)
//	device.Path("/log").Print().FollowOnly().Stream("log-tail", handler)
//	device.Path("/interface/monitor-traffic").With("interface","ether1").Stream("nic", handler)
//
// Builder tidak meng-import device langsung — ia bekerja dengan Executor
// interface yang di-implementasikan device. Ini menghindari cyclic import
// sekaligus memudahkan unit test.
package builder

import (
	"context"
	"time"

	"github.com/go-routeros/routeros/v3"
	"github.com/quiqxiq/roslib/cache"
	"github.com/quiqxiq/roslib/capability"
	"github.com/quiqxiq/roslib/stream"
	"github.com/sirupsen/logrus"
)

// Executor adalah kontrak yang harus dipenuhi oleh pemilik builder
// (umumnya *device.RouterDevice). Builder tidak peduli detail koneksi —
// ia hanya butuh cara untuk:
//
//  1. Mengirim command satu-shot (RunCommand).
//  2. Mendaftarkan listener jangka panjang (RegisterStream).
//  3. Membatalkan listener (UnregisterStream).
//  4. Akses ke registry+cache+config untuk validasi & caching.
type Executor interface {
	RunCommand(ctx context.Context, sentence []string) (*routeros.Reply, error)
	RegisterStream(spec stream.Spec) error
	UnregisterStream(id string) bool

	// Registry mengembalikan capability registry. nil → builder skip validasi.
	Registry() *capability.Registry

	// Cache mengembalikan instance Cache (NoopCache kalau disabled).
	Cache() cache.Cache

	// CacheTTL adalah default TTL untuk ExecCached saat caller pakai 0.
	CacheTTL() time.Duration

	// Strict melaporkan mode validator (true=error, false=log-warn).
	Strict() bool

	// Logger untuk log-warn saat strict=false.
	Logger() *logrus.Entry

	// DeviceID identifier unik device — dipakai untuk scope key cache
	// di lingkungan multi-router. Single-router cukup kembalikan address
	// atau string kosong.
	DeviceID() string
}
