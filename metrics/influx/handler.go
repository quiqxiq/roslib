package influx

import (
	"context"

	"github.com/quiqxiq/roslib/decode"
	"github.com/quiqxiq/roslib/poll"
	"github.com/quiqxiq/roslib/stream"
	"github.com/sirupsen/logrus"
)

// PollSink mengembalikan poll.Handler yang menulis tiap sentence ke Writer.
// Logger boleh nil; kalau diisi, error tulis akan di-log dengan field
// influx_measurement supaya mudah filter di centralized logging.
func PollSink(w *Writer, log *logrus.Entry) poll.Handler {
	return func(s *decode.Sentence) {
		if err := w.WriteSentence(context.Background(), s); err != nil && log != nil {
			log.WithError(err).WithField("influx_measurement", w.meas).
				Warn("influx write failed")
		}
	}
}

// StreamSink mirror PollSink untuk listener.
func StreamSink(w *Writer, log *logrus.Entry) stream.Handler {
	return func(s *decode.Sentence) {
		if err := w.WriteSentence(context.Background(), s); err != nil && log != nil {
			log.WithError(err).WithField("influx_measurement", w.meas).
				Warn("influx write failed")
		}
	}
}

// BatchedPollSink mirip PollSink tapi pakai BatchedWriter — sentence
// di-buffer, flush oleh goroutine internal BatchedWriter.
func BatchedPollSink(bw *BatchedWriter) poll.Handler {
	return func(s *decode.Sentence) { bw.AddSentence(s) }
}

// BatchedStreamSink mirror BatchedPollSink untuk listener.
func BatchedStreamSink(bw *BatchedWriter) stream.Handler {
	return func(s *decode.Sentence) { bw.AddSentence(s) }
}
