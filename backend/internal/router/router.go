package router

import (
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"mine-ventilation-network-simulator/backend/internal/config"
	"mine-ventilation-network-simulator/backend/internal/constants"
	"mine-ventilation-network-simulator/backend/internal/handler"
	"mine-ventilation-network-simulator/backend/internal/middleware"
	"mine-ventilation-network-simulator/backend/internal/repository"
	"mine-ventilation-network-simulator/backend/internal/service"
	"mine-ventilation-network-simulator/backend/pkg/api"
)

func New(cfg config.Config, db *gorm.DB, logger *slog.Logger) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	engine := gin.New()
	engine.Use(
		middleware.RequestIDMiddleware(),
		middleware.AccessLogMiddleware(logger),
		middleware.RecoveryMiddleware(logger),
		corsMiddleware(cfg.CORSOrigins),
	)

	supportRepo := repository.NewSupportRepository(db)
	nodeRepo := repository.NewVentilationNodeRepository(db)
	edgeRepo := repository.NewAirwayEdgeRepository(db)
	scenarioRepo := repository.NewFanScenarioRepository(db)
	runRepo := repository.NewSimulationRunRepository(db)

	supportService := service.NewSupportService(supportRepo, cfg.JWTSecret, cfg.JWTTTL)
	nodeService := service.NewVentilationNodeService(nodeRepo, edgeRepo)
	edgeService := service.NewAirwayEdgeService(edgeRepo, nodeRepo)
	scenarioService := service.NewFanScenarioService(scenarioRepo)
	runService := service.NewSimulationService(runRepo, scenarioRepo, nodeRepo, edgeRepo)

	supportHandler := handler.NewSupportHandler(supportService)
	nodeHandler := handler.NewVentilationNodeHandler(nodeService)
	edgeHandler := handler.NewAirwayEdgeHandler(edgeService)
	scenarioHandler := handler.NewFanScenarioHandler(scenarioService)
	runHandler := handler.NewSimulationRunHandler(runService)

	engine.GET("/healthz", supportHandler.Health)
	engine.GET("/readyz", supportHandler.Ready)
	engine.GET("/api/healthz", supportHandler.Health)
	apiV1 := engine.Group("/api/v1")
	loginLimiter := middleware.NewRateLimiter("login", 12, time.Minute)
	apiV1.POST("/auth/login", loginLimiter.Middleware(), supportHandler.Login)

	protected := apiV1.Group("")
	protected.Use(middleware.AuthMiddleware(supportService))
	protected.GET("/auth/me", supportHandler.Me)
	protected.GET("/audits", middleware.RBACMiddleware(string(constants.RoleReviewer), string(constants.RoleAdmin)), supportHandler.Audits)
	registerVentilationNodeRoutes(protected, nodeHandler)
	registerAirwayEdgeRoutes(protected, edgeHandler)
	registerFanScenarioRoutes(protected, scenarioHandler)
	runLimiter := middleware.NewRateLimiter("simulation", 10, time.Minute)
	registerSimulationRoutes(protected, runHandler, runLimiter)

	engine.NoRoute(func(c *gin.Context) {
		api.Fail(c, api.NewError(http.StatusNotFound, "ROUTE_NOT_FOUND", "请求的接口不存在", nil))
	})
	engine.NoMethod(func(c *gin.Context) {
		api.Fail(c, api.NewError(http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "该接口不支持当前请求方法", nil))
	})
	return engine
}

func corsMiddleware(origins []string) gin.HandlerFunc {
	allowed := make(map[string]struct{}, len(origins))
	for _, origin := range origins {
		allowed[strings.TrimSpace(origin)] = struct{}{}
	}
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if _, ok := allowed[origin]; ok {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Vary", "Origin")
			c.Header("Access-Control-Allow-Credentials", "true")
			c.Header("Access-Control-Allow-Headers", "Authorization, Content-Type, X-Request-ID")
			c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, OPTIONS")
		}
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}
