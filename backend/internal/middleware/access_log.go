package middleware

import (
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"
)

func AccessLogMiddleware(logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		started := time.Now()
		path := c.Request.URL.Path
		c.Next()
		logger.InfoContext(c.Request.Context(), "http_request",
			"request_id", c.GetString("request_id"),
			"method", c.Request.Method,
			"path", path,
			"status", c.Writer.Status(),
			"latency_ms", time.Since(started).Milliseconds(),
			"client_ip", c.ClientIP(),
			"bytes", c.Writer.Size(),
		)
	}
}
