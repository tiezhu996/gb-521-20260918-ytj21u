package service

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"mine-ventilation-network-simulator/backend/internal/constants"
	"mine-ventilation-network-simulator/backend/internal/dto"
	"mine-ventilation-network-simulator/backend/internal/model"
	"mine-ventilation-network-simulator/backend/internal/repository"
	"mine-ventilation-network-simulator/backend/pkg/api"
)

type NetworkValidation struct {
	Valid        bool               `json:"valid"`
	NodeCount    int                `json:"node_count"`
	EdgeCount    int                `json:"edge_count"`
	IntakeCount  int                `json:"intake_count"`
	ExhaustCount int                `json:"exhaust_count"`
	Issues       []dto.NetworkIssue `json:"issues"`
}

type VentilationNodeService struct {
	nodes *repository.VentilationNodeRepository
	edges *repository.AirwayEdgeRepository
}

func NewVentilationNodeService(nodes *repository.VentilationNodeRepository, edges *repository.AirwayEdgeRepository) *VentilationNodeService {
	return &VentilationNodeService{nodes: nodes, edges: edges}
}

func (s *VentilationNodeService) List(ctx context.Context, query dto.NodeListQuery) ([]model.VentilationNode, int64, int, int, error) {
	page, pageSize := normalizePage(query.Page, query.PageSize)
	if query.Type != "" && !constants.ValidNodeType(query.Type) {
		return nil, 0, page, pageSize, api.BadRequest("INVALID_NODE_TYPE", "节点类型不在允许范围内", nil)
	}
	if query.Status != "" && !constants.ValidNodeStatus(query.Status) {
		return nil, 0, page, pageSize, api.BadRequest("INVALID_NODE_STATUS", "节点状态不在允许范围内", nil)
	}
	items, total, err := s.nodes.List(ctx, page, pageSize, query.Type, query.Status, strings.TrimSpace(query.Search))
	if err != nil {
		return nil, 0, page, pageSize, mapRepositoryError(err, "通风节点")
	}
	return items, total, page, pageSize, nil
}

func (s *VentilationNodeService) Get(ctx context.Context, id uint) (*model.VentilationNode, error) {
	node, err := s.nodes.Find(ctx, id)
	return node, mapRepositoryError(err, "通风节点")
}

func (s *VentilationNodeService) Create(ctx context.Context, input dto.CreateVentilationNodeRequest, actor Actor) (*model.VentilationNode, error) {
	if err := validateNode(input.NodeType, input.Status, input.RequiredAirflowM3S); err != nil {
		return nil, err
	}
	node := &model.VentilationNode{
		Code: strings.ToUpper(strings.TrimSpace(input.Code)), NodeType: input.NodeType,
		ElevationM: input.ElevationM, RequiredAirflowM3S: input.RequiredAirflowM3S,
		PressurePa: input.PressurePa, Status: input.Status,
	}
	audit := actor.Audit("ventilation_node.created", "ventilation_node")
	if err := s.nodes.Create(ctx, node, audit); err != nil {
		return nil, mapRepositoryError(err, "通风节点")
	}
	return node, nil
}

func (s *VentilationNodeService) Update(ctx context.Context, id uint, input dto.UpdateVentilationNodeRequest, actor Actor) (*model.VentilationNode, error) {
	if err := validateNode(input.NodeType, input.Status, input.RequiredAirflowM3S); err != nil {
		return nil, err
	}
	existing, err := s.nodes.Find(ctx, id)
	if err != nil {
		return nil, mapRepositoryError(err, "通风节点")
	}
	existing.NodeType = input.NodeType
	existing.ElevationM = input.ElevationM
	existing.RequiredAirflowM3S = input.RequiredAirflowM3S
	existing.PressurePa = input.PressurePa
	existing.Status = input.Status
	if err := s.nodes.Update(ctx, existing, actor.Audit("ventilation_node.updated", "ventilation_node")); err != nil {
		return nil, mapRepositoryError(err, "通风节点")
	}
	return existing, nil
}

func (s *VentilationNodeService) ValidateNetwork(ctx context.Context) (*NetworkValidation, error) {
	nodes, err := s.nodes.AllActive(ctx)
	if err != nil {
		return nil, mapRepositoryError(err, "通风网络")
	}
	edges, err := s.edges.AllEnabled(ctx)
	if err != nil {
		return nil, mapRepositoryError(err, "通风网络")
	}
	result := validateDirectedNetwork(nodes, edges)
	return &result, nil
}

func validateNode(nodeType, status string, required float64) error {
	if !constants.ValidNodeType(nodeType) {
		return api.BadRequest("INVALID_NODE_TYPE", "节点类型必须是 intake、exhaust、workface 或 junction", nil)
	}
	if !constants.ValidNodeStatus(status) {
		return api.BadRequest("INVALID_NODE_STATUS", "节点状态必须是 active、inactive 或 blocked", nil)
	}
	if nodeType != string(constants.NodeTypeWorkface) && required > 0 {
		return api.BadRequest("AIRFLOW_DEMAND_NOT_ALLOWED", "只有工作面节点可以设置最低需风量", nil)
	}
	return nil
}

func validateDirectedNetwork(nodes []model.VentilationNode, edges []model.AirwayEdge) NetworkValidation {
	result := NetworkValidation{NodeCount: len(nodes), EdgeCount: len(edges), Issues: []dto.NetworkIssue{}}
	nodeByID := make(map[uint]model.VentilationNode, len(nodes))
	adjacency := make(map[uint][]uint, len(nodes))
	degree := make(map[uint]int, len(nodes))
	intakes := make([]uint, 0)
	for _, node := range nodes {
		nodeByID[node.ID] = node
		switch constants.NodeType(node.NodeType) {
		case constants.NodeTypeIntake:
			result.IntakeCount++
			intakes = append(intakes, node.ID)
		case constants.NodeTypeExhaust:
			result.ExhaustCount++
		}
	}
	for _, edge := range edges {
		if edge.FromNodeID == edge.ToNodeID {
			result.Issues = append(result.Issues, dto.NetworkIssue{Code: "SELF_LOOP", EntityType: "airway_edge", EntityID: edge.ID, Message: "巷道边不能连接同一节点", Severity: "critical"})
			continue
		}
		if _, ok := nodeByID[edge.FromNodeID]; !ok {
			result.Issues = append(result.Issues, dto.NetworkIssue{Code: "MISSING_FROM_NODE", EntityType: "airway_edge", EntityID: edge.ID, Message: "巷道起点不存在或未启用", Severity: "critical"})
			continue
		}
		if _, ok := nodeByID[edge.ToNodeID]; !ok {
			result.Issues = append(result.Issues, dto.NetworkIssue{Code: "MISSING_TO_NODE", EntityType: "airway_edge", EntityID: edge.ID, Message: "巷道终点不存在或未启用", Severity: "critical"})
			continue
		}
		adjacency[edge.FromNodeID] = append(adjacency[edge.FromNodeID], edge.ToNodeID)
		degree[edge.FromNodeID]++
		degree[edge.ToNodeID]++
	}
	if result.IntakeCount == 0 {
		result.Issues = append(result.Issues, dto.NetworkIssue{Code: "MISSING_INTAKE", EntityType: "network", Message: "网络至少需要一个启用的进风边界", Severity: "critical"})
	}
	if result.ExhaustCount == 0 {
		result.Issues = append(result.Issues, dto.NetworkIssue{Code: "MISSING_EXHAUST", EntityType: "network", Message: "网络至少需要一个启用的回风边界", Severity: "critical"})
	}
	for _, node := range nodes {
		if degree[node.ID] == 0 {
			result.Issues = append(result.Issues, dto.NetworkIssue{Code: "ISOLATED_NODE", EntityType: "ventilation_node", EntityID: node.ID, Message: fmt.Sprintf("节点 %s 未连接任何启用巷道", node.Code), Severity: "critical"})
		}
	}
	reachable := make(map[uint]bool, len(nodes))
	queue := append([]uint(nil), intakes...)
	for len(queue) > 0 {
		id := queue[0]
		queue = queue[1:]
		if reachable[id] {
			continue
		}
		reachable[id] = true
		neighbors := append([]uint(nil), adjacency[id]...)
		sort.Slice(neighbors, func(i, j int) bool { return neighbors[i] < neighbors[j] })
		queue = append(queue, neighbors...)
	}
	for _, node := range nodes {
		if node.NodeType == string(constants.NodeTypeWorkface) && !reachable[node.ID] {
			result.Issues = append(result.Issues, dto.NetworkIssue{Code: "UNREACHABLE_WORKFACE", EntityType: "ventilation_node", EntityID: node.ID, Message: fmt.Sprintf("工作面 %s 无法从进风边界到达", node.Code), Severity: "critical"})
		}
	}
	result.Valid = len(result.Issues) == 0
	return result
}
