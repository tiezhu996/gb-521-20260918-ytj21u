package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"gorm.io/gorm"

	"mine-ventilation-network-simulator/backend/internal/model"
)

type SimulationRunRepository struct{ db *gorm.DB }

func NewSimulationRunRepository(db *gorm.DB) *SimulationRunRepository {
	return &SimulationRunRepository{db: db}
}

func (r *SimulationRunRepository) List(ctx context.Context, page, pageSize int, status string, scenarioID uint) ([]model.SimulationRun, int64, error) {
	query := r.db.WithContext(ctx).Model(&model.SimulationRun{})
	if status != "" {
		query = query.Where("run_status = ?", status)
	}
	if scenarioID > 0 {
		query = query.Where("scenario_id = ?", scenarioID)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count simulation runs: %w", err)
	}
	var runs []model.SimulationRun
	err := query.Preload("Scenario").Order("started_at DESC, id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&runs).Error
	if err != nil {
		return nil, 0, fmt.Errorf("list simulation runs: %w", err)
	}
	return runs, total, nil
}

func (r *SimulationRunRepository) Find(ctx context.Context, id uint) (*model.SimulationRun, error) {
	var run model.SimulationRun
	if err := r.db.WithContext(ctx).Preload("Scenario").First(&run, id).Error; err != nil {
		return nil, fmt.Errorf("find simulation run: %w", err)
	}
	return &run, nil
}

func (r *SimulationRunRepository) Create(ctx context.Context, run *model.SimulationRun, audit AuditRecord) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(run).Error; err != nil {
			return fmt.Errorf("create simulation run: %w", err)
		}
		after, _ := json.Marshal(run)
		audit.EntityID = run.ID
		audit.AfterState = string(after)
		return writeAudit(tx, audit)
	})
}

func (r *SimulationRunRepository) ConfirmRisks(ctx context.Context, id, actorID uint, note string, audit AuditRecord) (*model.SimulationRun, error) {
	var updated model.SimulationRun
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var before model.SimulationRun
		if err := tx.First(&before, id).Error; err != nil {
			return fmt.Errorf("load simulation before confirmation: %w", err)
		}
		if before.RiskConfirmedAt != nil {
			return gorm.ErrDuplicatedKey
		}
		now := time.Now().UTC()
		result := tx.Model(&model.SimulationRun{}).
			Where("id = ? AND risk_confirmed_at IS NULL", id).
			Updates(map[string]interface{}{"risk_confirmed_by": actorID, "risk_confirmed_at": now, "confirmation_note": note})
		if result.Error != nil {
			return fmt.Errorf("confirm simulation risks: %w", result.Error)
		}
		if result.RowsAffected != 1 {
			return ErrVersionConflict
		}
		if err := tx.First(&updated, id).Error; err != nil {
			return err
		}
		beforeJSON, _ := json.Marshal(before)
		afterJSON, _ := json.Marshal(updated)
		audit.EntityID = id
		audit.BeforeState = string(beforeJSON)
		audit.AfterState = string(afterJSON)
		return writeAudit(tx, audit)
	})
	if err != nil {
		return nil, err
	}
	return &updated, nil
}
