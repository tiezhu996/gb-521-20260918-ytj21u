package handler

import (
	"github.com/gin-gonic/gin"

	"mine-ventilation-network-simulator/backend/internal/dto"
	"mine-ventilation-network-simulator/backend/internal/service"
	"mine-ventilation-network-simulator/backend/pkg/api"
)

type FanScenarioHandler struct{ service *service.FanScenarioService }

func NewFanScenarioHandler(service *service.FanScenarioService) *FanScenarioHandler {
	return &FanScenarioHandler{service: service}
}

func (h *FanScenarioHandler) List(c *gin.Context) {
	var query dto.ScenarioListQuery
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

func (h *FanScenarioHandler) Get(c *gin.Context) {
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

func (h *FanScenarioHandler) Create(c *gin.Context) {
	actor, err := ActorFromContext(c)
	if err != nil {
		api.Fail(c, err)
		return
	}
	var input dto.CreateFanScenarioRequest
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

func (h *FanScenarioHandler) Transition(c *gin.Context) {
	id, ok := ParseID(c)
	if !ok {
		return
	}
	actor, err := ActorFromContext(c)
	if err != nil {
		api.Fail(c, err)
		return
	}
	var input dto.TransitionScenarioRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		api.BindError(c, err)
		return
	}
	item, err := h.service.Transition(c.Request.Context(), id, input, actor)
	if err != nil {
		api.Fail(c, err)
		return
	}
	api.OK(c, item)
}
