// Package mikrotik adalah thin wrapper command di atas *roslib.Device
// untuk semua RouterOS API path yang dipakai mikhmonv3.
//
// Boundary:
//
//   - Setiap sub-paket (hotspot, system, ppp, network, syslog) menyediakan
//     Client yang dikonstruksi dengan New(dev *roslib.Device). Method
//     mengembalikan domain types dari roslib-mikhmon/domain.
//   - Query snapshot pakai dev.Path(path).Print()...Exec(ctx).
//   - Mutation pakai dev.Path(path).Add/Set/Remove/Enable/Disable.
//   - Streaming (Print follow / inherent stream) dan polling tersedia
//     sebagai method khusus per resource (mis. hotspot.Client.ActiveStream,
//     system.Client.MonitorResource).
//   - Decode hasil pakai *decode.Sentence dari roslib (Get/Bool/Int/Bytes/
//     Duration/Time + Or-variants).
//
// Tidak ada business logic di paket ini. Cascade orchestration ada di
// paket workflows/.
package mikrotik
