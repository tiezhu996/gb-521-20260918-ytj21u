package router

import (
	"github.com/gin-gonic/gin"

	"mine-ventilation-network-simulator/backend/internal/constants"
	"mine-ventilation-network-simulator/backend/internal/handler"
	"mine-ventilation-network-simulator/backend/internal/middleware"
)

func registerVentilationNodeRoutes(group *gin.RouterGroup, h *handler.VentilationNodeHandler) {
	group.GET("/network/validate", h.ValidateNetwork)
	nodes := group.Group("/nodes")
	nodes.GET("", h.List)
	nodes.GET("/:id", h.Get)
	nodes.POST("", middleware.RBACMiddleware(string(constants.RoleEngineer), string(constants.RoleAdmin)), h.Create)
	nodes.PUT("/:id", middleware.RBACMiddleware(string(constants.RoleEngineer), string(constants.RoleAdmin)), h.Update)
}
