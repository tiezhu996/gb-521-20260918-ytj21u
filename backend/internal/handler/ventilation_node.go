package handler

import (
	"github.com/gin-gonic/gin"

	"mine-ventilation-network-simulator/backend/internal/dto"
	"mine-ventilation-network-simulator/backend/internal/service"
	"mine-ventilation-network-simulator/backend/pkg/api"
)

type VentilationNodeHandler struct {
	service *service.VentilationNodeService
}

func NewVentilationNodeHandler(service *service.VentilationNodeService) *VentilationNodeHandler {
	return &VentilationNodeHandler{service: service}
}

func (h *VentilationNodeHandler) List(c *gin.Context) {
	var query dto.NodeListQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		api.BindError(c, err)
		return
	}
	items, total, page, pageSize, err := h.service.List(c.Request.Context(), query)
	if err != nil {
		api.Fail(c, err)
		return
	}
	api.Page(c, items, page, pageSize, total)
}

func (h *VentilationNodeHandler) Get(c *gin.Context) {
	id, ok := ParseID(c)
	if !ok {
		return
	}
	item, err := h.service.Get(c.Request.Context(), id)
	if err != nil {
		api.Fail(c, err)
		return
	}
	api.OK(c, item)
}

func (h *VentilationNodeHandler) Create(c *gin.Context) {
	actor, err := ActorFromContext(c)
	if err != nil {
		api.Fail(c, err)
		return
	}
	var input dto.CreateVentilationNodeRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		api.BindError(c, err)
		return
	}
	item, err := h.service.Create(c.Request.Context(), input, actor)
	if err != nil {
		api.Fail(c, err)
		return
	}
	api.Created(c, item)
}

func (h *VentilationNodeHandler) Update(c *gin.Context) {
	id, ok := ParseID(c)
	if !ok {
		return
	}
	actor, err := ActorFromContext(c)
	if err != nil {
		api.Fail(c, err)
		return
	}
	var input dto.UpdateVentilationNodeRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		api.BindError(c, err)
		return
	}
	item, err := h.service.Update(c.Request.Context(), id, input, actor)
	if err != nil {
		api.Fail(c, err)
		return
	}
	api.OK(c, item)
}

func (h *VentilationNodeHandler) ValidateNetwork(c *gin.Context) {
	result, err := h.service.ValidateNetwork(c.Request.Context())
	if err != nil {
		api.Fail(c, err)
		return
	}
	api.OK(c, result)
}
