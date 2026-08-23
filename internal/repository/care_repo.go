package repository

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"
	"plant-diary/internal/model"
)

type CareRepository struct {
	db *gorm.DB
}

func NewCareRepository(db *gorm.DB) *CareRepository {
	return &CareRepository{db: db}
}

func (r *CareRepository) CreateLog(log *model.CareLog) error {
	return r.db.Create(log).Error
}

func (r *CareRepository) ListLogs(plantID string) ([]model.CareLog, error) {
	var logs []model.CareLog
	err := r.db.Where("plant_id = ?", plantID).Order("created_at DESC").Find(&logs).Error
	return logs, err
}

func (r *CareRepository) UpsertCycle(cycle *model.CareCycle) error {
	// Upsert one cycle per (plant_id, type). The unique index
	// idx_care_cycles_plant_type is the guard: even if two updates race, at most one
	// row survives, so a stale reminder can no longer linger beside the new one.
	// Common path: the row exists, update it in place. The insert handles the
	// first-ever set; if a concurrent writer slipped in between, the unique index
	// makes our insert fail and we fall back to an update on the now-existing row.
	var existing model.CareCycle
	err := r.db.Where("plant_id = ? AND type = ?", cycle.PlantID, cycle.Type).First(&existing).Error
	switch {
	case err == nil:
		cycle.ID = existing.ID
		cycle.CreatedAt = existing.CreatedAt
		return r.db.Model(&existing).Updates(map[string]any{
			"interval_days": cycle.IntervalDays,
			"last_date":      cycle.LastDate,
			"next_date":      cycle.NextDate,
		}).Error
	case errors.Is(err, gorm.ErrRecordNotFound):
		if err := r.db.Create(cycle).Error; err == nil || !isUniqueConstraintError(err) {
			return err
		}
		// lost the race to insert; reload and update instead
		var winner model.CareCycle
		if err := r.db.Where("plant_id = ? AND type = ?", cycle.PlantID, cycle.Type).First(&winner).Error; err != nil {
			return err
		}
		cycle.ID = winner.ID
		cycle.CreatedAt = winner.CreatedAt
		return r.db.Model(&winner).Updates(map[string]any{
			"interval_days": cycle.IntervalDays,
			"last_date":      cycle.LastDate,
			"next_date":      cycle.NextDate,
		}).Error
	default:
		return err
	}
}

// isUniqueConstraintError reports whether err is a UNIQUE-constraint violation,
// used to distinguish "row already exists, update it" from a real write failure.
func isUniqueConstraintError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "UNIQUE constraint failed") || strings.Contains(msg, "constraint failed: UNIQUE")
}

// DeduplicateCycles collapses any pre-existing duplicate (plant_id, type) rows to
// a single survivor, keeping the one with the latest next_date (the most recently
// scheduled reminder). It must run before AutoMigrate adds the unique index, which
// would otherwise fail — or leave the table untouched — when stale duplicates exist.
func (r *CareRepository) DeduplicateCycles() error {
	return r.db.Exec(`DELETE FROM care_cycles
WHERE id NOT IN (
	SELECT id FROM care_cycles c
	WHERE c.id = (
		SELECT c2.id FROM care_cycles c2
		WHERE c2.plant_id = c.plant_id AND c2.type = c.type
		ORDER BY c2.next_date DESC, c2.id DESC LIMIT 1
	)
)`).Error
}

// MigrateCycles runs the care_cycles migration, tolerating databases that already
// contain duplicate (plant_id, type) rows from the prior non-atomic upsert. The
// first AutoMigrate creates the table but fails to add the unique index while any
// duplicate exists; we deduplicate, then retry so the index lands cleanly.
func (r *CareRepository) MigrateCycles() error {
	if err := r.db.AutoMigrate(&model.CareCycle{}); err == nil {
		return nil
	} else if !isUniqueConstraintError(err) {
		return err
	}
	if err := r.DeduplicateCycles(); err != nil {
		return fmt.Errorf("deduplicate care cycles: %w", err)
	}
	return r.db.AutoMigrate(&model.CareCycle{})
}

func (r *CareRepository) DeleteCycle(plantID, cycleType string) error {
	return r.db.Where("plant_id = ? AND type = ?", plantID, cycleType).Delete(&model.CareCycle{}).Error
}

func (r *CareRepository) Transaction(fn func(*CareRepository) error) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		return fn(&CareRepository{db: tx})
	})
}

func (r *CareRepository) ListDueCycles(userID string, today time.Time) ([]model.CareCycle, error) {
	var cycles []model.CareCycle
	err := r.db.Model(&model.CareCycle{}).
		Joins("JOIN plants ON plants.id = care_cycles.plant_id").
		Where("plants.user_id = ? AND care_cycles.next_date <= ?", userID, today).
		Preload("Plant").Order("care_cycles.next_date ASC").Find(&cycles).Error
	return cycles, err
}

func (r *CareRepository) CountLogsAfter(userID string, from time.Time) (int64, error) {
	var count int64
	err := r.db.Model(&model.CareLog{}).
		Joins("JOIN plants ON plants.id = care_logs.plant_id").
		Where("plants.user_id = ? AND care_logs.created_at >= ?", userID, from).
		Count(&count).Error
	return count, err
}
