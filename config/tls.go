package config

import "crypto/tls"

// newTLSConfig membuat *tls.Config minimal untuk koneksi RouterOS.
// User boleh build tls.Config sendiri dan inject langsung lewat
// device.Options jika butuh trust store custom atau client cert.
func newTLSConfig(insecure bool) *tls.Config {
	return &tls.Config{
		InsecureSkipVerify: insecure, //nolint:gosec // explicit opt-in via config
		MinVersion:         tls.VersionTLS12,
	}
}
