package device

import (
	"github.com/cenkalti/backoff/v4"
)

// superviseCommand menunggu di asyncErr channel connCommand. Saat koneksi
// mati, redial dengan exponential backoff dan re-attach poll engine.
//
// Channel asyncErr emit MAKSIMAL satu error lalu di-close oleh go-routeros.
// Setiap reconnect kita refresh channel reference dari koneksi baru.
func (d *RouterDevice) superviseCommand() {
	for {
		d.mu.RLock()
		ch := d.commandAsyncErr
		d.mu.RUnlock()

		select {
		case <-d.ctx.Done():
			return
		case err, ok := <-ch:
			if !ok && err == nil {
				// channel closed normally karena context cancel.
				return
			}
			if d.ctx.Err() != nil {
				return
			}
			d.log.WithError(err).Warn("connCommand died, reconnecting")
			if rerr := d.reconnectCommand(); rerr != nil {
				d.log.WithError(rerr).Error("connCommand reconnect aborted")
				return
			}
		}
	}
}

// superviseStream mirror superviseCommand untuk koneksi listener.
// Bedanya: setelah reconnect kita panggil ReattachAll agar semua spec
// listener didaftarkan ulang di koneksi baru.
func (d *RouterDevice) superviseStream() {
	for {
		d.mu.RLock()
		ch := d.streamAsyncErr
		d.mu.RUnlock()

		select {
		case <-d.ctx.Done():
			return
		case err, ok := <-ch:
			if !ok && err == nil {
				return
			}
			if d.ctx.Err() != nil {
				return
			}
			d.log.WithError(err).Warn("connStream died, reconnecting")
			if rerr := d.reconnectStream(); rerr != nil {
				d.log.WithError(rerr).Error("connStream reconnect aborted")
				return
			}
		}
	}
}

func (d *RouterDevice) reconnectCommand() error {
	bo := d.newBackoff()
	op := func() error {
		res, err := d.dialOne("command")
		if err != nil {
			d.log.WithError(err).Warn("redial connCommand failed")
			return err
		}
		d.mu.Lock()
		if d.connCommand != nil {
			_ = d.connCommand.Close()
		}
		d.connCommand = res.conn
		d.commandAsyncErr = res.asyncErr
		d.mu.Unlock()

		d.polls.AttachConn(res.conn)
		d.log.Info("connCommand reconnected")
		return nil
	}
	return backoff.Retry(op, backoff.WithContext(bo, d.ctx))
}

func (d *RouterDevice) reconnectStream() error {
	bo := d.newBackoff()
	op := func() error {
		res, err := d.dialOne("stream")
		if err != nil {
			d.log.WithError(err).Warn("redial connStream failed")
			return err
		}
		d.mu.Lock()
		if d.connStream != nil {
			_ = d.connStream.Close()
		}
		d.connStream = res.conn
		d.streamAsyncErr = res.asyncErr
		d.mu.Unlock()

		d.streams.ReattachAll(res.conn)
		d.log.Info("connStream reconnected")
		return nil
	}
	return backoff.Retry(op, backoff.WithContext(bo, d.ctx))
}

func (d *RouterDevice) newBackoff() *backoff.ExponentialBackOff {
	bo := backoff.NewExponentialBackOff()
	bo.InitialInterval = d.opts.reconnectInitial()
	bo.MaxInterval = d.opts.reconnectMax()
	bo.MaxElapsedTime = d.opts.ReconnectMaxElapsed // 0 = unlimited
	return bo
}
