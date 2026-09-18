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
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
}

func (FanScenario) TableName() string { return "fan_scenarios" }
