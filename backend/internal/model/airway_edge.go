package model

import "time"

type AirwayEdge struct {
	ID              uint            `gorm:"primaryKey" json:"id"`
	Code            string          `gorm:"size:48;uniqueIndex;not null" json:"code"`
	FromNodeID      uint            `gorm:"not null;index;uniqueIndex:idx_airway_direction" json:"from_node_id"`
	ToNodeID        uint            `gorm:"not null;index;uniqueIndex:idx_airway_direction" json:"to_node_id"`
	ResistanceNS2M8 float64         `gorm:"not null" json:"resistance_ns2m8"`
	AreaM2          float64         `gorm:"not null" json:"area_m2"`
	MaxVelocityMS   float64         `gorm:"not null" json:"max_velocity_ms"`
	DoorState       string          `gorm:"size:20;not null" json:"door_state"`
	Enabled         bool            `gorm:"not null;default:true;index" json:"enabled"`
	CriticalPath    bool            `gorm:"not null;default:false" json:"critical_path"`
	Version         uint            `gorm:"not null;default:1" json:"version"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
	FromNode        VentilationNode `gorm:"foreignKey:FromNodeID" json:"from_node,omitempty"`
	ToNode          VentilationNode `gorm:"foreignKey:ToNodeID" json:"to_node,omitempty"`
}

func (AirwayEdge) TableName() string { return "airway_edges" }
