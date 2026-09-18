package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"gorm.io/gorm"

	"mine-ventilation-network-simulator/backend/internal/model"
)

type VentilationNodeRepository struct{ db *gorm.DB }

func NewVentilationNodeRepository(db *gorm.DB) *VentilationNodeRepository {
	return &VentilationNodeRepository{db: db}
}

func (r *VentilationNodeRepository) List(ctx context.Context, page, pageSize int, nodeType, status, search string) ([]model.VentilationNode, int64, error) {
	query := r.db.WithContext(ctx).Model(&model.VentilationNode{})
	if nodeType != "" {
		query = query.Where("node_type = ?", nodeType)
	}
	if status != "" {
		query = query.Where("status = ?", status)
	}
	if search != "" {
		query = query.Where("LOWER(code) LIKE ?", "%"+strings.ToLower(search)+"%")
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count ventilation nodes: %w", err)
	}
	var nodes []model.VentilationNode
	if err := query.Order("code ASC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&nodes).Error; err != nil {
		return nil, 0, fmt.Errorf("list ventilation nodes: %w", err)
	}
	return nodes, total, nil
}

func (r *VentilationNodeRepository) AllActive(ctx context.Context) ([]model.VentilationNode, error) {
	var nodes []model.VentilationNode
	if err := r.db.WithContext(ctx).Where("status = ?", "active").Order("id ASC").Find(&nodes).Error; err != nil {
		return nil, fmt.Errorf("list active ventilation nodes: %w", err)
	}
	return nodes, nil
}

func (r *VentilationNodeRepository) Find(ctx context.Context, id uint) (*model.VentilationNode, error) {
	var node model.VentilationNode
	if err := r.db.WithContext(ctx).First(&node, id).Error; err != nil {
		return nil, fmt.Errorf("find ventilation node: %w", err)
	}
	return &node, nil
}

func (r *VentilationNodeRepository) Create(ctx context.Context, node *model.VentilationNode, audit AuditRecord) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(node).Error; err != nil {
			return fmt.Errorf("create ventilation node: %w", err)
		}
		audit.EntityID = node.ID
		after, _ := json.Marshal(node)
		audit.AfterState = string(after)
		return writeAudit(tx, audit)
	})
}

func (r *VentilationNodeRepository) Update(ctx context.Context, node *model.VentilationNode, audit AuditRecord) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var before model.VentilationNode
		if err := tx.First(&before, node.ID).Error; err != nil {
			return fmt.Errorf("load ventilation node before update: %w", err)
		}
		if err := tx.Model(&before).Select("node_type", "elevation_m", "required_airflow_m3_s", "pressure_pa", "status").Updates(node).Error; err != nil {
			return fmt.Errorf("update ventilation node: %w", err)
		}
		if err := tx.First(node, node.ID).Error; err != nil {
			return fmt.Errorf("reload ventilation node: %w", err)
		}
		beforeJSON, _ := json.Marshal(before)
		afterJSON, _ := json.Marshal(node)
		audit.EntityID = node.ID
		audit.BeforeState = string(beforeJSON)
		audit.AfterState = string(afterJSON)
		return writeAudit(tx, audit)
	})
}
