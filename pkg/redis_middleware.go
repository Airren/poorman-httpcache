package pkg

import (
	"net/http"

	"github.com/go-redis/cache/v9"
)

func NewRedisMiddleware(redisCache *cache.Cache) Middleware {
	// TODO: acutally use the redisCache
	return func(next http.RoundTripper) http.RoundTripper {
		return next
	}
}
