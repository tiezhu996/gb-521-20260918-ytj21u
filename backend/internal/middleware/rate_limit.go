package middleware

import (
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"

	"mine-ventilation-network-simulator/backend/pkg/api"
)

type rateWindow struct {
	Started time.Time
	Count   int
}

type RateLimiter struct {
	mu      sync.Mutex
	windows map[string]rateWindow
	limit   int
	window  time.Duration
	scope   string
}

func NewRateLimiter(scope string, limit int, window time.Duration) *RateLimiter {
	return &RateLimiter{windows: make(map[string]rateWindow), limit: limit, window: window, scope: scope}
}

func (l *RateLimiter) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		now := time.Now()
		key := l.scope + ":" + c.ClientIP()
		l.mu.Lock()
		entry, exists := l.windows[key]
		if !exists || now.Sub(entry.Started) >= l.window {
			entry = rateWindow{Started: now}
		}
		entry.Count++
		l.windows[key] = entry
		remaining := l.limit - entry.Count
		reset := entry.Started.Add(l.window)
		if len(l.windows) > 2048 {
			l.pruneLocked(now)
		}
		l.mu.Unlock()
		if remaining < 0 {
			remaining = 0
		}
		c.Header("X-RateLimit-Limit", strconv.Itoa(l.limit))
		c.Header("X-RateLimit-Remaining", strconv.Itoa(remaining))
		c.Header("X-RateLimit-Reset", strconv.FormatInt(reset.Unix(), 10))
		if entry.Count > l.limit {
			c.Header("Retry-After", strconv.Itoa(int(time.Until(reset).Seconds())+1))
			api.Fail(c, api.NewError(429, "RATE_LIMITED", "请求过于频繁，请稍后再试", nil))
			return
		}
		c.Next()
	}
}

func (l *RateLimiter) pruneLocked(now time.Time) {
	for key, entry := range l.windows {
		if now.Sub(entry.Started) >= l.window*2 {
			delete(l.windows, key)
		}
	}
}
