// Package influx menyediakan integrasi ringkas dengan InfluxDB3:
//
//   - NewClient(InfluxConfig) → *influxdb3.Client (wrap SDK upstream)
//   - Writer untuk sink hasil poll/stream ke series InfluxDB
//   - BatchedWriter untuk buffering opt-in
//   - Reader tipis di atas Query
//   - PollSink / StreamSink helper untuk wiring satu baris
package influx

import (
	"errors"

	"github.com/InfluxCommunity/influxdb3-go/v2/influxdb3"
	"github.com/quiqxiq/roslib/config"
)

// NewClient membangun *influxdb3.Client dari InfluxConfig.
//
// Tidak melakukan ping/healthcheck — biarkan operasi pertama yang gagal.
// User wajib panggil Close() saat shutdown supaya idle HTTP connection
// & Flight client di-release.
func NewClient(cfg config.InfluxConfig) (*influxdb3.Client, error) {
	if !cfg.Enabled {
		return nil, errors.New("influx: cfg.Enabled is false")
	}
	if cfg.Host == "" || cfg.Token == "" || cfg.Database == "" {
		return nil, errors.New("influx: Host, Token, Database are required")
	}
	return influxdb3.New(influxdb3.ClientConfig{
		Host:         cfg.Host,
		Token:        cfg.Token,
		Database:     cfg.Database,
		Organization: cfg.Organization,
	})
}
