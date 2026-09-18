package service

import (
	"context"
	"strings"

	"mine-ventilation-network-simulator/backend/internal/constants"
	"mine-ventilation-network-simulator/backend/internal/dto"
	"mine-ventilation-network-simulator/backend/internal/model"
	"mine-ventilation-network-simulator/backend/internal/repository"
	"mine-ventilation-network-simulator/backend/pkg/api"
)

type AirwayEdgeService struct {
	edges *repository.AirwayEdgeRepository
	nodes *repository.VentilationNodeRepository
}

func NewAirwayEdgeService(edges *repository.AirwayEdgeRepository, nodes *repository.VentilationNodeRepository) *AirwayEdgeService {
	return &AirwayEdgeService{edges: edges, nodes: nodes}
}

func (s *AirwayEdgeService) List(ctx context.Context, query dto.EdgeListQuery) ([]model.AirwayEdge, int64, int, int, error) {
	page, pageSize := normalizePage(query.Page, query.PageSize)
	if query.Enabled != "" && query.Enabled != "true" && query.Enabled != "false" {
		return nil, 0, page, pageSize, api.BadRequest("INVALID_ENABLED_FILTER", "enabled 筛选只能是 true 或 false", nil)
	}
	items, total, err := s.edges.List(ctx, page, pageSize, query.Enabled, strings.TrimSpace(query.Search))
	if err != nil {
		return nil, 0, page, pageSize, mapRepositoryError(err, "巷道边")
	}
	return items, total, page, pageSize, nil
}

func (s *AirwayEdgeService) Get(ctx context.Context, id uint) (*model.AirwayEdge, error) {
	edge, err := s.edges.Find(ctx, id)
	return edge, mapRepositoryError(err, "巷道边")
}

func (s *AirwayEdgeService) Create(ctx context.Context, input dto.CreateAirwayEdgeRequest, actor Actor) (*model.AirwayEdge, error) {
	if err := s.validateEndpoints(ctx, input.FromNodeID, input.ToNodeID, input.DoorState); err != nil {
		return nil, err
	}
	edge := &model.AirwayEdge{
		Code: strings.ToUpper(strings.TrimSpace(input.Code)), FromNodeID: input.FromNodeID, ToNodeID: input.ToNodeID,
		ResistanceNS2M8: input.ResistanceNS2M8, AreaM2: input.AreaM2, MaxVelocityMS: input.MaxVelocityMS,
		DoorState: input.DoorState, Enabled: *input.Enabled, CriticalPath: input.CriticalPath, Version: 1,
	}
	if err := s.edges.Create(ctx, edge, actor.Audit("airway_edge.created", "airway_edge")); err != nil {
		return nil, mapRepositoryError(err, "巷道边")
	}
	return s.Get(ctx, edge.ID)
}

func (s *AirwayEdgeService) Update(ctx context.Context, id uint, input dto.UpdateAirwayEdgeRequest, actor Actor) (*model.AirwayEdge, error) {
	existing, err := s.edges.Find(ctx, id)
	if err != nil {
		return nil, mapRepositoryError(err, "巷道边")
	}
	if !constants.ValidDoorState(input.DoorState) {
		return nil, api.BadRequest("INVALID_DOOR_STATE", "风门状态必须是 open、closed 或 regulating", nil)
	}
	existing.ResistanceNS2M8 = input.ResistanceNS2M8
	existing.AreaM2 = input.AreaM2
	existing.MaxVelocityMS = input.MaxVelocityMS
	existing.DoorState = input.DoorState
	existing.Enabled = *input.Enabled
	existing.CriticalPath = input.CriticalPath
	if err := s.edges.Update(ctx, existing, input.Version, actor.Audit("airway_edge.updated", "airway_edge")); err != nil {
		return nil, mapRepositoryError(err, "巷道边")
	}
	return existing, nil
}

func (s *AirwayEdgeService) validateEndpoints(ctx context.Context, fromID, toID uint, doorState string) error {
	if fromID == toID {
		return api.BadRequest("SELF_LOOP", "巷道边的起点和终点必须不同", nil)
	}
	if !constants.ValidDoorState(doorState) {
		return api.BadRequest("INVALID_DOOR_STATE", "风门状态必须是 open、closed 或 regulating", nil)
	}
	from, err := s.nodes.Find(ctx, fromID)
	if err != nil {
		return mapRepositoryError(err, "起点节点")
	}
	to, err := s.nodes.Find(ctx, toID)
	if err != nil {
		return mapRepositoryError(err, "终点节点")
	}
	if from.Status != string(constants.NodeStatusActive) || to.Status != string(constants.NodeStatusActive) {
		return api.Conflict("NODE_NOT_ACTIVE", "只有启用节点可以建立巷道连接")
	}
	return nil
}
