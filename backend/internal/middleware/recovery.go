package middleware

import (
	"fmt"
	"log/slog"

	"github.com/gin-gonic/gin"

	"mine-ventilation-network-simulator/backend/pkg/api"
)

func RecoveryMiddleware(logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if recovered := recover(); recovered != nil {
				logger.ErrorContext(c.Request.Context(), "panic_recovered",
					"request_id", c.GetString("request_id"),
					"path", c.Request.URL.Path,
					"panic", fmt.Sprint(recovered),
				)
				api.Fail(c, api.Internal(fmt.Errorf("panic recovered: %v", recovered)))
			}
		}()
		c.Next()
	}
}
