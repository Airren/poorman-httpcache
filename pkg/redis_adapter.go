/*
MIT License

Copyright (c) 2018 Victor Springer

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
*/

package pkg

import (
	"context"
	"time"

	redisCache "github.com/go-redis/cache/v9"
	"github.com/redis/go-redis/v9"
	"github.com/vmihailenco/msgpack/v5"
)

// RedisAdapter is the memory adapter data structure.
type RedisAdapter struct {
	store *redisCache.Cache
}

// Get implements the cache Adapter interface Get method.
func (ra *RedisAdapter) Get(ctx context.Context, key uint64) ([]byte, bool) {
	var c []byte
	if err := ra.store.Get(ctx, KeyAsString(key), &c); err == nil {
		return c, true
	}

	return nil, false
}

// Set implements the cache Adapter interface Set method.
func (ra *RedisAdapter) Set(key uint64, response []byte, expiration time.Time) {
	ra.store.Set(&redisCache.Item{
		Key:   KeyAsString(key),
		Value: response,
		TTL:   time.Until(expiration),
	})
}

// Release implements the cache Adapter interface Release method.
func (ra *RedisAdapter) Release(ctx context.Context, key uint64) {
	ra.store.Delete(ctx, KeyAsString(key))
}

// NewRedisAdapter initializes Redis adapter
func NewRedisAdapter(opt *redis.RingOptions) *RedisAdapter {
	ring := redis.NewRing(opt)
	store := redisCache.New(&redisCache.Options{
		Redis: ring,
		Marshal: func(v interface{}) ([]byte, error) {
			return msgpack.Marshal(v)
		},
		Unmarshal: func(b []byte, v interface{}) error {
			return msgpack.Unmarshal(b, v)
		},
		LocalCache: redisCache.NewTinyLFU(1000, 10*time.Minute),
	})
	return &RedisAdapter{
		store: store,
	}
}
