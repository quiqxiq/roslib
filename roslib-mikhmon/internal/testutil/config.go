package testutil

import (
	"github.com/quiqxiq/roslib/config"
)

// loadConfigFromEnv adalah wrapper tipis di atas config.LoadFromEnv.
// Ada di sini supaya gampang di-stub kalau nanti perlu override.
func loadConfigFromEnv() (*config.Config, error) {
	return config.LoadFromEnv()
}
