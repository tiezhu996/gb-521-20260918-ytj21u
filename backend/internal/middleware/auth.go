package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"

	"mine-ventilation-network-simulator/backend/internal/service"
	"mine-ventilation-network-simulator/backend/pkg/api"
)

func AuthMiddleware(auth *service.SupportService) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := strings.TrimSpace(c.GetHeader("Authorization"))
		parts := strings.Fields(header)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			api.Fail(c, api.Unauthorized("请使用 Authorization: Bearer <token> 提供登录凭据"))
			return
		}
		actor, err := auth.AuthenticateToken(c.Request.Context(), parts[1])
		if err != nil {
			api.Fail(c, err)
			return
		}
		actor.RequestID = c.GetString("request_id")
		c.Set("actor", actor)
		c.Next()
	}
}
