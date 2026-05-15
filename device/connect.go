package device

import (
	"context"
	"fmt"

	"github.com/go-routeros/routeros/v3"
)

// dialResult menyatukan koneksi dan channel error async-loop-nya.
// Channel asyncErr emit sekali kalau loop async mati (TCP putus, !fatal,
// atau context cancel). Supervisor goroutine men-block di channel ini.
type dialResult struct {
	conn     *routeros.Client
	asyncErr <-chan error
}

// dialOne membuka koneksi baru, men-set log handler, dan langsung
// mengaktifkan mode async. Role hanya untuk konteks logging.
func (d *RouterDevice) dialOne(role string) (*dialResult, error) {
	log := d.log.WithField("conn", role)
	log.Debug("dialing")

	dialCtx, cancel := context.WithTimeout(d.ctx, d.opts.dialTimeout())
	defer cancel()

	var (
		c   *routeros.Client
		err error
	)
	if d.opts.TLS != nil {
		c, err = routeros.DialTLSContext(dialCtx, d.opts.Address, d.opts.Username, d.opts.Password, d.opts.TLS)
	} else {
		c, err = routeros.DialContext(dialCtx, d.opts.Address, d.opts.Username, d.opts.Password)
	}
	if err != nil {
		return nil, fmt.Errorf("dial %s: %w", role, err)
	}

	c.Queue = d.opts.listenQueueSize()
	c.SetLogHandler(NewSlogHandlerFromLogrus(d.opts.Logger))
	asyncErr := c.AsyncContext(d.ctx)

	log.Info("connection established (async mode)")
	return &dialResult{conn: c, asyncErr: asyncErr}, nil
}

// dialBoth membuka kedua koneksi (stream + command) di awal hidup device.
// Kalau salah satu gagal, koneksi yang sudah jadi langsung ditutup.
func (d *RouterDevice) dialBoth() error {
	streamRes, err := d.dialOne("stream")
	if err != nil {
		return fmt.Errorf("dial connStream: %w", err)
	}

	cmdRes, err := d.dialOne("command")
	if err != nil {
		_ = streamRes.conn.Close()
		return fmt.Errorf("dial connCommand: %w", err)
	}

	d.mu.Lock()
	d.connStream = streamRes.conn
	d.streamAsyncErr = streamRes.asyncErr
	d.connCommand = cmdRes.conn
	d.commandAsyncErr = cmdRes.asyncErr
	d.mu.Unlock()
	return nil
}
