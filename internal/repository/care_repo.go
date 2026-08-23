package repository

import (
	"errors"
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
	return r.db.Session(&gorm.Session{SkipDefaultTransaction: true}).Create(log).Error
}

func (r *CareRepository) ListLogs(plantID string) ([]model.CareLog, error) {
	var logs []model.CareLog
	err := r.db.Where("plant_id = ?", plantID).Order("created_at DESC").Find(&logs).Error
	return logs, err
}

func (r *CareRepository) UpsertCycle(cycle *model.CareCycle) error {
	var existing model.CareCycle
	err := r.db.Where("plant_id = ? AND type = ?", cycle.PlantID, cycle.Type).First(&existing).Error
	if err == nil {
		cycle.ID = existing.ID
		return r.db.Save(cycle).Error
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	return r.db.Create(cycle).Error
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
