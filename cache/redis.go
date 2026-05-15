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

// RedisCache mengimplementasikan Cache + PathAwareCache di atas go-redis.
//
// Path tracking: untuk tiap path, sebuah Redis SET dengan nama
// "roslib:idx:<path>" menyimpan daftar key cache yang berasal dari path
// tersebut. InvalidatePath melakukan SMEMBERS → DEL keys + DEL idx-set
// dalam satu pipeline.
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

func (r *RedisCache) SetForPath(ctx context.Context, path, key string, val []byte, ttl time.Duration) error {
	pipe := r.client.TxPipeline()
	pipe.Set(ctx, key, val, ttl)
	if path != "" {
		idx := "roslib:idx:" + path
		pipe.SAdd(ctx, idx, key)
		if ttl > 0 {
			// Index TTL slightly longer dari entry TTL agar invalidate tidak
			// kehilangan reference saat entry expire alami sebelum index.
			pipe.Expire(ctx, idx, ttl+time.Minute)
		}
	}
	_, err := pipe.Exec(ctx)
	return err
}

func (r *RedisCache) InvalidatePath(ctx context.Context, path string) error {
	if path == "" {
		return nil
	}
	idx := "roslib:idx:" + path
	keys, err := r.client.SMembers(ctx, idx).Result()
	if err != nil {
		return err
	}
	if len(keys) == 0 {
		return r.client.Del(ctx, idx).Err()
	}
	pipe := r.client.TxPipeline()
	pipe.Del(ctx, keys...)
	pipe.Del(ctx, idx)
	_, err = pipe.Exec(ctx)
	return err
}

// InvalidatePathForDevice menghapus entry path hanya untuk device tertentu.
// Implementasi: SMEMBERS idx → filter prefix "roslib:<deviceID>:" → SREM + DEL.
func (r *RedisCache) InvalidatePathForDevice(ctx context.Context, deviceID, path string) error {
	if path == "" || deviceID == "" {
		return nil
	}
	idx := "roslib:idx:" + path
	keys, err := r.client.SMembers(ctx, idx).Result()
	if err != nil {
		return err
	}
	prefix := "roslib:" + deviceID + ":"
	scoped := make([]string, 0, len(keys))
	for _, k := range keys {
		if len(k) >= len(prefix) && k[:len(prefix)] == prefix {
			scoped = append(scoped, k)
		}
	}
	if len(scoped) == 0 {
		return nil
	}
	pipe := r.client.TxPipeline()
	pipe.Del(ctx, scoped...)
	// SREM accepts variadic any, but old API uses []string via SRem with values.
	args := make([]any, len(scoped))
	for i, k := range scoped {
		args[i] = k
	}
	pipe.SRem(ctx, idx, args...)
	_, err = pipe.Exec(ctx)
	return err
}

// Close menutup koneksi Redis.
func (r *RedisCache) Close() error { return r.client.Close() }
