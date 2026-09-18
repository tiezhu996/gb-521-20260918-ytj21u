package router

import (
	"github.com/gin-gonic/gin"

	"mine-ventilation-network-simulator/backend/internal/constants"
	"mine-ventilation-network-simulator/backend/internal/handler"
	"mine-ventilation-network-simulator/backend/internal/middleware"
)

func registerAirwayEdgeRoutes(group *gin.RouterGroup, h *handler.AirwayEdgeHandler) {
	edges := group.Group("/edges")
	edges.GET("", h.List)
	edges.GET("/:id", h.Get)
	edges.POST("", middleware.RBACMiddleware(string(constants.RoleEngineer), string(constants.RoleAdmin)), h.Create)
	edges.PUT("/:id", middleware.RBACMiddleware(string(constants.RoleEngineer), string(constants.RoleAdmin)), h.Update)
}
