package router

import (
	"github.com/gin-gonic/gin"

	"mine-ventilation-network-simulator/backend/internal/constants"
	"mine-ventilation-network-simulator/backend/internal/handler"
	"mine-ventilation-network-simulator/backend/internal/middleware"
)

func registerSimulationRoutes(group *gin.RouterGroup, h *handler.SimulationRunHandler, limiter *middleware.RateLimiter) {
	runs := group.Group("/simulations")
	runs.GET("", h.List)
	runs.GET("/:id", h.Get)
	runs.POST("", middleware.RBACMiddleware(string(constants.RoleEngineer), string(constants.RoleAdmin)), limiter.Middleware(), h.Start)
	runs.POST("/:id/confirm-risks", middleware.RBACMiddleware(string(constants.RoleReviewer), string(constants.RoleAdmin)), h.ConfirmRisks)
}
