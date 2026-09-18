package router

import (
	"github.com/gin-gonic/gin"

	"mine-ventilation-network-simulator/backend/internal/constants"
	"mine-ventilation-network-simulator/backend/internal/handler"
	"mine-ventilation-network-simulator/backend/internal/middleware"
)

func registerFanScenarioRoutes(group *gin.RouterGroup, h *handler.FanScenarioHandler) {
	scenarios := group.Group("/scenarios")
	scenarios.GET("", h.List)
	scenarios.GET("/:id", h.Get)
	scenarios.POST("", middleware.RBACMiddleware(string(constants.RoleEngineer), string(constants.RoleAdmin)), h.Create)
	scenarios.POST("/:id/transition", middleware.RBACMiddleware(string(constants.RoleEngineer), string(constants.RoleReviewer), string(constants.RoleAdmin)), h.Transition)
}
