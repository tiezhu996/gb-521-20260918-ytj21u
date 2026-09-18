package model

import (
	"time"

	"gorm.io/datatypes"
)

type FanScenario struct {
	ID              uint           `gorm:"primaryKey" json:"id"`
	Name            string         `gorm:"size:120;not null" json:"name"`
	Description     string         `gorm:"size:600;not null" json:"description"`
	FanCurveJSON    datatypes.JSON `gorm:"type:jsonb;not null" json:"fan_curve_json"`
	OperatingMode   string         `gorm:"size:40;not null" json:"operating_mode"`
	ScenarioStatus  string         `gorm:"size:24;index;not null;check:scenario_status IN ('draft','pending_review','approved','archived')" json:"scenario_status"`
	SolverTolerance float64        `gorm:"not null;default:0.02" json:"solver_tolerance"`
	MaxIterations   int            `gorm:"not null;default:80" json:"max_iterations"`
	Version         uint           `gorm:"not null;default:1" json:"version"`
	CreatedBy       uint           `gorm:"not null;index" json:"created_by"`
	ApprovedBy      *uint          `gorm:"index" json:"approved_by"`
	RejectReason    string         `gorm:"size:400" json:"reject_reason"`
	// NetworkRevision 记录批准时绑定的通风网络版本号；为空表示从未绑定（非已批准方案）。
	NetworkRevision *uint64 `gorm:"index" json:"network_revision,omitempty"`
	// NetworkSnapshotHash 是批准时绑定的网络内容指纹，防止版本号之外的静默篡改。
	NetworkSnapshotHash string `gorm:"size:64" json:"network_snapshot_hash,omitempty"`
	// InvalidationReason 记录网络变化导致批准被系统撤销的原因，重新批准时清空。
	InvalidationReason string    `gorm:"size:400" json:"invalidation_reason,omitempty"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

func (FanScenario) TableName() string { return "fan_scenarios" }

// NetworkRevision 是全通风网络（节点与巷道）的单行版本计数器。
// 任何节点/巷道新增、停用或参数变化都必须在同一事务内推进该版本，
// 并锁定本行以串行化并发网络变更与推演准入。
type NetworkRevision struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Revision  uint64    `gorm:"not null;default:0" json:"revision"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (NetworkRevision) TableName() string { return "network_revisions" }
