package model

import (
	"time"

	"gorm.io/datatypes"
)

type SimulationRun struct {
	ID                uint           `gorm:"primaryKey" json:"id"`
	ScenarioID        uint           `gorm:"not null;index" json:"scenario_id"`
	RunStatus         string         `gorm:"size:24;not null;index;check:run_status IN ('queued','running','converged','not_converged','invalid_input','failed')" json:"run_status"`
	IterationCount    int            `gorm:"not null;default:0" json:"iteration_count"`
	Residual          float64        `gorm:"not null;default:0" json:"residual"`
	InputSnapshotJSON datatypes.JSON `gorm:"type:jsonb;not null" json:"input_snapshot_json"`
	NodePressuresJSON datatypes.JSON `gorm:"type:jsonb;not null" json:"node_pressures_json"`
	EdgeFlowsJSON     datatypes.JSON `gorm:"type:jsonb;not null" json:"edge_flows_json"`
	ResidualsJSON     datatypes.JSON `gorm:"type:jsonb;not null" json:"residuals_json"`
	RiskFlagsJSON     datatypes.JSON `gorm:"type:jsonb;not null" json:"risk_flags_json"`
	AlgorithmVersion  string         `gorm:"size:32;not null" json:"algorithm_version"`
	StartedBy         uint           `gorm:"not null;index" json:"started_by"`
	StartedAt         time.Time      `json:"started_at"`
	FinishedAt        *time.Time     `json:"finished_at"`
	RiskConfirmedBy   *uint          `gorm:"index" json:"risk_confirmed_by"`
	RiskConfirmedAt   *time.Time     `json:"risk_confirmed_at"`
	ConfirmationNote  string         `gorm:"size:500" json:"confirmation_note"`
	Scenario          FanScenario    `gorm:"foreignKey:ScenarioID" json:"scenario,omitempty"`
	CreatedAt         time.Time      `json:"created_at"`
	UpdatedAt         time.Time      `json:"updated_at"`
}

func (SimulationRun) TableName() string { return "simulation_runs" }
