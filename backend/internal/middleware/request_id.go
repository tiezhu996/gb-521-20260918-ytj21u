package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const requestIDHeader = "X-Request-ID"

func RequestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := strings.TrimSpace(c.GetHeader(requestIDHeader))
		if requestID == "" || len(requestID) > 64 {
			requestID = uuid.NewString()
		}
		c.Set("request_id", requestID)
		c.Header(requestIDHeader, requestID)
		c.Next()
	}
}
