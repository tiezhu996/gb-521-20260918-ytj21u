package repository

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"mine-ventilation-network-simulator/backend/internal/constants"
	"mine-ventilation-network-simulator/backend/internal/model"
)

type FanScenarioRepository struct{ db *gorm.DB }

func NewFanScenarioRepository(db *gorm.DB) *FanScenarioRepository {
	return &FanScenarioRepository{db: db}
}

func (r *FanScenarioRepository) List(ctx context.Context, page, pageSize int, status, search string) ([]model.FanScenario, int64, error) {
	query := r.db.WithContext(ctx).Model(&model.FanScenario{})
	if status != "" {
		query = query.Where("scenario_status = ?", status)
	}
	if search != "" {
		query = query.Where("LOWER(name) LIKE ?", "%"+strings.ToLower(search)+"%")
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count fan scenarios: %w", err)
	}
	var items []model.FanScenario
	if err := query.Order("updated_at DESC, id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&items).Error; err != nil {
		return nil, 0, fmt.Errorf("list fan scenarios: %w", err)
	}
	return items, total, nil
}

func (r *FanScenarioRepository) Find(ctx context.Context, id uint) (*model.FanScenario, error) {
	var scenario model.FanScenario
	if err := r.db.WithContext(ctx).First(&scenario, id).Error; err != nil {
		return nil, fmt.Errorf("find fan scenario: %w", err)
	}
	return &scenario, nil
}

func (r *FanScenarioRepository) Create(ctx context.Context, scenario *model.FanScenario, audit AuditRecord) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(scenario).Error; err != nil {
			return fmt.Errorf("create fan scenario: %w", err)
		}
		after, _ := json.Marshal(scenario)
		audit.EntityID = scenario.ID
		audit.AfterState = string(after)
		return writeAudit(tx, audit)
	})
}

func (r *FanScenarioRepository) Transition(ctx context.Context, id, actorID uint, expectedVersion uint, from, to, reason string, audit AuditRecord) (*model.FanScenario, error) {
	var updated model.FanScenario
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var before model.FanScenario
		if err := tx.First(&before, id).Error; err != nil {
			return fmt.Errorf("load fan scenario before transition: %w", err)
		}
		values := map[string]interface{}{
			"scenario_status":     to,
			"reject_reason":       "",
			"invalidation_reason": "",
			"version":             gorm.Expr("version + 1"),
		}
		if to == "approved" {
			values["approved_by"] = actorID
		}
		if to == "draft" {
			values["approved_by"] = nil
			values["reject_reason"] = reason
		}
		result := tx.Model(&model.FanScenario{}).
			Where("id = ? AND version = ? AND scenario_status = ?", id, expectedVersion, from).
			Updates(values)
		if result.Error != nil {
			return fmt.Errorf("transition fan scenario: %w", result.Error)
		}
		if result.RowsAffected != 1 {
			return ErrVersionConflict
		}
		if err := tx.First(&updated, id).Error; err != nil {
			return err
		}
		beforeJSON, _ := json.Marshal(before)
		afterJSON, _ := json.Marshal(updated)
		audit.EntityID = id
		audit.BeforeState = string(beforeJSON)
		audit.AfterState = string(afterJSON)
		return writeAudit(tx, audit)
	})
	if err != nil {
		return nil, err
	}
	return &updated, nil
}

// ErrSnapshotStale 表示方案批准时绑定的网络快照与当前网络不一致，必须重新复核。
var ErrSnapshotStale = errors.New("approved network snapshot is stale")

const networkRevisionSingletonID uint = 1

// NetworkSnapshotRepository 管理通风网络版本号、网络内容指纹，以及
// 网络变更、方案批准与推演准入之间的事务级同步。
type NetworkSnapshotRepository struct{ db *gorm.DB }

func NewNetworkSnapshotRepository(db *gorm.DB) *NetworkSnapshotRepository {
	return &NetworkSnapshotRepository{db: db}
}

// EnsureSingleton 保证单行网络版本计数器存在（版本从 0 开始）。
func (r *NetworkSnapshotRepository) EnsureSingleton(ctx context.Context) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return ensureNetworkRevisionRow(tx)
	})
}

func ensureNetworkRevisionRow(tx *gorm.DB) error {
	var row model.NetworkRevision
	err := tx.Where("id = ?", networkRevisionSingletonID).First(&row).Error
	if err == nil {
		return nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return fmt.Errorf("load network revision: %w", err)
	}
	if err := tx.Create(&model.NetworkRevision{ID: networkRevisionSingletonID, Revision: 0}).Error; err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return nil
		}
		return fmt.Errorf("seed network revision: %w", err)
	}
	return nil
}

// CurrentRevision 返回当前网络版本号。
func (r *NetworkSnapshotRepository) CurrentRevision(ctx context.Context) (uint64, error) {
	var row model.NetworkRevision
	if err := r.db.WithContext(ctx).Where("id = ?", networkRevisionSingletonID).First(&row).Error; err != nil {
		return 0, fmt.Errorf("read current network revision: %w", err)
	}
	return row.Revision, nil
}

type fingerprintNode struct {
	ID                 uint    `json:"id"`
	Code               string  `json:"code"`
	NodeType           string  `json:"node_type"`
	ElevationM         float64 `json:"elevation_m"`
	RequiredAirflowM3S float64 `json:"required_airflow_m3s"`
	PressurePa         float64 `json:"pressure_pa"`
	Status             string  `json:"status"`
}

type fingerprintEdge struct {
	ID              uint    `json:"id"`
	Code            string  `json:"code"`
	FromNodeID      uint    `json:"from_node_id"`
	ToNodeID        uint    `json:"to_node_id"`
	ResistanceNS2M8 float64 `json:"resistance_ns2_m8"`
	AreaM2          float64 `json:"area_m2"`
	MaxVelocityMS   float64 `json:"max_velocity_ms"`
	DoorState       string  `json:"door_state"`
	Enabled         bool    `json:"enabled"`
	CriticalPath    bool    `json:"critical_path"`
}

type fingerprintPayload struct {
	Nodes []fingerprintNode `json:"nodes"`
	Edges []fingerprintEdge `json:"edges"`
}

// ComputeNetworkFingerprint 对给定网络的拓扑与参数计算确定性 SHA-256 指纹，
// 作为批准快照除版本号之外的第二道一致性凭据。
func ComputeNetworkFingerprint(nodes []model.VentilationNode, edges []model.AirwayEdge) string {
	nodes = append([]model.VentilationNode(nil), nodes...)
	edges = append([]model.AirwayEdge(nil), edges...)
	sort.Slice(nodes, func(i, j int) bool { return nodes[i].ID < nodes[j].ID })
	sort.Slice(edges, func(i, j int) bool { return edges[i].ID < edges[j].ID })
	payload := fingerprintPayload{Nodes: make([]fingerprintNode, 0, len(nodes)), Edges: make([]fingerprintEdge, 0, len(edges))}
	for _, n := range nodes {
		payload.Nodes = append(payload.Nodes, fingerprintNode{ID: n.ID, Code: n.Code, NodeType: n.NodeType, ElevationM: n.ElevationM, RequiredAirflowM3S: n.RequiredAirflowM3S, PressurePa: n.PressurePa, Status: n.Status})
	}
	for _, e := range edges {
		payload.Edges = append(payload.Edges, fingerprintEdge{ID: e.ID, Code: e.Code, FromNodeID: e.FromNodeID, ToNodeID: e.ToNodeID, ResistanceNS2M8: e.ResistanceNS2M8, AreaM2: e.AreaM2, MaxVelocityMS: e.MaxVelocityMS, DoorState: e.DoorState, Enabled: e.Enabled, CriticalPath: e.CriticalPath})
	}
	data, _ := json.Marshal(payload)
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

// activeNetworkInTx 在事务内加载推演实际消费的活动节点与启用巷道。
func activeNetworkInTx(tx *gorm.DB) ([]model.VentilationNode, []model.AirwayEdge, error) {
	var nodes []model.VentilationNode
	if err := tx.Where("status = ?", string(constants.NodeStatusActive)).Order("id ASC").Find(&nodes).Error; err != nil {
		return nil, nil, fmt.Errorf("load active nodes: %w", err)
	}
	var edges []model.AirwayEdge
	if err := tx.Where("enabled = ?", true).Order("id ASC").Find(&edges).Error; err != nil {
		return nil, nil, fmt.Errorf("load enabled edges: %w", err)
	}
	return nodes, edges, nil
}

// lockNetworkRevision 锁定版本计数器单行，串行化所有网络写事务、批准事务与推演事务。
// PostgreSQL 使用 FOR UPDATE 行锁；SQLite 由连接池 MaxOpenConns(1) 保证串行。
func lockNetworkRevision(tx *gorm.DB, dialect string) (*model.NetworkRevision, error) {
	if err := ensureNetworkRevisionRow(tx); err != nil {
		return nil, err
	}
	query := tx.Where("id = ?", networkRevisionSingletonID)
	if dialect != "sqlite" {
		query.Clauses(clause.Locking{Strength: "UPDATE"})
	}
	var row model.NetworkRevision
	if err := query.First(&row).Error; err != nil {
		return nil, fmt.Errorf("lock network revision: %w", err)
	}
	return &row, nil
}

// BumpRevisionAndInvalidate 在网络变更事务内推进版本号并失效全部已批准方案，
// 为每个被失效的方案写审计。调用方必须已在同一事务内完成实体写入（锁顺序：
// 先实体后版本行也可，因为所有互斥方都最终争用版本行，且失效更新针对批准方案）。
// 返回新版本号与失效方案数。
func BumpRevisionAndInvalidate(tx *gorm.DB, dialect, reason string, auditTemplate AuditRecord) (uint64, int64, error) {
	row, err := lockNetworkRevision(tx, dialect)
	if err != nil {
		return 0, 0, err
	}
	newRevision := row.Revision + 1
	if err := tx.Model(&model.NetworkRevision{}).
		Where("id = ?", networkRevisionSingletonID).
		Updates(map[string]interface{}{"revision": newRevision, "updated_at": time.Now().UTC()}).Error; err != nil {
		return 0, 0, fmt.Errorf("bump network revision: %w", err)
	}

	var approved []model.FanScenario
	if err := tx.Where("scenario_status = ?", string(constants.ScenarioStatusApproved)).Find(&approved).Error; err != nil {
		return 0, 0, fmt.Errorf("load approved scenarios: %w", err)
	}
	for _, before := range approved {
		now := time.Now().UTC()
		result := tx.Model(&model.FanScenario{}).
			Where("id = ? AND scenario_status = ? AND version = ?", before.ID, string(constants.ScenarioStatusApproved), before.Version).
			Updates(map[string]interface{}{
				"scenario_status":       string(constants.ScenarioStatusPendingReview),
				"approved_by":           nil,
				"network_revision":      nil,
				"network_snapshot_hash": "",
				"invalidation_reason":   reason,
				"reject_reason":         "",
				"version":               gorm.Expr("version + 1"),
				"updated_at":            now,
			})
		if result.Error != nil {
			return 0, 0, fmt.Errorf("invalidate approved scenario %d: %w", before.ID, result.Error)
		}
		if result.RowsAffected != 1 {
			// 版本行锁串行期间批准方案不应被并发改动；出现差异必须整体回滚，杜绝漏失效。
			return 0, 0, fmt.Errorf("approved scenario %d changed during invalidation: %w", before.ID, ErrVersionConflict)
		}
		var after model.FanScenario
		if err := tx.First(&after, before.ID).Error; err != nil {
			return 0, 0, fmt.Errorf("reload invalidated scenario %d: %w", before.ID, err)
		}
		beforeJSON, _ := json.Marshal(before)
		afterJSON, _ := json.Marshal(after)
		event := auditTemplate
		event.Action = "fan_scenario.auto_invalidated"
		event.EntityType = "fan_scenario"
		event.EntityID = before.ID
		event.BeforeState = string(beforeJSON)
		event.AfterState = string(afterJSON)
		metadata, _ := json.Marshal(map[string]interface{}{"reason": reason, "previous_network_revision": before.NetworkRevision, "new_network_revision": newRevision})
		event.Metadata = string(metadata)
		if err := writeAudit(tx, event); err != nil {
			return 0, 0, err
		}
	}
	return newRevision, int64(len(approved)), nil
}

// BindApprovalSnapshot 在批准事务内把当前网络版本号与内容指纹绑定到方案。
// 条件更新保证状态机与乐观锁不被并发越级；锁内计算保证绑定的就是提交时的网络。
func (r *NetworkSnapshotRepository) BindApprovalSnapshot(ctx context.Context, id, actorID uint, expectedVersion uint, audit AuditRecord) (*model.FanScenario, uint64, error) {
	var updated model.FanScenario
	var boundRevision uint64
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		rev, err := lockNetworkRevision(tx, r.db.Dialector.Name())
		if err != nil {
			return err
		}
		var before model.FanScenario
		if err := tx.First(&before, id).Error; err != nil {
			return fmt.Errorf("load fan scenario before approval: %w", err)
		}
		if before.ScenarioStatus != string(constants.ScenarioStatusPendingReview) || before.Version != expectedVersion {
			return ErrVersionConflict
		}
		nodes, edges, err := activeNetworkInTx(tx)
		if err != nil {
			return err
		}
		fingerprint := ComputeNetworkFingerprint(nodes, edges)
		now := time.Now().UTC()
		result := tx.Model(&model.FanScenario{}).
			Where("id = ? AND version = ? AND scenario_status = ?", id, expectedVersion, string(constants.ScenarioStatusPendingReview)).
			Updates(map[string]interface{}{
				"scenario_status":       string(constants.ScenarioStatusApproved),
				"approved_by":           actorID,
				"network_revision":      rev.Revision,
				"network_snapshot_hash": fingerprint,
				"invalidation_reason":   "",
				"reject_reason":         "",
				"version":               gorm.Expr("version + 1"),
				"updated_at":            now,
			})
		if result.Error != nil {
			return fmt.Errorf("approve fan scenario: %w", result.Error)
		}
		if result.RowsAffected != 1 {
			return ErrVersionConflict
		}
		if err := tx.First(&updated, id).Error; err != nil {
			return err
		}
		boundRevision = rev.Revision
		beforeJSON, _ := json.Marshal(before)
		afterJSON, _ := json.Marshal(updated)
		audit.EntityID = id
		audit.BeforeState = string(beforeJSON)
		audit.AfterState = string(afterJSON)
		// 合并服务层给出的操作原因与本次绑定的网络快照凭据。
		meta := map[string]interface{}{}
		if strings.TrimSpace(audit.Metadata) != "" {
			_ = json.Unmarshal([]byte(audit.Metadata), &meta)
		}
		meta["network_revision"] = rev.Revision
		meta["network_snapshot_hash"] = fingerprint
		meta["node_count"] = len(nodes)
		meta["edge_count"] = len(edges)
		merged, _ := json.Marshal(meta)
		audit.Metadata = string(merged)
		return writeAudit(tx, audit)
	})
	if err != nil {
		return nil, 0, err
	}
	return &updated, boundRevision, nil
}

// VerifiedRunBuilder 由服务层提供：在已验证快照的事务内构造推演记录。
// 返回的 run 不应预置主键；metadata 写入启动审计。
type VerifiedRunBuilder func(scenario model.FanScenario, nodes []model.VentilationNode, edges []model.AirwayEdge) (run *model.SimulationRun, metadata string, err error)

// CreateVerifiedRun 在锁定网络版本的事务内复核方案绑定快照：
// 方案必须处于 approved、绑定版本号等于当前版本且内容指纹一致，否则回滚并返回
// ErrSnapshotStale。校验通过后调用 builder 求解并在同一事务落库，保证输入快照、
// 准入凭据与结果不可分割。
func (r *NetworkSnapshotRepository) CreateVerifiedRun(ctx context.Context, scenarioID uint, audit AuditRecord, build VerifiedRunBuilder) (*model.SimulationRun, error) {
	var saved model.SimulationRun
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		rev, err := lockNetworkRevision(tx, r.db.Dialector.Name())
		if err != nil {
			return err
		}
		var scenario model.FanScenario
		if err := tx.First(&scenario, scenarioID).Error; err != nil {
			return fmt.Errorf("load fan scenario for simulation: %w", err)
		}
		if scenario.ScenarioStatus != string(constants.ScenarioStatusApproved) {
			return ErrSnapshotStale
		}
		if scenario.NetworkRevision == nil || *scenario.NetworkRevision != rev.Revision || scenario.NetworkSnapshotHash == "" {
			return ErrSnapshotStale
		}
		nodes, edges, err := activeNetworkInTx(tx)
		if err != nil {
			return err
		}
		if ComputeNetworkFingerprint(nodes, edges) != scenario.NetworkSnapshotHash {
			return ErrSnapshotStale
		}
		run, metadata, err := build(scenario, nodes, edges)
		if err != nil {
			return err
		}
		if err := tx.Create(run).Error; err != nil {
			return fmt.Errorf("create simulation run: %w", err)
		}
		audit.EntityID = run.ID
		after, _ := json.Marshal(run)
		audit.AfterState = string(after)
		audit.Metadata = metadata
		if err := writeAudit(tx, audit); err != nil {
			return err
		}
		saved = *run
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &saved, nil
}

// Dialector 暴露底层驱动名（SQLite 与 PostgreSQL 锁策略不同）。
func (r *NetworkSnapshotRepository) Dialector() string { return r.db.Dialector.Name() }
