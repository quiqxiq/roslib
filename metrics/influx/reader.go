package influx

import (
	"context"

	"github.com/InfluxCommunity/influxdb3-go/v2/influxdb3"
)

// Reader adalah wrapper tipis di atas Client.Query agar user
// melihat satu permukaan import yang sama untuk read & write.
type Reader struct {
	cli *influxdb3.Client
}

// NewReader membungkus client untuk read API.
func NewReader(cli *influxdb3.Client) *Reader {
	return &Reader{cli: cli}
}

// Query menjalankan SQL query (default) atau InfluxQL (lewat opts).
// Iterator dikonsumsi pemanggil: Next() / Value() / AsPoints().
func (r *Reader) Query(ctx context.Context, sql string, opts ...influxdb3.QueryOption) (*influxdb3.QueryIterator, error) {
	return r.cli.Query(ctx, sql, opts...)
}

// QueryWithParameters menjalankan query parameterized.
func (r *Reader) QueryWithParameters(ctx context.Context, sql string, params influxdb3.QueryParameters, opts ...influxdb3.QueryOption) (*influxdb3.QueryIterator, error) {
	return r.cli.QueryWithParameters(ctx, sql, params, opts...)
}
