package service

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"mine-ventilation-network-simulator/backend/internal/constants"
	"mine-ventilation-network-simulator/backend/internal/dto"
	"mine-ventilation-network-simulator/backend/internal/model"
	"mine-ventilation-network-simulator/backend/internal/repository"
	"mine-ventilation-network-simulator/backend/pkg/api"
)

var approvalTestDBSeq int64

type approvalTestEnv struct {
	db            *gorm.DB
	scenarios     *FanScenarioService
	nodes         *VentilationNodeService
	edges         *AirwayEdgeService
	runs          *SimulationService
	runRepo       *repository.SimulationRunRepository
	networkStates *repository.NetworkStateRepository
	engineer      Actor
	reviewer      Actor
	nodeIDs       map[string]uint
	edgeIDs       map[string]uint
}

func newApprovalTestEnv(t *testing.T) *approvalTestEnv {
	t.Helper()
	seq := atomic.AddInt64(&approvalTestDBSeq, 1)
	return newApprovalTestEnvWithDSN(t, fmt.Sprintf("file:approval-loop-%d?mode=memory&cache=shared&_busy_timeout=10000", seq))
}

// newApprovalTestEnvWithDSN 允许并发用例使用文件库：内存共享缓存模式的
// 表锁（SQLITE_LOCKED）不会触发 busy 等待，文件库才能让并发写事务真正串行。
func newApprovalTestEnvWithDSN(t *testing.T, dsn string) *approvalTestEnv {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		TranslateError: true,
		Logger:         gormlogger.Default.LogMode(gormlogger.Silent),
	})
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	if err := db.AutoMigrate(
		&model.User{}, &model.VentilationNode{}, &model.AirwayEdge{},
		&model.FanScenario{}, &model.SimulationRun{}, &model.AuditEvent{}, &model.NetworkState{},
	); err != nil {
		t.Fatalf("migrate test database: %v", err)
	}
	t.Cleanup(func() {
		if sqlDB, err := db.DB(); err == nil {
			_ = sqlDB.Close()
		}
	})

	nodeRepo := repository.NewVentilationNodeRepository(db)
	edgeRepo := repository.NewAirwayEdgeRepository(db)
	scenarioRepo := repository.NewFanScenarioRepository(db)
	runRepo := repository.NewSimulationRunRepository(db)
	env := &approvalTestEnv{
		db:            db,
		scenarios:     NewFanScenarioService(scenarioRepo),
		nodes:         NewVentilationNodeService(nodeRepo, edgeRepo),
		edges:         NewAirwayEdgeService(edgeRepo, nodeRepo),
		runs:          NewSimulationService(runRepo, scenarioRepo),
		runRepo:       runRepo,
		networkStates: repository.NewNetworkStateRepository(db),
		engineer:      Actor{ID: 11, Email: "engineer@mine.local", Role: string(constants.RoleEngineer), RequestID: "req-engineer"},
		reviewer:      Actor{ID: 12, Email: "reviewer@mine.local", Role: string(constants.RoleReviewer), RequestID: "req-reviewer"},
		nodeIDs:       map[string]uint{},
		edgeIDs:       map[string]uint{},
	}
	env.seedNetwork(t)
	return env
}

func (e *approvalTestEnv) seedNetwork(t *testing.T) {
	t.Helper()
	ctx := context.Background()
	nodes := []dto.CreateVentilationNodeRequest{
		{Code: "INT-01", NodeType: "intake", ElevationM: 12, PressurePa: 1250, Status: "active"},
		{Code: "JCT-12", NodeType: "junction", ElevationM: -85, PressurePa: 680, Status: "active"},
		{Code: "WF-07", NodeType: "workface", ElevationM: -126, RequiredAirflowM3S: 18, PressurePa: 410, Status: "active"},
		{Code: "EXT-02", NodeType: "exhaust", ElevationM: 6, PressurePa: 0, Status: "active"},
	}
	for _, input := range nodes {
		node, err := e.nodes.Create(ctx, input, e.engineer)
		if err != nil {
			t.Fatalf("seed node %s: %v", input.Code, err)
		}
		e.nodeIDs[node.Code] = node.ID
	}
	enabled := true
	edges := []dto.CreateAirwayEdgeRequest{
		{Code: "AW-101", FromNodeID: e.nodeIDs["INT-01"], ToNodeID: e.nodeIDs["JCT-12"], ResistanceNS2M8: 1.8, AreaM2: 8.2, MaxVelocityMS: 8, DoorState: "open", Enabled: &enabled, CriticalPath: true},
		{Code: "AW-102", FromNodeID: e.nodeIDs["JCT-12"], ToNodeID: e.nodeIDs["WF-07"], ResistanceNS2M8: 2.4, AreaM2: 6.4, MaxVelocityMS: 7, DoorState: "open", Enabled: &enabled, CriticalPath: true},
		{Code: "AW-103", FromNodeID: e.nodeIDs["WF-07"], ToNodeID: e.nodeIDs["EXT-02"], ResistanceNS2M8: 2.1, AreaM2: 7.0, MaxVelocityMS: 8, DoorState: "open", Enabled: &enabled, CriticalPath: true},
	}
	for _, input := range edges {
		edge, err := e.edges.Create(ctx, input, e.engineer)
		if err != nil {
			t.Fatalf("seed edge %s: %v", input.Code, err)
		}
		e.edgeIDs[edge.Code] = edge.ID
	}
}

func (e *approvalTestEnv) approveScenario(t *testing.T) *model.FanScenario {
	t.Helper()
	ctx := context.Background()
	scenario, err := e.scenarios.Create(ctx, dto.CreateFanScenarioRequest{
		Name: "闭环验证方案", Description: "验证批准快照绑定与失效闭环",
		FanCurve: []dto.FanCurvePoint{
			{FlowM3S: 0, PressurePa: 1450},
			{FlowM3S: 30, PressurePa: 1180},
			{FlowM3S: 60, PressurePa: 720},
		},
		OperatingMode: "normal", SolverTolerance: 0.02, MaxIterations: 100,
	}, e.engineer)
	if err != nil {
		t.Fatalf("create scenario: %v", err)
	}
	scenario, err = e.scenarios.Transition(ctx, scenario.ID, dto.TransitionScenarioRequest{TargetStatus: "pending_review", Version: scenario.Version}, e.engineer)
	if err != nil {
		t.Fatalf("submit scenario: %v", err)
	}
	scenario, err = e.scenarios.Transition(ctx, scenario.ID, dto.TransitionScenarioRequest{TargetStatus: "approved", Version: scenario.Version}, e.reviewer)
	if err != nil {
		t.Fatalf("approve scenario: %v", err)
	}
	return scenario
}

func (e *approvalTestEnv) currentFingerprint(t *testing.T) string {
	t.Helper()
	var nodes []model.VentilationNode
	if err := e.db.Order("id ASC").Find(&nodes).Error; err != nil {
		t.Fatalf("load nodes: %v", err)
	}
	var edges []model.AirwayEdge
	if err := e.db.Order("id ASC").Find(&edges).Error; err != nil {
		t.Fatalf("load edges: %v", err)
	}
	return model.ComputeNetworkFingerprint(nodes, edges)
}

func (e *approvalTestEnv) updateWorkfacePressure(t *testing.T, pressure float64) {
	t.Helper()
	_, err := e.nodes.Update(context.Background(), e.nodeIDs["WF-07"], dto.UpdateVentilationNodeRequest{
		NodeType: "workface", ElevationM: -126, RequiredAirflowM3S: 18, PressurePa: pressure, Status: "active",
	}, e.engineer)
	if err != nil {
		t.Fatalf("update workface pressure: %v", err)
	}
}

func appErrorCode(t *testing.T, err error) string {
	t.Helper()
	var appErr *api.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected *api.AppError, got %T: %v", err, err)
	}
	return appErr.Code
}

// TestApprovalSnapshotClosedLoop 验证完整闭环：批准绑定快照 → 无变化可推演 →
// 网络变更立即失效并拒绝推演 → 复核员重新批准恢复 → 历史结果保持原快照。
func TestApprovalSnapshotClosedLoop(t *testing.T) {
	env := newApprovalTestEnv(t)
	ctx := context.Background()
	scenario := env.approveScenario(t)

	// 1. 批准时绑定了当时的网络快照
	if scenario.ApprovedNetworkFingerprint == "" || scenario.ApprovedNetworkRevision == 0 {
		t.Fatalf("approval must bind network snapshot, got revision=%d fingerprint=%q",
			scenario.ApprovedNetworkRevision, scenario.ApprovedNetworkFingerprint)
	}
	if want := env.currentFingerprint(t); scenario.ApprovedNetworkFingerprint != want {
		t.Fatalf("bound fingerprint %s != current network %s", scenario.ApprovedNetworkFingerprint, want)
	}

	// 2. 无变化仍可推演
	firstRun, err := env.runs.Start(ctx, scenario.ID, env.engineer)
	if err != nil {
		t.Fatalf("simulation with unchanged network must start: %v", err)
	}

	// 3. 节点参数变化后旧批准立即失效
	env.updateWorkfacePressure(t, 555)
	invalidated, err := env.scenarios.Get(ctx, scenario.ID)
	if err != nil {
		t.Fatalf("reload scenario: %v", err)
	}
	if invalidated.ScenarioStatus != string(constants.ScenarioStatusPendingReview) {
		t.Fatalf("approval must be invalidated after network change, got %s", invalidated.ScenarioStatus)
	}
	if !strings.Contains(invalidated.InvalidationReason, "WF-07") {
		t.Fatalf("invalidation reason must name the changed node, got %q", invalidated.InvalidationReason)
	}
	if invalidated.Version != scenario.Version+1 {
		t.Fatalf("invalidation must bump version: got %d want %d", invalidated.Version, scenario.Version+1)
	}
	state, err := env.networkStates.Current(ctx)
	if err != nil {
		t.Fatalf("load network state: %v", err)
	}
	if state.Revision != scenario.ApprovedNetworkRevision+1 {
		t.Fatalf("network revision must advance: got %d want %d", state.Revision, scenario.ApprovedNetworkRevision+1)
	}

	// 4. 推演入口拒绝启动并提示重新复核
	if _, err := env.runs.Start(ctx, scenario.ID, env.engineer); err == nil {
		t.Fatal("simulation must be rejected after approval invalidation")
	} else {
		if code := appErrorCode(t, err); code != "APPROVAL_INVALIDATED" {
			t.Fatalf("expected APPROVAL_INVALIDATED, got %s (%v)", code, err)
		}
		var appErr *api.AppError
		_ = errors.As(err, &appErr)
		if !strings.Contains(appErr.Message, "重新批准") {
			t.Fatalf("rejection must prompt re-review, got %q", appErr.Message)
		}
	}

	// 5. 失效写入不可变审计，操作者是变更网络的人
	var audits []model.AuditEvent
	if err := env.db.Where("action = ? AND entity_id = ?", "fan_scenario.approval_invalidated", scenario.ID).Find(&audits).Error; err != nil {
		t.Fatalf("list invalidation audits: %v", err)
	}
	if len(audits) != 1 {
		t.Fatalf("expected exactly one invalidation audit, got %d", len(audits))
	}
	if audits[0].ActorEmail != env.engineer.Email || audits[0].RequestID != env.engineer.RequestID {
		t.Fatalf("invalidation audit must record the mutating actor: %+v", audits[0])
	}
	if !strings.Contains(audits[0].Metadata, "WF-07") {
		t.Fatalf("invalidation audit metadata must carry the reason: %s", audits[0].Metadata)
	}

	// 6. 复核员重新批准后恢复，绑定新的网络快照
	reapproved, err := env.scenarios.Transition(ctx, scenario.ID, dto.TransitionScenarioRequest{TargetStatus: "approved", Version: invalidated.Version}, env.reviewer)
	if err != nil {
		t.Fatalf("reviewer re-approval must succeed: %v", err)
	}
	if reapproved.InvalidationReason != "" {
		t.Fatalf("re-approval must clear invalidation reason, got %q", reapproved.InvalidationReason)
	}
	if want := env.currentFingerprint(t); reapproved.ApprovedNetworkFingerprint != want {
		t.Fatal("re-approval must bind the changed network fingerprint")
	}
	if reapproved.ApprovedNetworkRevision != state.Revision {
		t.Fatalf("re-approval must bind revision %d, got %d", state.Revision, reapproved.ApprovedNetworkRevision)
	}
	secondRun, err := env.runs.Start(ctx, scenario.ID, env.engineer)
	if err != nil {
		t.Fatalf("simulation after re-approval must start: %v", err)
	}

	// 7. 历史结果保持原快照，新结果使用新快照
	persistedFirst, err := env.runRepo.Find(ctx, firstRun.ID)
	if err != nil {
		t.Fatalf("reload first run: %v", err)
	}
	if !strings.Contains(string(persistedFirst.InputSnapshotJSON), `"pressure_pa":410`) {
		t.Fatal("historical run must keep the original network snapshot")
	}
	if strings.Contains(string(persistedFirst.InputSnapshotJSON), `"pressure_pa":555`) {
		t.Fatal("historical run snapshot must not be rewritten by later changes")
	}
	persistedSecond, err := env.runRepo.Find(ctx, secondRun.ID)
	if err != nil {
		t.Fatalf("reload second run: %v", err)
	}
	if !strings.Contains(string(persistedSecond.InputSnapshotJSON), `"pressure_pa":555`) {
		t.Fatal("new run must snapshot the changed network")
	}
}

// TestNetworkMutationsInvalidateApproval 覆盖新增、停用、参数变化与无变化保存。
func TestNetworkMutationsInvalidateApproval(t *testing.T) {
	boolPtr := func(v bool) *bool { return &v }
	cases := []struct {
		name              string
		mutate            func(t *testing.T, env *approvalTestEnv)
		expectInvalidated bool
	}{
		{"新增节点", func(t *testing.T, env *approvalTestEnv) {
			_, err := env.nodes.Create(context.Background(), dto.CreateVentilationNodeRequest{
				Code: "JCT-88", NodeType: "junction", ElevationM: -50, Status: "active",
			}, env.engineer)
			if err != nil {
				t.Fatalf("create node: %v", err)
			}
		}, true},
		{"停用节点", func(t *testing.T, env *approvalTestEnv) {
			_, err := env.nodes.Update(context.Background(), env.nodeIDs["JCT-12"], dto.UpdateVentilationNodeRequest{
				NodeType: "junction", ElevationM: -85, PressurePa: 680, Status: "inactive",
			}, env.engineer)
			if err != nil {
				t.Fatalf("deactivate node: %v", err)
			}
		}, true},
		{"节点参数变化", func(t *testing.T, env *approvalTestEnv) {
			_, err := env.nodes.Update(context.Background(), env.nodeIDs["JCT-12"], dto.UpdateVentilationNodeRequest{
				NodeType: "junction", ElevationM: -90, PressurePa: 680, Status: "active",
			}, env.engineer)
			if err != nil {
				t.Fatalf("update node: %v", err)
			}
		}, true},
		{"无变化节点保存", func(t *testing.T, env *approvalTestEnv) {
			_, err := env.nodes.Update(context.Background(), env.nodeIDs["JCT-12"], dto.UpdateVentilationNodeRequest{
				NodeType: "junction", ElevationM: -85, PressurePa: 680, Status: "active",
			}, env.engineer)
			if err != nil {
				t.Fatalf("noop node save: %v", err)
			}
		}, false},
		{"新增巷道", func(t *testing.T, env *approvalTestEnv) {
			_, err := env.edges.Create(context.Background(), dto.CreateAirwayEdgeRequest{
				Code: "AW-105", FromNodeID: env.nodeIDs["JCT-12"], ToNodeID: env.nodeIDs["EXT-02"],
				ResistanceNS2M8: 3.6, AreaM2: 5.8, MaxVelocityMS: 6, DoorState: "regulating", Enabled: boolPtr(true),
			}, env.engineer)
			if err != nil {
				t.Fatalf("create edge: %v", err)
			}
		}, true},
		{"停用巷道", func(t *testing.T, env *approvalTestEnv) {
			env.updateEdge(t, func(edge *model.AirwayEdge, input *dto.UpdateAirwayEdgeRequest) {
				input.Enabled = boolPtr(false)
			})
		}, true},
		{"巷道参数变化", func(t *testing.T, env *approvalTestEnv) {
			env.updateEdge(t, func(edge *model.AirwayEdge, input *dto.UpdateAirwayEdgeRequest) {
				input.ResistanceNS2M8 = 2.3
			})
		}, true},
		{"无变化巷道保存", func(t *testing.T, env *approvalTestEnv) {
			env.updateEdge(t, func(edge *model.AirwayEdge, input *dto.UpdateAirwayEdgeRequest) {})
		}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			env := newApprovalTestEnv(t)
			ctx := context.Background()
			scenario := env.approveScenario(t)
			tc.mutate(t, env)
			updated, err := env.scenarios.Get(ctx, scenario.ID)
			if err != nil {
				t.Fatalf("reload scenario: %v", err)
			}
			if tc.expectInvalidated {
				if updated.ScenarioStatus != string(constants.ScenarioStatusPendingReview) {
					t.Fatalf("%s 后批准必须失效，当前状态 %s", tc.name, updated.ScenarioStatus)
				}
				if updated.InvalidationReason == "" {
					t.Fatalf("%s 后必须记录失效原因", tc.name)
				}
				if _, err := env.runs.Start(ctx, scenario.ID, env.engineer); err == nil {
					t.Fatalf("%s 后推演入口必须拒绝启动", tc.name)
				}
			} else {
				if updated.ScenarioStatus != string(constants.ScenarioStatusApproved) {
					t.Fatalf("%s 不应使批准失效，当前状态 %s", tc.name, updated.ScenarioStatus)
				}
				if _, err := env.runs.Start(ctx, scenario.ID, env.engineer); err != nil {
					t.Fatalf("%s 后无变化必须仍可推演：%v", tc.name, err)
				}
			}
		})
	}
}

func (e *approvalTestEnv) updateEdge(t *testing.T, mutate func(edge *model.AirwayEdge, input *dto.UpdateAirwayEdgeRequest)) {
	t.Helper()
	edge, err := e.edges.Get(context.Background(), e.edgeIDs["AW-101"])
	if err != nil {
		t.Fatalf("load edge: %v", err)
	}
	enabled := edge.Enabled
	input := dto.UpdateAirwayEdgeRequest{
		ResistanceNS2M8: edge.ResistanceNS2M8, AreaM2: edge.AreaM2, MaxVelocityMS: edge.MaxVelocityMS,
		DoorState: edge.DoorState, Enabled: &enabled, CriticalPath: edge.CriticalPath, Version: edge.Version,
	}
	mutate(edge, &input)
	if _, err := e.edges.Update(context.Background(), edge.ID, input, e.engineer); err != nil {
		t.Fatalf("update edge: %v", err)
	}
}

// TestConcurrentApprovalAndMutationNeverStale 并发变更与重新批准交错执行，
// 任何时刻处于已批准状态的方案都必须绑定当前网络指纹，旧批准不得漏失效。
func TestConcurrentApprovalAndMutationNeverStale(t *testing.T) {
	env := newApprovalTestEnvWithDSN(t, fmt.Sprintf("file:%s?_busy_timeout=10000",
		filepath.Join(t.TempDir(), "approval-race.db")))
	ctx := context.Background()
	scenario := env.approveScenario(t)

	allowedStartErrors := map[string]bool{
		"APPROVAL_INVALIDATED":   true,
		"SCENARIO_NOT_APPROVED":  true,
		"APPROVAL_STALE_NETWORK": true,
	}
	for round := 0; round < 12; round++ {
		// 每轮把方案重置回待复核，制造“变更与重新批准并发”的窗口。
		if err := env.db.Model(&model.FanScenario{}).Where("id = ?", scenario.ID).
			Update("scenario_status", "pending_review").Error; err != nil {
			t.Fatalf("round %d reset: %v", round, err)
		}
		current, err := env.scenarios.Get(ctx, scenario.ID)
		if err != nil {
			t.Fatalf("round %d reload: %v", round, err)
		}
		pressure := 600 + float64(round)

		var wg sync.WaitGroup
		var approveErr, mutateErr, startErr error
		wg.Add(3)
		go func() {
			defer wg.Done()
			_, approveErr = env.scenarios.Transition(ctx, scenario.ID,
				dto.TransitionScenarioRequest{TargetStatus: "approved", Version: current.Version}, env.reviewer)
		}()
		go func() {
			defer wg.Done()
			_, mutateErr = env.nodes.Update(ctx, env.nodeIDs["JCT-12"], dto.UpdateVentilationNodeRequest{
				NodeType: "junction", ElevationM: -85, PressurePa: pressure, Status: "active",
			}, env.engineer)
		}()
		go func() {
			defer wg.Done()
			_, startErr = env.runs.Start(ctx, scenario.ID, env.engineer)
		}()
		wg.Wait()

		if mutateErr != nil {
			t.Fatalf("round %d mutation must succeed: %v", round, mutateErr)
		}
		if approveErr != nil {
			if code := appErrorCode(t, approveErr); code != "VERSION_CONFLICT" {
				t.Fatalf("round %d approval failed unexpectedly: %v", round, approveErr)
			}
		}
		if startErr != nil {
			if code := appErrorCode(t, startErr); !allowedStartErrors[code] {
				t.Fatalf("round %d simulation start failed unexpectedly: %v", round, startErr)
			}
		}

		final, err := env.scenarios.Get(ctx, scenario.ID)
		if err != nil {
			t.Fatalf("round %d final reload: %v", round, err)
		}
		if final.ScenarioStatus == string(constants.ScenarioStatusApproved) {
			if fingerprint := env.currentFingerprint(t); final.ApprovedNetworkFingerprint != fingerprint {
				t.Fatalf("round %d: 旧批准漏失效，绑定指纹 %s，当前网络指纹 %s",
					round, final.ApprovedNetworkFingerprint, fingerprint)
			}
		}
	}
}
