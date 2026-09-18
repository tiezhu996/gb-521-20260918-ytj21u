package middleware

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"mine-ventilation-network-simulator/backend/internal/service"
)

func TestRequestIDMiddlewarePreservesValidID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.Use(RequestIDMiddleware())
	engine.GET("/", func(c *gin.Context) { c.String(http.StatusOK, c.GetString("request_id")) })
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set("X-Request-ID", "trace-521")
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK || recorder.Body.String() != "trace-521" || recorder.Header().Get("X-Request-ID") != "trace-521" {
		t.Fatalf("request ID was not propagated: status=%d body=%q header=%q", recorder.Code, recorder.Body.String(), recorder.Header().Get("X-Request-ID"))
	}
}

func TestAccessLogMiddlewareDoesNotChangeResponse(t *testing.T) {
	engine := gin.New()
	engine.Use(AccessLogMiddleware(slog.New(slog.NewTextHandler(io.Discard, nil))))
	engine.GET("/", func(c *gin.Context) { c.Status(http.StatusNoContent) })
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/", nil))
	if recorder.Code != http.StatusNoContent {
		t.Fatalf("access logger changed status: %d", recorder.Code)
	}
}

func TestAuthMiddlewareRejectsMissingBearerToken(t *testing.T) {
	engine := gin.New()
	engine.Use(RequestIDMiddleware(), AuthMiddleware(nil))
	engine.GET("/", func(c *gin.Context) { c.Status(http.StatusNoContent) })
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/", nil))
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", recorder.Code)
	}
}

func TestRBACMiddlewareAllowsConfiguredRole(t *testing.T) {
	engine := gin.New()
	engine.Use(func(c *gin.Context) { c.Set("actor", service.Actor{ID: 3, Role: "admin"}); c.Next() })
	engine.Use(RBACMiddleware("admin"))
	engine.GET("/", func(c *gin.Context) { c.Status(http.StatusNoContent) })
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/", nil))
	if recorder.Code != http.StatusNoContent {
		t.Fatalf("expected admin to pass, got %d", recorder.Code)
	}
}

func TestRecoveryMiddlewareReturnsStructuredError(t *testing.T) {
	engine := gin.New()
	engine.Use(RequestIDMiddleware(), RecoveryMiddleware(slog.New(slog.NewTextHandler(io.Discard, nil))))
	engine.GET("/", func(*gin.Context) { panic("test panic") })
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/", nil))
	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("expected recovered 500, got %d", recorder.Code)
	}
	if recorder.Header().Get("X-Request-ID") == "" {
		t.Fatal("recovered response omitted request ID")
	}
}

func TestRateLimitMiddlewareBlocksAfterLimit(t *testing.T) {
	engine := gin.New()
	engine.Use(RequestIDMiddleware(), NewRateLimiter("test", 1, time.Minute).Middleware())
	engine.GET("/", func(c *gin.Context) { c.Status(http.StatusNoContent) })
	first := httptest.NewRecorder()
	engine.ServeHTTP(first, httptest.NewRequest(http.MethodGet, "/", nil))
	second := httptest.NewRecorder()
	engine.ServeHTTP(second, httptest.NewRequest(http.MethodGet, "/", nil))
	if first.Code != http.StatusNoContent || second.Code != http.StatusTooManyRequests {
		t.Fatalf("unexpected statuses: first=%d second=%d", first.Code, second.Code)
	}
}
