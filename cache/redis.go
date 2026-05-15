//go:build redis

// File ini hanya dikompilasi dengan `-tags=redis`. Lihat README.
//
// Skeleton implementasi Cache di atas github.com/redis/go-redis/v9.
// Untuk pakainya: `go get github.com/redis/go-redis/v9` di app, lalu
// `go build -tags=redis ./...`.

package cache

import (
	"context"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisOptions adalah parameter koneksi untuk RedisCache.
type RedisOptions struct {
	Addr     string
	Password string
	DB       int
}

// RedisCache mengimplementasikan Cache di atas go-redis.
type RedisCache struct {
	client *redis.Client
}

// NewRedis membuat RedisCache dan melakukan PING untuk verifikasi koneksi.
func NewRedis(opts RedisOptions) (*RedisCache, error) {
	c := redis.NewClient(&redis.Options{
		Addr:     opts.Addr,
		Password: opts.Password,
		DB:       opts.DB,
	})
	if err := c.Ping(context.Background()).Err(); err != nil {
		return nil, err
	}
	return &RedisCache{client: c}, nil
}

func (r *RedisCache) Get(ctx context.Context, key string) ([]byte, bool, error) {
	val, err := r.client.Get(ctx, key).Bytes()
	if errors.Is(err, redis.Nil) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	return val, true, nil
}

func (r *RedisCache) Set(ctx context.Context, key string, val []byte, ttl time.Duration) error {
	return r.client.Set(ctx, key, val, ttl).Err()
}

// Close menutup koneksi Redis.
func (r *RedisCache) Close() error { return r.client.Close() }
