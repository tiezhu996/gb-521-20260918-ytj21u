package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"gorm.io/datatypes"

	"mine-ventilation-network-simulator/backend/internal/constants"
	"mine-ventilation-network-simulator/backend/internal/dto"
	"mine-ventilation-network-simulator/backend/internal/model"
	"mine-ventilation-network-simulator/backend/internal/repository"
	"mine-ventilation-network-simulator/backend/pkg/api"
)

type FanScenarioService struct {
	scenarios *repository.FanScenarioRepository
}

func NewFanScenarioService(scenarios *repository.FanScenarioRepository) *FanScenarioService {
	return &FanScenarioService{scenarios: scenarios}
}

func (s *FanScenarioService) List(ctx context.Context, query dto.ScenarioListQuery) ([]model.FanScenario, int64, int, int, error) {
	page, pageSize := normalizePage(query.Page, query.PageSize)
	if query.Status != "" && !constants.ValidScenarioStatus(query.Status) {
		return nil, 0, page, pageSize, api.BadRequest("INVALID_SCENARIO_STATUS", "方案状态筛选值无效", nil)
	}
	items, total, err := s.scenarios.List(ctx, page, pageSize, query.Status, strings.TrimSpace(query.Search))
	if err != nil {
		return nil, 0, page, pageSize, mapRepositoryError(err, "风机方案")
	}
	return items, total, page, pageSize, nil
}

func (s *FanScenarioService) Get(ctx context.Context, id uint) (*model.FanScenario, error) {
	item, err := s.scenarios.Find(ctx, id)
	return item, mapRepositoryError(err, "风机方案")
}

func (s *FanScenarioService) Create(ctx context.Context, input dto.CreateFanScenarioRequest, actor Actor) (*model.FanScenario, error) {
	if err := validateFanCurve(input.FanCurve); err != nil {
		return nil, err
	}
	curve, err := json.Marshal(input.FanCurve)
	if err != nil {
		return nil, api.BadRequest("INVALID_FAN_CURVE", "风机曲线无法编码", err.Error())
	}
	scenario := &model.FanScenario{
		Name: strings.TrimSpace(input.Name), Description: strings.TrimSpace(input.Description),
		FanCurveJSON: datatypes.JSON(curve), OperatingMode: input.OperatingMode,
		ScenarioStatus: string(constants.ScenarioStatusDraft), SolverTolerance: input.SolverTolerance,
		MaxIterations: input.MaxIterations, Version: 1, CreatedBy: actor.ID,
	}
	if err := s.scenarios.Create(ctx, scenario, actor.Audit("fan_scenario.created", "fan_scenario")); err != nil {
		return nil, mapRepositoryError(err, "风机方案")
	}
	return scenario, nil
}

func (s *FanScenarioService) Transition(ctx context.Context, id uint, input dto.TransitionScenarioRequest, actor Actor) (*model.FanScenario, error) {
	scenario, err := s.scenarios.Find(ctx, id)
	if err != nil {
		return nil, mapRepositoryError(err, "风机方案")
	}
	from := constants.ScenarioStatus(scenario.ScenarioStatus)
	to := constants.ScenarioStatus(input.TargetStatus)
	if !constants.ValidScenarioStatus(input.TargetStatus) || !constants.CanTransitionScenario(from, to) {
		return nil, api.Conflict("ILLEGAL_SCENARIO_TRANSITION", fmt.Sprintf("方案不能从 %s 迁移到 %s", from, to))
	}
	if err := authorizeTransition(from, to, actor.Role, strings.TrimSpace(input.Reason)); err != nil {
		return nil, err
	}
	audit := actor.Audit("fan_scenario."+string(to), "fan_scenario")
	audit.Metadata = fmt.Sprintf(`{"reason":%q}`, strings.TrimSpace(input.Reason))
	updated, err := s.scenarios.Transition(ctx, id, actor.ID, input.Version, string(from), string(to), strings.TrimSpace(input.Reason), audit)
	if err != nil {
		return nil, mapRepositoryError(err, "风机方案")
	}
	return updated, nil
}

func validateFanCurve(points []dto.FanCurvePoint) error {
	if len(points) < 2 {
		return api.BadRequest("FAN_CURVE_TOO_SHORT", "风机曲线至少需要两个点", nil)
	}
	for i := range points {
		if i > 0 && points[i].FlowM3S <= points[i-1].FlowM3S {
			return api.BadRequest("FAN_CURVE_FLOW_ORDER", "风机曲线流量必须严格递增", map[string]int{"point": i + 1})
		}
		if i > 0 && points[i].PressurePa > points[i-1].PressurePa {
			return api.BadRequest("FAN_CURVE_PRESSURE_ORDER", "风机曲线压力必须随流量保持不升", map[string]int{"point": i + 1})
		}
	}
	return nil
}

func authorizeTransition(from, to constants.ScenarioStatus, role, reason string) error {
	switch {
	case from == constants.ScenarioStatusDraft && to == constants.ScenarioStatusPendingReview:
		if role != string(constants.RoleEngineer) && role != string(constants.RoleAdmin) {
			return api.Forbidden("只有工程师或管理员可以提交复核")
		}
	case from == constants.ScenarioStatusPendingReview && to == constants.ScenarioStatusApproved:
		if role != string(constants.RoleReviewer) && role != string(constants.RoleAdmin) {
			return api.Forbidden("只有复核员或管理员可以批准方案")
		}
	case from == constants.ScenarioStatusPendingReview && to == constants.ScenarioStatusDraft:
		if role != string(constants.RoleReviewer) && role != string(constants.RoleAdmin) {
			return api.Forbidden("只有复核员或管理员可以驳回方案")
		}
		if len(reason) < 4 {
			return api.BadRequest("REJECT_REASON_REQUIRED", "驳回方案时必须填写至少 4 个字符的原因", nil)
		}
	case from == constants.ScenarioStatusApproved && to == constants.ScenarioStatusArchived:
		if role != string(constants.RoleAdmin) {
			return api.Forbidden("只有管理员可以归档已批准方案")
		}
	}
	return nil
}
