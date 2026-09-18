package model

import "time"

type VentilationNode struct {
	ID                 uint      `gorm:"primaryKey" json:"id"`
	Code               string    `gorm:"size:48;uniqueIndex;not null" json:"code"`
	NodeType           string    `gorm:"size:24;index;not null" json:"node_type"`
	ElevationM         float64   `gorm:"not null" json:"elevation_m"`
	RequiredAirflowM3S float64   `gorm:"not null;default:0" json:"required_airflow_m3s"`
	PressurePa         float64   `gorm:"not null;default:0" json:"pressure_pa"`
	Status             string    `gorm:"size:20;index;not null" json:"status"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

func (VentilationNode) TableName() string { return "ventilation_nodes" }
