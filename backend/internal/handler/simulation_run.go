package handler

import (
	"github.com/gin-gonic/gin"

	"mine-ventilation-network-simulator/backend/internal/dto"
	"mine-ventilation-network-simulator/backend/internal/service"
	"mine-ventilation-network-simulator/backend/pkg/api"
)

type SimulationRunHandler struct{ service *service.SimulationService }

func NewSimulationRunHandler(service *service.SimulationService) *SimulationRunHandler {
	return &SimulationRunHandler{service: service}
}

func (h *SimulationRunHandler) List(c *gin.Context) {
	var query dto.SimulationListQuery
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

func (h *SimulationRunHandler) Get(c *gin.Context) {
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

func (h *SimulationRunHandler) Start(c *gin.Context) {
	actor, err := ActorFromContext(c)
	if err != nil {
		api.Fail(c, err)
		return
	}
	var input dto.StartSimulationRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		api.BindError(c, err)
		return
	}
	item, err := h.service.Start(c.Request.Context(), input.ScenarioID, actor)
	if err != nil {
		api.Fail(c, err)
		return
	}
	api.Created(c, item)
}

func (h *SimulationRunHandler) ConfirmRisks(c *gin.Context) {
	id, ok := ParseID(c)
	if !ok {
		return
	}
	actor, err := ActorFromContext(c)
	if err != nil {
		api.Fail(c, err)
		return
	}
	var input dto.ConfirmRisksRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		api.BindError(c, err)
		return
	}
	item, err := h.service.ConfirmRisks(c.Request.Context(), id, input.Note, actor)
	if err != nil {
		api.Fail(c, err)
		return
	}
	api.OK(c, item)
}
