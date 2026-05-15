// Package device menyediakan RouterDevice: 2 koneksi async persisten ke
// satu router RouterOS (1 untuk stream/listen, 1 untuk query+mutation+poll),
// supervisor reconnect otomatis, dan integrasi ke StreamManager + PollEngine.
package device

import (
	"crypto/tls"
	"time"

	"github.com/quiqxiq/roslib/cache"
	"github.com/quiqxiq/roslib/capability"
	"github.com/sirupsen/logrus"
)

// Options menampung parameter konfigurasi RouterDevice.
//
// Address: "host:port", typical "192.168.88.1:8728" (TCP) atau ":8729" (TLS).
// Logger: logrus logger user; internal go-routeros di-redirect lewat adapter.
// TLS: kalau non-nil, dial dengan TLS.
// ListenQueueSize: channel buffer untuk ListenReply.Chan(). 0 = pakai default
//
//	(routeros.Client.Queue zero = unbounded? — di v3 zero artinya make(chan,0)).
//	100 adalah default aman.
//
// DialTimeout: timeout per attempt dial. 0 = pakai default 10 detik.
// ReconnectMaxElapsed: total waktu max untuk retry reconnect. 0 = tak terbatas
//
//	(retry selamanya sampai context cancel).
type Options struct {
	Address  string
	Username string
	Password string

	Logger *logrus.Logger
	TLS    *tls.Config

	ListenQueueSize     int
	DialTimeout         time.Duration
	ReconnectMaxElapsed time.Duration

	// ReconnectInitialInterval awal backoff. 0 = default 500ms.
	ReconnectInitialInterval time.Duration
	// ReconnectMaxInterval cap atas backoff. 0 = default 30s.
	ReconnectMaxInterval time.Duration

	// Registry adalah capability registry untuk validasi command/arg di
	// builder. nil = tanpa validasi (semua command lewat tanpa cek).
	Registry *capability.Registry

	// Cache di-share lewat Executor untuk PrintBuilder.ExecCached.
	// nil = NoopCache (akan diisi otomatis oleh New).
	Cache cache.Cache

	// CacheTTL default untuk ExecCached saat caller pakai TTL 0.
	CacheTTL time.Duration

	// StrictCapability: kalau true (default), validator return error Go
	// saat command/arg/class invalid. False = hanya log warning.
	StrictCapability bool
}

func (o *Options) listenQueueSize() int {
	if o.ListenQueueSize > 0 {
		return o.ListenQueueSize
	}
	return 100
}

func (o *Options) dialTimeout() time.Duration {
	if o.DialTimeout > 0 {
		return o.DialTimeout
	}
	return 10 * time.Second
}

func (o *Options) reconnectInitial() time.Duration {
	if o.ReconnectInitialInterval > 0 {
		return o.ReconnectInitialInterval
	}
	return 500 * time.Millisecond
}

func (o *Options) reconnectMax() time.Duration {
	if o.ReconnectMaxInterval > 0 {
		return o.ReconnectMaxInterval
	}
	return 30 * time.Second
}
