package handler

import (
	"github.com/gin-gonic/gin"

	"mine-ventilation-network-simulator/backend/internal/dto"
	"mine-ventilation-network-simulator/backend/internal/service"
	"mine-ventilation-network-simulator/backend/pkg/api"
)

type AirwayEdgeHandler struct{ service *service.AirwayEdgeService }

func NewAirwayEdgeHandler(service *service.AirwayEdgeService) *AirwayEdgeHandler {
	return &AirwayEdgeHandler{service: service}
}

func (h *AirwayEdgeHandler) List(c *gin.Context) {
	var query dto.EdgeListQuery
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

func (h *AirwayEdgeHandler) Get(c *gin.Context) {
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

func (h *AirwayEdgeHandler) Create(c *gin.Context) {
	actor, err := ActorFromContext(c)
	if err != nil {
		api.Fail(c, err)
		return
	}
	var input dto.CreateAirwayEdgeRequest
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

func (h *AirwayEdgeHandler) Update(c *gin.Context) {
	id, ok := ParseID(c)
	if !ok {
		return
	}
	actor, err := ActorFromContext(c)
	if err != nil {
		api.Fail(c, err)
		return
	}
	var input dto.UpdateAirwayEdgeRequest
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
