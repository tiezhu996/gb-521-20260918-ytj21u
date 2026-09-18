package service

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"mine-ventilation-network-simulator/backend/internal/constants"
	"mine-ventilation-network-simulator/backend/internal/dto"
	"mine-ventilation-network-simulator/backend/internal/model"
	"mine-ventilation-network-simulator/backend/internal/repository"
	"mine-ventilation-network-simulator/backend/pkg/api"
)

type snapshotHarness struct {
	db            *gorm.DB
	scenarioSvc   *FanScenarioService
	simulationSvc *SimulationService
	nodeSvc       *VentilationNodeService
	actorEng      Actor
	actorReviewer Actor
}

func newSnapshotHarness(t *testing.T) snapshotHarness {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared&_pragma=busy_timeout(5000)"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	sqlDB, _ := db.DB()
	sqlDB.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = sqlDB.Close() })
	if err := db.AutoMigrate(&model.User{}, &model.NetworkRevision{}, &model.VentilationNode{}, &model.AirwayEdge{}, &model.FanScenario{}, &model.SimulationRun{}, &model.AuditEvent{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	db.Exec("INSERT INTO network_revisions (id, revision, updated_at) VALUES (1, 0, CURRENT_TIMESTAMP)")

	nodeRepo := repository.NewVentilationNodeRepository(db)
	edgeRepo := repository.NewAirwayEdgeRepository(db)
	scenarioRepo := repository.NewFanScenarioRepository(db)
	runRepo := repository.NewSimulationRunRepository(db)
	snapshotRepo := repository.NewNetworkSnapshotRepository(db)

	return snapshotHarness{
		db:            db,
		nodeSvc:       NewVentilationNodeService(nodeRepo, edgeRepo),
		scenarioSvc:   NewFanScenarioService(scenarioRepo, snapshotRepo),
		simulationSvc: NewSimulationService(runRepo, scenarioRepo, snapshotRepo),
		actorEng:      Actor{ID: 1, Email: "engineer@mine.local", Name: "工程师", Role: string(constants.RoleEngineer), RequestID: "req-eng"},
		actorReviewer: Actor{ID: 2, Email: "reviewer@mine.local", Name: "复核员", Role: string(constants.RoleReviewer), RequestID: "req-rev"},
	}
}

func (h snapshotHarness) seedValidNetwork(t *testing.T) {
	t.Helper()
	ctx := context.Background()
	nodes := []dto.CreateVentilationNodeRequest{
		{Code: "in1", NodeType: string(constants.NodeTypeIntake), Status: string(constants.NodeStatusActive), PressurePa: 1000},
		{Code: "wf1", NodeType: string(constants.NodeTypeWorkface), Status: string(constants.NodeStatusActive), RequiredAirflowM3S: 8, PressurePa: 500},
		{Code: "out1", NodeType: string(constants.NodeTypeExhaust), Status: string(constants.NodeStatusActive)},
	}
	ids := make([]uint, 0, 3)
	for _, n := range nodes {
		created, err := h.nodeSvc.Create(ctx, n, h.actorEng)
		if err != nil {
			t.Fatalf("create node %s: %v", n.Code, err)
		}
		ids = append(ids, created.ID)
	}
	edgeService := NewAirwayEdgeService(repository.NewAirwayEdgeRepository(h.db), repository.NewVentilationNodeRepository(h.db))
	for _, e := range []dto.CreateAirwayEdgeRequest{
		{Code: "e1", FromNodeID: ids[0], ToNodeID: ids[1], ResistanceNS2M8: 2, AreaM2: 5, MaxVelocityMS: 5, DoorState: string(constants.DoorStateOpen), Enabled: boolPtr(true), CriticalPath: true},
		{Code: "e2", FromNodeID: ids[1], ToNodeID: ids[2], ResistanceNS2M8: 2, AreaM2: 5, MaxVelocityMS: 5, DoorState: string(constants.DoorStateOpen), Enabled: boolPtr(true), CriticalPath: true},
	} {
		if _, err := edgeService.Create(ctx, e, h.actorEng); err != nil {
			t.Fatalf("create edge %s: %v", e.Code, err)
		}
	}
}

func boolPtr(v bool) *bool { return &v }

func (h snapshotHarness) createSubmitApproveScenario(t *testing.T) *model.FanScenario {
	t.Helper()
	ctx := context.Background()
	curve := []dto.FanCurvePoint{{FlowM3S: 0, PressurePa: 900}, {FlowM3S: 30, PressurePa: 600}, {FlowM3S: 60, PressurePa: 300}}
	sc, err := h.scenarioSvc.Create(ctx, dto.CreateFanScenarioRequest{
		Name: "闭环方案", Description: "服务层闭环测试方案", FanCurve: curve, OperatingMode: "normal",
		SolverTolerance: 0.02, MaxIterations: 100,
	}, h.actorEng)
	if err != nil {
		t.Fatalf("create scenario: %v", err)
	}
	if _, err := h.scenarioSvc.Transition(ctx, sc.ID, dto.TransitionScenarioRequest{TargetStatus: "pending_review", Version: sc.Version}, h.actorEng); err != nil {
		t.Fatalf("submit: %v", err)
	}
	approved, err := h.scenarioSvc.Transition(ctx, sc.ID, dto.TransitionScenarioRequest{TargetStatus: "approved", Version: sc.Version + 1}, h.actorReviewer)
	if err != nil {
		t.Fatalf("approve: %v", err)
	}
	return approved
}

func TestServiceSimulationRejectsStaleApprovalAndRecoversAfterReapproval(t *testing.T) {
	h := newSnapshotHarness(t)
	ctx := context.Background()
	h.seedValidNetwork(t)
	scenario := h.createSubmitApproveScenario(t)

	// 批准后无网络变化：推演可启动。
	run1, err := h.simulationSvc.Start(ctx, scenario.ID, h.actorEng)
	if err != nil {
		t.Fatalf("simulation on valid snapshot: %v", err)
	}
	if run1.InputSnapshotJSON == nil || string(run1.InputSnapshotJSON) == "null" {
		t.Fatal("simulation must persist its input snapshot")
	}

	// 修改网络参数（节点压力）→ 系统失效旧批准。
	var node model.VentilationNode
	if err := h.db.Where("code = ?", "WF1").First(&node).Error; err != nil {
		t.Fatalf("load node: %v", err)
	}
	_, err = h.nodeSvc.Update(ctx, node.ID, dto.UpdateVentilationNodeRequest{
		NodeType: node.NodeType, ElevationM: node.ElevationM, RequiredAirflowM3S: node.RequiredAirflowM3S,
		PressurePa: node.PressurePa + 120, Status: node.Status,
	}, h.actorEng)
	if err != nil {
		t.Fatalf("update node: %v", err)
	}

	// 推演入口必须拒绝并返回 409 SCENARIO_SNAPSHOT_STALE。
	_, err = h.simulationSvc.Start(ctx, scenario.ID, h.actorEng)
	var appErr *api.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected AppError, got %v", err)
	}
	if appErr.Code != "SCENARIO_SNAPSHOT_STALE" || appErr.Status != 409 {
		t.Fatalf("expected 409 SCENARIO_SNAPSHOT_STALE, got %d %s: %s", appErr.Status, appErr.Code, appErr.Message)
	}

	// 方案当前为 pending_review 且带失效原因。
	stale, _ := h.scenarioSvc.Get(ctx, scenario.ID)
	if stale.ScenarioStatus != "pending_review" || stale.InvalidationReason == "" {
		t.Fatalf("approval not invalidated: %+v", stale)
	}

	// 复核员重新批准（不要求工程师重新提交），绑定新快照后恢复推演。
	reapproved, err := h.scenarioSvc.Transition(ctx, scenario.ID, dto.TransitionScenarioRequest{TargetStatus: "approved", Version: stale.Version}, h.actorReviewer)
	if err != nil {
		t.Fatalf("re-approve: %v", err)
	}
	if reapproved.NetworkRevision == nil || reapproved.NetworkSnapshotHash == "" {
		t.Fatal("re-approval did not bind new snapshot")
	}
	if _, err := h.simulationSvc.Start(ctx, scenario.ID, h.actorEng); err != nil {
		t.Fatalf("simulation must recover after re-approval: %v", err)
	}

	// 历史结果保持原快照：两条 run 都在，且第一条快照仍含旧压力。
	var runs []model.SimulationRun
	h.db.Where("scenario_id = ?", scenario.ID).Order("id ASC").Find(&runs)
	if len(runs) != 2 {
		t.Fatalf("expected 2 immutable runs, got %d", len(runs))
	}
	var firstSnapshot struct {
		Nodes []model.VentilationNode `json:"nodes"`
	}
	if err := json.Unmarshal(runs[0].InputSnapshotJSON, &firstSnapshot); err != nil {
		t.Fatalf("decode first snapshot: %v", err)
	}
	foundOld := false
	for _, n := range firstSnapshot.Nodes {
		if n.Code == "WF1" && n.PressurePa == 500 {
			foundOld = true
		}
	}
	if !foundOld {
		t.Fatal("historical run snapshot was mutated by later network changes")
	}
}

func TestServiceSimulationRejectsUnapprovedScenario(t *testing.T) {
	h := newSnapshotHarness(t)
	ctx := context.Background()
	h.seedValidNetwork(t)
	curve := []dto.FanCurvePoint{{FlowM3S: 0, PressurePa: 900}, {FlowM3S: 30, PressurePa: 600}}
	sc, err := h.scenarioSvc.Create(ctx, dto.CreateFanScenarioRequest{
		Name: "草稿方案", Description: "未批准的草稿方案测试", FanCurve: curve, OperatingMode: "normal",
		SolverTolerance: 0.02, MaxIterations: 100,
	}, h.actorEng)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	_, err = h.simulationSvc.Start(ctx, sc.ID, h.actorEng)
	var appErr *api.AppError
	if !errors.As(err, &appErr) || appErr.Code != "SCENARIO_SNAPSHOT_STALE" {
		t.Fatalf("draft scenario must be rejected at entry, got %v", err)
	}
}
