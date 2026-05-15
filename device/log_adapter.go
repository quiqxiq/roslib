package device

import (
	"log/slog"

	slogrus "github.com/samber/slog-logrus/v2"
	"github.com/sirupsen/logrus"
)

// NewSlogHandlerFromLogrus membungkus *logrus.Logger menjadi slog.Handler
// yang sesuai untuk routeros.Client.SetLogHandler.
//
// Level dipasang ke Debug supaya filter benar-benar dilakukan oleh logrus
// (slog tidak men-drop apa-apa lebih dulu) — perilaku yang diharapkan saat
// user pakai logrus.SetLevel di luar library.
func NewSlogHandlerFromLogrus(l *logrus.Logger) slog.Handler {
	return slogrus.Option{
		Level:  slog.LevelDebug,
		Logger: l,
	}.NewLogrusHandler()
}
