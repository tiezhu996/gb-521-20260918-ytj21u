package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"mine-ventilation-network-simulator/backend/internal/model"
)

var (
	// ErrScenarioNotApproved 表示推演启动事务内复核时方案已不在已批准状态。
	ErrScenarioNotApproved = errors.New("scenario is not approved")
	// ErrApprovalStale 表示方案绑定的网络指纹与当前网络不一致。
	ErrApprovalStale = errors.New("approved network snapshot is stale")
)

const networkStateSingletonID = 1

type NetworkStateRepository struct{ db *gorm.DB }

func NewNetworkStateRepository(db *gorm.DB) *NetworkStateRepository {
	return &NetworkStateRepository{db: db}
}

func (r *NetworkStateRepository) Current(ctx context.Context) (*model.NetworkState, error) {
	var state model.NetworkState
	if err := r.db.WithContext(ctx).First(&state, networkStateSingletonID).Error; err != nil {
		return nil, fmt.Errorf("load network state: %w", err)
	}
	return &state, nil
}

// ensureNetworkStateTx 在事务内保证单行网络版本记录存在。
func ensureNetworkStateTx(tx *gorm.DB) error {
	state := model.NetworkState{ID: networkStateSingletonID, Revision: 1}
	if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&state).Error; err != nil {
		return fmt.Errorf("ensure network state: %w", err)
	}
	return nil
}

// lockNetworkState 串行化"批准绑定"与"网络变更"：PostgreSQL 使用行级
// SELECT ... FOR UPDATE，SQLite 用单行写操作把事务升级为写事务，
// 两种驱动下并发网络变更都无法让旧批准漏失效。
func lockNetworkState(tx *gorm.DB) error {
	if err := ensureNetworkStateTx(tx); err != nil {
		return err
	}
	if tx.Dialector.Name() == "sqlite" {
		if err := tx.Exec("UPDATE network_states SET revision = revision WHERE id = ?", networkStateSingletonID).Error; err != nil {
			return fmt.Errorf("lock network state: %w", err)
		}
		return nil
	}
	var state model.NetworkState
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&state, networkStateSingletonID).Error; err != nil {
		return fmt.Errorf("lock network state: %w", err)
	}
	return nil
}

// bumpNetworkRevision 在网络内容真实变化后提升版本，并对并发批准形成串行点。
func bumpNetworkRevision(tx *gorm.DB) error {
	if err := ensureNetworkStateTx(tx); err != nil {
		return err
	}
	result := tx.Exec("UPDATE network_states SET revision = revision + 1, updated_at = ? WHERE id = ?", time.Now().UTC(), networkStateSingletonID)
	if result.Error != nil {
		return fmt.Errorf("bump network revision: %w", result.Error)
	}
	if result.RowsAffected != 1 {
		return fmt.Errorf("network state singleton missing")
	}
	return nil
}

func loadNetworkSnapshot(tx *gorm.DB) ([]model.VentilationNode, []model.AirwayEdge, error) {
	var nodes []model.VentilationNode
	if err := tx.Order("id ASC").Find(&nodes).Error; err != nil {
		return nil, nil, fmt.Errorf("load nodes for network fingerprint: %w", err)
	}
	var edges []model.AirwayEdge
	if err := tx.Order("id ASC").Find(&edges).Error; err != nil {
		return nil, nil, fmt.Errorf("load edges for network fingerprint: %w", err)
	}
	return nodes, edges, nil
}

func currentNetworkFingerprint(tx *gorm.DB) (string, error) {
	nodes, edges, err := loadNetworkSnapshot(tx)
	if err != nil {
		return "", err
	}
	return model.ComputeNetworkFingerprint(nodes, edges), nil
}

// invalidateIfNetworkChanged 比较变更前后指纹：只有网络内容真实变化时才
// 提升版本并使全部已批准方案失效；无变化时批准保持有效，仍可推演。
func invalidateIfNetworkChanged(tx *gorm.DB, beforeFingerprint string, trigger AuditRecord, reason string) error {
	after, err := currentNetworkFingerprint(tx)
	if err != nil {
		return err
	}
	if after == beforeFingerprint {
		return nil
	}
	if err := bumpNetworkRevision(tx); err != nil {
		return err
	}
	return invalidateApprovedScenarios(tx, trigger, reason)
}

// invalidateApprovedScenarios 把全部已批准方案条件更新回待复核并写入审计；
// 条件更新保证并发变更重复执行时不会覆盖彼此或漏掉任何一个批准。
func invalidateApprovedScenarios(tx *gorm.DB, trigger AuditRecord, reason string) error {
	var approved []model.FanScenario
	if err := tx.Where("scenario_status = ?", "approved").Find(&approved).Error; err != nil {
		return fmt.Errorf("list approved scenarios for invalidation: %w", err)
	}
	for _, scenario := range approved {
		result := tx.Model(&model.FanScenario{}).
			Where("id = ? AND scenario_status = ?", scenario.ID, "approved").
			Updates(map[string]interface{}{
				"scenario_status":     "pending_review",
				"invalidation_reason": reason,
				"version":             gorm.Expr("version + 1"),
			})
		if result.Error != nil {
			return fmt.Errorf("invalidate scenario %d approval: %w", scenario.ID, result.Error)
		}
		if result.RowsAffected != 1 {
			continue // 并发事务已处理该方案
		}
		var after model.FanScenario
		if err := tx.First(&after, scenario.ID).Error; err != nil {
			return fmt.Errorf("reload invalidated scenario: %w", err)
		}
		beforeJSON, _ := json.Marshal(scenario)
		afterJSON, _ := json.Marshal(after)
		audit := AuditRecord{
			RequestID: trigger.RequestID, ActorID: trigger.ActorID, ActorEmail: trigger.ActorEmail,
			Action: "fan_scenario.approval_invalidated", EntityType: "fan_scenario", EntityID: scenario.ID,
			BeforeState: string(beforeJSON), AfterState: string(afterJSON),
			Metadata: fmt.Sprintf(`{"reason":%q,"trigger_action":%q,"trigger_entity_id":%d}`, reason, trigger.Action, trigger.EntityID),
		}
		if err := writeAudit(tx, audit); err != nil {
			return err
		}
	}
	return nil
}
