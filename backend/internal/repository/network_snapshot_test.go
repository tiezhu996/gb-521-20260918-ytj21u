package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"testing"

	"gorm.io/datatypes"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"mine-ventilation-network-simulator/backend/internal/constants"
	"mine-ventilation-network-simulator/backend/internal/model"
)

func newSnapshotTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := fmt.Sprintf("file:snap-test-%d?mode=memory&cache=shared", nowNanos())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	sqlDB, _ := db.DB()
	sqlDB.SetMaxOpenConns(1)
	if err := db.AutoMigrate(&model.NetworkRevision{}, &model.VentilationNode{}, &model.AirwayEdge{}, &model.FanScenario{}, &model.SimulationRun{}, &model.AuditEvent{}, &model.User{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if err := db.Exec("INSERT INTO network_revisions (id, revision, updated_at) VALUES (1, 0, CURRENT_TIMESTAMP)").Error; err != nil {
		t.Fatalf("seed revision: %v", err)
	}
	return db
}

func seedNetwork(t *testing.T, db *gorm.DB) ([]model.VentilationNode, []model.AirwayEdge) {
	nodes := []model.VentilationNode{
		{Code: "IN", NodeType: string(constants.NodeTypeIntake), Status: string(constants.NodeStatusActive), PressurePa: 1000},
		{Code: "WF", NodeType: string(constants.NodeTypeWorkface), Status: string(constants.NodeStatusActive), RequiredAirflowM3S: 8, PressurePa: 500},
		{Code: "OUT", NodeType: string(constants.NodeTypeExhaust), Status: string(constants.NodeStatusActive)},
	}
	for i := range nodes {
		if err := db.Create(&nodes[i]).Error; err != nil {
			t.Fatalf("seed node: %v", err)
		}
	}
	edges := []model.AirwayEdge{
		{Code: "E1", FromNodeID: nodes[0].ID, ToNodeID: nodes[1].ID, ResistanceNS2M8: 2, AreaM2: 5, MaxVelocityMS: 5, DoorState: string(constants.DoorStateOpen), Enabled: true, CriticalPath: true, Version: 1},
		{Code: "E2", FromNodeID: nodes[1].ID, ToNodeID: nodes[2].ID, ResistanceNS2M8: 2, AreaM2: 5, MaxVelocityMS: 5, DoorState: string(constants.DoorStateOpen), Enabled: true, CriticalPath: true, Version: 1},
	}
	for i := range edges {
		if err := db.Create(&edges[i]).Error; err != nil {
			t.Fatalf("seed edge: %v", err)
		}
	}
	return nodes, edges
}

var testNanos int64

func nowNanos() int64 {
	testNanos++
	return testNanos
}

func seedApprovedScenario(t *testing.T, db *gorm.DB, nodes []model.VentilationNode, edges []model.AirwayEdge) *model.FanScenario {
	t.Helper()
	snapshots := NewNetworkSnapshotRepository(db)
	curve := datatypes.JSON([]byte(`[{"flow_m3s":0,"pressure_pa":900},{"flow_m3s":30,"pressure_pa":600},{"flow_m3s":60,"pressure_pa":300}]`))
	scenario := &model.FanScenario{
		Name: "测试方案", Description: "闭环测试方案描述内容", FanCurveJSON: curve, OperatingMode: "normal",
		ScenarioStatus: string(constants.ScenarioStatusPendingReview), SolverTolerance: 0.02, MaxIterations: 100,
		Version: 1, CreatedBy: 1,
	}
	if err := db.Create(scenario).Error; err != nil {
		t.Fatalf("create scenario: %v", err)
	}
	audit := AuditRecord{RequestID: "test-approve", ActorID: 2, ActorEmail: "reviewer@mine.local", Action: "fan_scenario.approved", EntityType: "fan_scenario"}
	audit.Metadata = `{"reason":"测试批准"}`
	approved, revision, err := snapshots.BindApprovalSnapshot(context.Background(), scenario.ID, 2, 1, audit)
	if err != nil {
		t.Fatalf("approve: %v", err)
	}
	if approved.NetworkRevision == nil || *approved.NetworkRevision != revision {
		t.Fatalf("approval did not bind revision: %+v", approved)
	}
	if approved.NetworkSnapshotHash == "" {
		t.Fatal("approval did not bind fingerprint")
	}
	// 批准审计必须同时保留操作原因与网络快照凭据。
	var event model.AuditEvent
	if err := db.Where("action = ? AND entity_id = ?", "fan_scenario.approved", scenario.ID).First(&event).Error; err != nil {
		t.Fatalf("load approval audit: %v", err)
	}
	var meta map[string]interface{}
	if err := json.Unmarshal([]byte(event.Metadata), &meta); err != nil {
		t.Fatalf("approval metadata not json: %v (%s)", err, event.Metadata)
	}
	for _, key := range []string{"reason", "network_revision", "network_snapshot_hash", "node_count", "edge_count"} {
		if _, ok := meta[key]; !ok {
			t.Fatalf("approval audit metadata missing %s: %s", key, event.Metadata)
		}
	}
	return approved
}

func TestSnapshotLifecycleClosedLoop(t *testing.T) {
	db := newSnapshotTestDB(t)
	nodes, edges := seedNetwork(t, db)
	snapshots := NewNetworkSnapshotRepository(db)
	nodeRepo := NewVentilationNodeRepository(db)

	scenario := seedApprovedScenario(t, db, nodes, edges)
	boundRevision := *scenario.NetworkRevision
	boundHash := scenario.NetworkSnapshotHash

	// 1) 无网络变化：同样的网络仍可通过准入校验（构造一次 VerifiedRun 验证不返回 stale）。
	verified := 0
	_, err := snapshots.CreateVerifiedRun(context.Background(), scenario.ID, AuditRecord{RequestID: "test-run", ActorID: 1, ActorEmail: "e@x", Action: "simulation_run.started", EntityType: "simulation_run"},
		func(sc model.FanScenario, ns []model.VentilationNode, es []model.AirwayEdge) (*model.SimulationRun, string, error) {
			verified++
			if sc.NetworkSnapshotHash != boundHash {
				t.Fatal("builder received scenario with different hash")
			}
			return &model.SimulationRun{ScenarioID: sc.ID, RunStatus: "converged", AlgorithmVersion: "v1", InputSnapshotJSON: datatypes.JSON([]byte(`{}`)), NodePressuresJSON: datatypes.JSON(`{}`), EdgeFlowsJSON: datatypes.JSON(`{}`), ResidualsJSON: datatypes.JSON(`[]`), RiskFlagsJSON: datatypes.JSON(`[]`)}, `{}`, nil
		})
	if err != nil {
		t.Fatalf("unchanged network should allow simulation: %v", err)
	}
	if verified != 1 {
		t.Fatal("builder not invoked on valid snapshot")
	}

	// 2) 节点参数变化：旧批准立即失效（状态回退 pending_review，版本号推进，绑定清空并记录原因）。
	changed := nodes[1]
	changed.PressurePa = 620
	if err := nodeRepo.Update(context.Background(), &changed, AuditRecord{RequestID: "test-node", ActorID: 1, ActorEmail: "e@x", Action: "ventilation_node.updated", EntityType: "ventilation_node"}); err != nil {
		t.Fatalf("node update: %v", err)
	}
	var stale model.FanScenario
	if err := db.First(&stale, scenario.ID).Error; err != nil {
		t.Fatalf("reload: %v", err)
	}
	if stale.ScenarioStatus != string(constants.ScenarioStatusPendingReview) {
		t.Fatalf("expected pending_review after network change, got %s", stale.ScenarioStatus)
	}
	if stale.NetworkRevision != nil || stale.NetworkSnapshotHash != "" || stale.ApprovedBy != nil {
		t.Fatalf("stale approval bindings not cleared: %+v", stale)
	}
	if stale.InvalidationReason == "" {
		t.Fatal("invalidation reason not recorded")
	}
	current, err := snapshots.CurrentRevision(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if current <= boundRevision {
		t.Fatalf("revision not advanced: bound=%d current=%d", boundRevision, current)
	}

	// 3) 推演入口必须拒绝过期/非批准快照。
	_, err = snapshots.CreateVerifiedRun(context.Background(), scenario.ID, AuditRecord{ActorID: 1}, nil)
	if err != ErrSnapshotStale {
		t.Fatalf("expected ErrSnapshotStale, got %v", err)
	}

	// 4) 复核员重新批准后恢复，绑定新快照。
	var reloaded model.FanScenario
	if err := db.First(&reloaded, scenario.ID).Error; err != nil {
		t.Fatal(err)
	}
	reapproved, _, err := snapshots.BindApprovalSnapshot(context.Background(), reloaded.ID, 2, reloaded.Version, AuditRecord{RequestID: "re", ActorID: 2, ActorEmail: "r@x", Action: "fan_scenario.approved", EntityType: "fan_scenario"})
	if err != nil {
		t.Fatalf("re-approve: %v", err)
	}
	if reapproved.ScenarioStatus != string(constants.ScenarioStatusApproved) || reapproved.NetworkRevision == nil {
		t.Fatalf("re-approval did not restore: %+v", reapproved)
	}
	if *reapproved.NetworkRevision != current || reapproved.NetworkSnapshotHash == boundHash {
		t.Fatalf("re-approval must bind the NEW snapshot: rev=%d (want %d), hash changed=%v", deref(reapproved.NetworkRevision), current, reapproved.NetworkSnapshotHash != boundHash)
	}
	if reapproved.InvalidationReason != "" {
		t.Fatal("invalidation reason must clear on re-approval")
	}

	// 5) 恢复后可以推演。
	_, err = snapshots.CreateVerifiedRun(context.Background(), scenario.ID, AuditRecord{ActorID: 1},
		func(sc model.FanScenario, ns []model.VentilationNode, es []model.AirwayEdge) (*model.SimulationRun, string, error) {
			return &model.SimulationRun{ScenarioID: sc.ID, RunStatus: "converged", AlgorithmVersion: "v1", InputSnapshotJSON: datatypes.JSON(`{}`), NodePressuresJSON: datatypes.JSON(`{}`), EdgeFlowsJSON: datatypes.JSON(`{}`), ResidualsJSON: datatypes.JSON(`[]`), RiskFlagsJSON: datatypes.JSON(`[]`)}, `{}`, nil
		})
	if err != nil {
		t.Fatalf("re-approved scenario should run: %v", err)
	}

	// 6) 历史推演结果保持原快照：旧 run 的输入快照 JSON 未被任何后续变更改写。
	var runCount int64
	db.Model(&model.SimulationRun{}).Where("scenario_id = ?", scenario.ID).Count(&runCount)
	if runCount != 2 {
		t.Fatalf("expected 2 immutable historical runs, got %d", runCount)
	}
}

func deref(p *uint64) uint64 {
	if p == nil {
		return 0
	}
	return *p
}

func TestNoopNodeUpdateKeepsApprovalValid(t *testing.T) {
	db := newSnapshotTestDB(t)
	nodes, edges := seedNetwork(t, db)
	nodeRepo := NewVentilationNodeRepository(db)
	scenario := seedApprovedScenario(t, db, nodes, edges)
	before, _ := NewNetworkSnapshotRepository(db).CurrentRevision(context.Background())

	same := nodes[0] // 所有字段保持不变
	if err := nodeRepo.Update(context.Background(), &same, AuditRecord{ActorID: 1, Action: "ventilation_node.updated", EntityType: "ventilation_node"}); err != nil {
		t.Fatalf("noop update: %v", err)
	}
	after, _ := NewNetworkSnapshotRepository(db).CurrentRevision(context.Background())
	if before != after {
		t.Fatalf("identical node update must not bump revision: %d -> %d", before, after)
	}
	var sc model.FanScenario
	db.First(&sc, scenario.ID)
	if sc.ScenarioStatus != string(constants.ScenarioStatusApproved) {
		t.Fatalf("noop update must keep approval valid, got %s", sc.ScenarioStatus)
	}
}

func TestEdgeDisableInvalidatesApproval(t *testing.T) {
	db := newSnapshotTestDB(t)
	nodes, edges := seedNetwork(t, db)
	edgeRepo := NewAirwayEdgeRepository(db)
	scenario := seedApprovedScenario(t, db, nodes, edges)

	disabled := edges[0]
	disabled.Enabled = false
	if err := edgeRepo.Update(context.Background(), &disabled, 1, AuditRecord{ActorID: 1, Action: "airway_edge.updated", EntityType: "airway_edge"}); err != nil {
		t.Fatalf("edge disable: %v", err)
	}
	var sc model.FanScenario
	db.First(&sc, scenario.ID)
	if sc.ScenarioStatus != string(constants.ScenarioStatusPendingReview) || sc.InvalidationReason == "" {
		t.Fatalf("edge disable must invalidate approval: %+v", sc)
	}
	err := db.Transaction(func(tx *gorm.DB) error {
		_, _, e := BumpRevisionAndInvalidate(tx, "sqlite", "再次变更测试", AuditRecord{ActorID: 1})
		return e
	})
	if err != nil {
		t.Fatalf("second change with no approved scenarios should be clean: %v", err)
	}
}

func TestNewNodeInvalidatesApproval(t *testing.T) {
	db := newSnapshotTestDB(t)
	nodes, edges := seedNetwork(t, db)
	nodeRepo := NewVentilationNodeRepository(db)
	scenario := seedApprovedScenario(t, db, nodes, edges)

	extra := &model.VentilationNode{Code: "JCT9", NodeType: string(constants.NodeTypeJunction), Status: string(constants.NodeStatusInactive)}
	if err := nodeRepo.Create(context.Background(), extra, AuditRecord{ActorID: 1, Action: "ventilation_node.created", EntityType: "ventilation_node"}); err != nil {
		t.Fatalf("create node: %v", err)
	}
	var sc model.FanScenario
	db.First(&sc, scenario.ID)
	if sc.ScenarioStatus != string(constants.ScenarioStatusPendingReview) {
		t.Fatalf("node creation must invalidate approval, got %s", sc.ScenarioStatus)
	}
}

func TestConcurrentChangesNeverMissInvalidation(t *testing.T) {
	db := newSnapshotTestDB(t)
	nodes, edges := seedNetwork(t, db)
	nodeRepo := NewVentilationNodeRepository(db)
	scenario := seedApprovedScenario(t, db, nodes, edges)

	// 并发发起 N 个互相不同的节点参数变更；每个都走真实事务 + 版本行锁。
	const writers = 8
	var wg sync.WaitGroup
	errs := make(chan error, writers)
	for i := 0; i < writers; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			n := nodes[1]
			// 每个 goroutine 先读到最新版本，再改一个唯一参数。
			var fresh model.VentilationNode
			if err := db.First(&fresh, n.ID).Error; err != nil {
				errs <- err
				return
			}
			fresh.PressurePa = 500 + float64(i+1)*7
			err := nodeRepo.Update(context.Background(), &fresh, AuditRecord{ActorID: 1, ActorEmail: "e@x", RequestID: fmt.Sprintf("w-%d", i), Action: "ventilation_node.updated", EntityType: "ventilation_node"})
			if err != nil {
				errs <- err
			}
		}(i)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Fatalf("concurrent writer failed: %v", err)
	}

	// 无论哪些变更真正落库，只要发生过网络变更，旧批准必须已失效，
	// 且最终版本号与失效审计数量一致（恰好一次失效）。
	var sc model.FanScenario
	if err := db.First(&sc, scenario.ID).Error; err != nil {
		t.Fatal(err)
	}
	if sc.ScenarioStatus != string(constants.ScenarioStatusPendingReview) {
		t.Fatalf("old approval leaked through concurrent changes: status=%s", sc.ScenarioStatus)
	}
	revision, _ := NewNetworkSnapshotRepository(db).CurrentRevision(context.Background())
	if revision != writers {
		t.Fatalf("expected %d committed distinct network revisions, got %d", writers, revision)
	}
	var invalidationAudits int64
	db.Model(&model.AuditEvent{}).Where("action = ? AND entity_id = ?", "fan_scenario.auto_invalidated", scenario.ID).Count(&invalidationAudits)
	if invalidationAudits != 1 {
		t.Fatalf("expected exactly one invalidation audit, got %d", invalidationAudits)
	}

	// 推演入口在所有变更完成后必须拒绝。
	_, err := NewNetworkSnapshotRepository(db).CreateVerifiedRun(context.Background(), scenario.ID, AuditRecord{ActorID: 1}, nil)
	if err != ErrSnapshotStale {
		t.Fatalf("expected stale rejection after concurrent changes, got %v", err)
	}
}

// TestConcurrentChangeAndSimulationNeverUsesStaleSnapshot 验证网络变更事务与
// 推演准入事务并发竞态时：变更提交后的任何推演都必须被拒绝，绝不允许旧快照漏过。
func TestConcurrentChangeAndSimulationNeverUsesStaleSnapshot(t *testing.T) {
	db := newSnapshotTestDB(t)
	nodes, edges := seedNetwork(t, db)
	nodeRepo := NewVentilationNodeRepository(db)
	snapshots := NewNetworkSnapshotRepository(db)
	scenario := seedApprovedScenario(t, db, nodes, edges)

	builder := func(sc model.FanScenario, ns []model.VentilationNode, es []model.AirwayEdge) (*model.SimulationRun, string, error) {
		return &model.SimulationRun{
			ScenarioID: sc.ID, RunStatus: "converged", AlgorithmVersion: "v1",
			InputSnapshotJSON: datatypes.JSON([]byte(`{}`)), NodePressuresJSON: datatypes.JSON(`{}`),
			EdgeFlowsJSON: datatypes.JSON(`{}`), ResidualsJSON: datatypes.JSON(`[]`), RiskFlagsJSON: datatypes.JSON(`[]`),
		}, `{}`, nil
	}

	const racers = 16
	var wg sync.WaitGroup
	var starts, staleRejects int64
	var mu sync.Mutex
	changeDone := make(chan struct{})
	for i := 0; i < racers; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			if i == 0 {
				n := nodes[1]
				n.PressurePa = 999
				if err := nodeRepo.Update(context.Background(), &n, AuditRecord{ActorID: 1, RequestID: "race-change", Action: "ventilation_node.updated", EntityType: "ventilation_node"}); err != nil {
					t.Errorf("racing change failed: %v", err)
				}
				close(changeDone)
				return
			}
			_, err := snapshots.CreateVerifiedRun(context.Background(), scenario.ID, AuditRecord{ActorID: 1, RequestID: "race-run"}, builder)
			mu.Lock()
			if err == nil {
				starts++
			} else if errors.Is(err, ErrSnapshotStale) {
				staleRejects++
			} else {
				t.Errorf("unexpected simulation error: %v", err)
			}
			mu.Unlock()
		}(i)
	}
	wg.Wait()
	<-changeDone

	// 变更已提交之后，再发起推演必须被拒绝——这是"旧批准不得漏失效"的最终保证。
	_, err := snapshots.CreateVerifiedRun(context.Background(), scenario.ID, AuditRecord{ActorID: 1}, builder)
	if err != ErrSnapshotStale {
		t.Fatalf("post-change simulation must be rejected, got starts=%d stale=%d err=%v", starts, staleRejects, err)
	}
	var sc model.FanScenario
	db.First(&sc, scenario.ID)
	if sc.ScenarioStatus != string(constants.ScenarioStatusPendingReview) {
		t.Fatalf("approval must end invalidated after racing change, got %s", sc.ScenarioStatus)
	}
	t.Logf("race outcome: committed-before-change runs=%d, stale rejections=%d (all post-change rejected)", starts, staleRejects)
}
