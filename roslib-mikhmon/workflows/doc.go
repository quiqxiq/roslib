// Package workflows berisi orchestrator multi-step ("cascade") yang
// menggabungkan beberapa command dari paket mikrotik/* sesuai alur yang
// dideskripsikan di analisis §4.
//
// Konvensi:
//
//   - Workflow tidak menyimpan state — function pure terhadap *Clients.
//   - Error pada satu step menghentikan cascade (fail-fast). Step yang
//     sudah berhasil tidak di-rollback (RouterOS API tidak transactional).
//   - Workflow di-test secara live ke router via build tag `integration`.
package workflows

import (
	"github.com/quiqxiq/roslib"
	"github.com/quiqxiq/roslib-mikhmon/mikrotik/hotspot"
	"github.com/quiqxiq/roslib-mikhmon/mikrotik/network"
	"github.com/quiqxiq/roslib-mikhmon/mikrotik/system"
)

// Clients adalah fasade yang menggabungkan sub-client mikrotik untuk
// kebutuhan workflow. Dibangun dari satu *roslib.Device — semua sub-client
// share koneksi async + tag demux yang sama.
type Clients struct {
	System  *system.Client
	Hotspot *hotspot.Client
	Network *network.Client
}

// New membangun Clients dari *roslib.Device. Caller biasanya dapat dev
// dari roslib.NewFromConfig → mgr.Get(roslib.DefaultDeviceKey).
func New(dev *roslib.Device) *Clients {
	return &Clients{
		System:  system.New(dev),
		Hotspot: hotspot.New(dev),
		Network: network.New(dev),
	}
}
