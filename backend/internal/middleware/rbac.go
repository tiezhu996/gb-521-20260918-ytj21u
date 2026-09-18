package middleware

import (
	"github.com/gin-gonic/gin"

	"mine-ventilation-network-simulator/backend/internal/service"
	"mine-ventilation-network-simulator/backend/pkg/api"
)

func RBACMiddleware(roles ...string) gin.HandlerFunc {
	allowed := make(map[string]struct{}, len(roles))
	for _, role := range roles {
		allowed[role] = struct{}{}
	}
	return func(c *gin.Context) {
		value, ok := c.Get("actor")
		actor, valid := value.(service.Actor)
		if !ok || !valid {
			api.Fail(c, api.Unauthorized("请求未包含有效登录身份"))
			return
		}
		if _, ok := allowed[actor.Role]; !ok {
			api.Fail(c, api.Forbidden("当前角色无权执行此操作"))
			return
		}
		c.Next()
	}
}
