package handler

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"mine-ventilation-network-simulator/backend/internal/service"
	"mine-ventilation-network-simulator/backend/pkg/api"
)

type SupportHandler struct {
	service *service.SupportService
}

func NewSupportHandler(service *service.SupportService) *SupportHandler {
	return &SupportHandler{service: service}
}

func (h *SupportHandler) Login(c *gin.Context) {
	var input struct {
		Email    string `json:"email" binding:"required,email,max=160"`
		Password string `json:"password" binding:"required,min=8,max=128"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		api.BindError(c, err)
		return
	}
	result, err := h.service.Login(c.Request.Context(), input.Email, input.Password)
	if err != nil {
		api.Fail(c, err)
		return
	}
	api.OK(c, result)
}

func (h *SupportHandler) Me(c *gin.Context) {
	actor, err := ActorFromContext(c)
	if err != nil {
		api.Fail(c, err)
		return
	}
	api.OK(c, actor)
}

func (h *SupportHandler) Audits(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "25"))
	from, err := parseOptionalTime(c.Query("from"))
	if err != nil {
		api.Fail(c, api.BadRequest("INVALID_FROM_TIME", "from 必须是 RFC3339 时间", nil))
		return
	}
	to, err := parseOptionalTime(c.Query("to"))
	if err != nil {
		api.Fail(c, api.BadRequest("INVALID_TO_TIME", "to 必须是 RFC3339 时间", nil))
		return
	}
	items, total, page, pageSize, err := h.service.ListAudits(c.Request.Context(), service.AuditQuery{
		Page: page, PageSize: pageSize, EntityType: strings.TrimSpace(c.Query("entity_type")),
		Action: strings.TrimSpace(c.Query("action")), Actor: strings.TrimSpace(c.Query("actor")),
		Status: strings.TrimSpace(c.Query("status")), From: from, To: to,
	})
	if err != nil {
		api.Fail(c, err)
		return
	}
	api.Page(c, items, page, pageSize, total)
}

func (h *SupportHandler) Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok", "service": "mine-ventilation-network-simulator", "time": time.Now().UTC().Format(time.RFC3339)})
}

func (h *SupportHandler) Ready(c *gin.Context) {
	if err := h.service.Ready(c.Request.Context()); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"status": "not_ready", "database": "unavailable"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ready", "database": "ok"})
}

func ActorFromContext(c *gin.Context) (service.Actor, error) {
	value, ok := c.Get("actor")
	if !ok {
		return service.Actor{}, api.Unauthorized("请求未包含有效登录身份")
	}
	actor, ok := value.(service.Actor)
	if !ok {
		return service.Actor{}, api.Unauthorized("登录身份格式无效")
	}
	actor.RequestID = api.RequestID(c)
	return actor, nil
}

func ParseID(c *gin.Context) (uint, bool) {
	value, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || value == 0 {
		api.Fail(c, api.BadRequest("INVALID_ID", "资源 ID 必须是正整数", nil))
		return 0, false
	}
	return uint(value), true
}

func parseOptionalTime(value string) (*time.Time, error) {
	if strings.TrimSpace(value) == "" {
		return nil, nil
	}
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return nil, err
	}
	return &parsed, nil
}
