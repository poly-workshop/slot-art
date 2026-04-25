package middleware

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/redis/go-redis/v9"
)

type RateLimitConfig struct {
	FingerprintLimit int
	TaskLimit        int
	FingerprintTTL   time.Duration
	TaskTTL          time.Duration
}

func DefaultRateLimitConfig() RateLimitConfig {
	return RateLimitConfig{
		FingerprintLimit: 60,
		TaskLimit:        5,
		FingerprintTTL:   time.Minute,
		TaskTTL:          5 * time.Minute,
	}
}

func RateLimiter(rdb *redis.Client, cfg RateLimitConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fp := r.Header.Get("X-Slot-Fingerprint")
			if fp != "" {
				key := "sa:ratelimit:" + fp + ":requests"
				n, err := rdb.Incr(r.Context(), key).Result()
				if err == nil && n == 1 {
					rdb.Expire(r.Context(), key, cfg.FingerprintTTL)
				}
				if err == nil && n > int64(cfg.FingerprintLimit) {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusTooManyRequests)
					json.NewEncoder(w).Encode(map[string]string{"error": "rate limit exceeded"})
					return
				}
			}

			uid := r.Header.Get("X-Slot-UID")
			if uid != "" {
				key := "sa:ratelimit:" + uid + ":tasks"
				if r.Method == http.MethodPost && r.URL.Path == "/api/v1/tasks" {
					n, err := rdb.Incr(r.Context(), key).Result()
					if err == nil && n == 1 {
						rdb.Expire(r.Context(), key, cfg.TaskTTL)
					}
					if err == nil && n > int64(cfg.TaskLimit) {
						w.Header().Set("Content-Type", "application/json")
						w.WriteHeader(http.StatusTooManyRequests)
						json.NewEncoder(w).Encode(map[string]string{"error": "task submission rate limit exceeded"})
						return
					}
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}
