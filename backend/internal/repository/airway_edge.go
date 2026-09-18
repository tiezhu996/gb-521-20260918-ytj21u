package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"gorm.io/gorm"

	"mine-ventilation-network-simulator/backend/internal/constants"
	"mine-ventilation-network-simulator/backend/internal/model"
)

type AirwayEdgeRepository struct {
	db      *gorm.DB
	dialect string
}

func NewAirwayEdgeRepository(db *gorm.DB) *AirwayEdgeRepository {
	return &AirwayEdgeRepository{db: db, dialect: db.Dialector.Name()}
}

// edgeAffectsSnapshot 判断巷道写入是否改变推演消费的网络快照字段。
func edgeAffectsSnapshot(before, after model.AirwayEdge) bool {
	return before.ResistanceNS2M8 != after.ResistanceNS2M8 ||
		before.AreaM2 != after.AreaM2 ||
		before.MaxVelocityMS != after.MaxVelocityMS ||
		before.DoorState != after.DoorState ||
		before.Enabled != after.Enabled ||
		before.CriticalPath != after.CriticalPath
}

func edgeInvalidationReason(edge *model.AirwayEdge) string {
	if !edge.Enabled {
		return fmt.Sprintf("巷道 %s 已停用，旧批准快照失效，须重新复核", edge.Code)
	}
	if edge.DoorState == string(constants.DoorStateClosed) {
		return fmt.Sprintf("巷道 %s 风门已关闭，旧批准快照失效，须重新复核", edge.Code)
	}
	return fmt.Sprintf("巷道 %s 参数已变化，旧批准快照失效，须重新复核", edge.Code)
}

func (r *AirwayEdgeRepository) List(ctx context.Context, page, pageSize int, enabled, search string) ([]model.AirwayEdge, int64, error) {
	query := r.db.WithContext(ctx).Model(&model.AirwayEdge{})
	if enabled == "true" || enabled == "false" {
		query = query.Where("enabled = ?", enabled == "true")
	}
	if search != "" {
		query = query.Where("LOWER(code) LIKE ?", "%"+strings.ToLower(search)+"%")
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count airway edges: %w", err)
	}
	var edges []model.AirwayEdge
	err := query.Preload("FromNode").Preload("ToNode").Order("code ASC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&edges).Error
	if err != nil {
		return nil, 0, fmt.Errorf("list airway edges: %w", err)
	}
	return edges, total, nil
}

func (r *AirwayEdgeRepository) AllEnabled(ctx context.Context) ([]model.AirwayEdge, error) {
	var edges []model.AirwayEdge
	err := r.db.WithContext(ctx).Where("enabled = ?", true).Preload("FromNode").Preload("ToNode").Order("id ASC").Find(&edges).Error
	if err != nil {
		return nil, fmt.Errorf("list enabled airway edges: %w", err)
	}
	return edges, nil
}

func (r *AirwayEdgeRepository) Find(ctx context.Context, id uint) (*model.AirwayEdge, error) {
	var edge model.AirwayEdge
	if err := r.db.WithContext(ctx).Preload("FromNode").Preload("ToNode").First(&edge, id).Error; err != nil {
		return nil, fmt.Errorf("find airway edge: %w", err)
	}
	return &edge, nil
}

func (r *AirwayEdgeRepository) CountByNode(ctx context.Context, nodeID uint) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.AirwayEdge{}).Where("from_node_id = ? OR to_node_id = ?", nodeID, nodeID).Count(&count).Error
	if err != nil {
		return 0, fmt.Errorf("count edges for node: %w", err)
	}
	return count, nil
}

func (r *AirwayEdgeRepository) Create(ctx context.Context, edge *model.AirwayEdge, audit AuditRecord) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(edge).Error; err != nil {
			return fmt.Errorf("create airway edge: %w", err)
		}
		after, _ := json.Marshal(edge)
		audit.EntityID = edge.ID
		audit.AfterState = string(after)
		if err := writeAudit(tx, audit); err != nil {
			return err
		}
		reason := fmt.Sprintf("新增巷道 %s，旧批准快照失效，须重新复核", edge.Code)
		if _, _, err := BumpRevisionAndInvalidate(tx, r.dialect, reason, audit); err != nil {
			return err
		}
		return nil
	})
}

func (r *AirwayEdgeRepository) Update(ctx context.Context, edge *model.AirwayEdge, expectedVersion uint, audit AuditRecord) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var before model.AirwayEdge
		if err := tx.First(&before, edge.ID).Error; err != nil {
			return fmt.Errorf("load airway edge before update: %w", err)
		}
		if before.Version != expectedVersion {
			return ErrVersionConflict
		}
		if !edgeAffectsSnapshot(before, *edge) {
			// 无实质网络参数变化：跳过版本推进与乐观锁，旧批准继续有效。
			return nil
		}
		result := tx.Model(&model.AirwayEdge{}).Where("id = ? AND version = ?", edge.ID, expectedVersion).Updates(map[string]interface{}{
			"resistance_ns2_m8": edge.ResistanceNS2M8, "area_m2": edge.AreaM2,
			"max_velocity_ms": edge.MaxVelocityMS, "door_state": edge.DoorState,
			"enabled": edge.Enabled, "critical_path": edge.CriticalPath,
			"version": gorm.Expr("version + 1"),
		})
		if result.Error != nil {
			return fmt.Errorf("update airway edge: %w", result.Error)
		}
		if result.RowsAffected != 1 {
			return ErrVersionConflict
		}
		if err := tx.First(edge, edge.ID).Error; err != nil {
			return err
		}
		beforeJSON, _ := json.Marshal(before)
		afterJSON, _ := json.Marshal(edge)
		audit.EntityID = edge.ID
		audit.BeforeState = string(beforeJSON)
		audit.AfterState = string(afterJSON)
		if err := writeAudit(tx, audit); err != nil {
			return err
		}
		if _, _, err := BumpRevisionAndInvalidate(tx, r.dialect, edgeInvalidationReason(edge), audit); err != nil {
			return err
		}
		return nil
	})
}
