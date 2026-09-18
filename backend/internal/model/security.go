package model

import "time"

type User struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	Email        string    `gorm:"size:160;uniqueIndex;not null" json:"email"`
	DisplayName  string    `gorm:"size:80;not null" json:"display_name"`
	PasswordHash string    `gorm:"size:255;not null" json:"-"`
	Role         string    `gorm:"size:24;index;not null" json:"role"`
	Active       bool      `gorm:"not null;default:true" json:"active"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type AuditEvent struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	RequestID   string    `gorm:"size:64;index;not null" json:"request_id"`
	ActorID     uint      `gorm:"index;not null" json:"actor_id"`
	ActorEmail  string    `gorm:"size:160;not null" json:"actor_email"`
	Action      string    `gorm:"size:80;index;not null" json:"action"`
	EntityType  string    `gorm:"size:64;index;not null" json:"entity_type"`
	EntityID    uint      `gorm:"index;not null" json:"entity_id"`
	BeforeState string    `gorm:"type:text;not null" json:"before_state"`
	AfterState  string    `gorm:"type:text;not null" json:"after_state"`
	Metadata    string    `gorm:"type:text;not null" json:"metadata"`
	CreatedAt   time.Time `gorm:"index" json:"created_at"`
}

func (User) TableName() string       { return "users" }
func (AuditEvent) TableName() string { return "audit_events" }
