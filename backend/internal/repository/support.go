package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"

	"mine-ventilation-network-simulator/backend/internal/model"
)

var ErrVersionConflict = errors.New("version conflict")

type AuditRecord struct {
	RequestID   string
	ActorID     uint
	ActorEmail  string
	Action      string
	EntityType  string
	EntityID    uint
	BeforeState string
	AfterState  string
	Metadata    string
}

type SupportRepository struct{ db *gorm.DB }

func NewSupportRepository(db *gorm.DB) *SupportRepository { return &SupportRepository{db: db} }

func (r *SupportRepository) Ready(ctx context.Context) error {
	sqlDB, err := r.db.DB()
	if err != nil {
		return fmt.Errorf("access database connection: %w", err)
	}
	if err := sqlDB.PingContext(ctx); err != nil {
		return fmt.Errorf("ping database: %w", err)
	}
	return nil
}

func (r *SupportRepository) FindUserByEmail(ctx context.Context, email string) (*model.User, error) {
	var user model.User
	if err := r.db.WithContext(ctx).Where("LOWER(email) = ?", strings.ToLower(strings.TrimSpace(email))).First(&user).Error; err != nil {
		return nil, fmt.Errorf("find user by email: %w", err)
	}
	return &user, nil
}

func (r *SupportRepository) FindUserByID(ctx context.Context, id uint) (*model.User, error) {
	var user model.User
	if err := r.db.WithContext(ctx).First(&user, id).Error; err != nil {
		return nil, fmt.Errorf("find user by id: %w", err)
	}
	return &user, nil
}

func (r *SupportRepository) WriteAudit(ctx context.Context, record AuditRecord) error {
	return writeAudit(r.db.WithContext(ctx), record)
}

func (r *SupportRepository) ListAudits(ctx context.Context, page, pageSize int, entityType, action, actor, status string, from, to *time.Time) ([]model.AuditEvent, int64, error) {
	query := r.db.WithContext(ctx).Model(&model.AuditEvent{})
	if entityType != "" {
		query = query.Where("entity_type = ?", entityType)
	}
	if action != "" {
		query = query.Where("action LIKE ?", "%"+action+"%")
	}
	if actor != "" {
		query = query.Where("LOWER(actor_email) LIKE ?", "%"+strings.ToLower(actor)+"%")
	}
	if status != "" {
		query = query.Where("after_state LIKE ?", "%\"scenario_status\":\""+status+"\"%")
	}
	if from != nil {
		query = query.Where("created_at >= ?", from.UTC())
	}
	if to != nil {
		query = query.Where("created_at <= ?", to.UTC())
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count audit events: %w", err)
	}
	var events []model.AuditEvent
	err := query.Order("created_at DESC, id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&events).Error
	if err != nil {
		return nil, 0, fmt.Errorf("list audit events: %w", err)
	}
	return events, total, nil
}

func writeAudit(tx *gorm.DB, record AuditRecord) error {
	event := model.AuditEvent{
		RequestID: record.RequestID, ActorID: record.ActorID, ActorEmail: record.ActorEmail,
		Action: record.Action, EntityType: record.EntityType, EntityID: record.EntityID,
		BeforeState: sanitizeAuditJSON(record.BeforeState), AfterState: sanitizeAuditJSON(record.AfterState), Metadata: sanitizeAuditJSON(record.Metadata),
		CreatedAt: time.Now().UTC(),
	}
	if event.RequestID == "" {
		event.RequestID = "untracked"
	}
	if event.BeforeState == "" {
		event.BeforeState = "{}"
	}
	if event.AfterState == "" {
		event.AfterState = "{}"
	}
	if event.Metadata == "" {
		event.Metadata = "{}"
	}
	if err := tx.Create(&event).Error; err != nil {
		return fmt.Errorf("write audit event: %w", err)
	}
	return nil
}

func sanitizeAuditJSON(raw string) string {
	if strings.TrimSpace(raw) == "" {
		return "{}"
	}
	var value interface{}
	if err := json.Unmarshal([]byte(raw), &value); err != nil {
		return `{"redacted":"invalid_json"}`
	}
	redactAuditValue(value)
	sanitized, err := json.Marshal(value)
	if err != nil {
		return `{"redacted":"serialization_failed"}`
	}
	return string(sanitized)
}

func redactAuditValue(value interface{}) {
	switch item := value.(type) {
	case map[string]interface{}:
		for key, nested := range item {
			normalized := strings.NewReplacer("_", "", "-", "", ".", "").Replace(strings.ToLower(key))
			if strings.Contains(normalized, "password") || strings.Contains(normalized, "passwd") || strings.Contains(normalized, "secret") || strings.Contains(normalized, "token") || strings.Contains(normalized, "authorization") || strings.Contains(normalized, "credential") || strings.Contains(normalized, "privatekey") {
				item[key] = "[REDACTED]"
				continue
			}
			redactAuditValue(nested)
		}
	case []interface{}:
		for _, nested := range item {
			redactAuditValue(nested)
		}
	}
}
