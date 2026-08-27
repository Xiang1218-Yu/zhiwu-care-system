package repository

import (
	"errors"

	"gorm.io/gorm"
	"plant-diary/internal/model"
)

type PlantRepository struct {
	db *gorm.DB
}

func NewPlantRepository(db *gorm.DB) *PlantRepository {
	return &PlantRepository{db: db}
}

func (r *PlantRepository) ListByUser(userID, status, location string) ([]model.Plant, error) {
	query := r.db.Where("user_id = ?", userID).Preload("CareLogs", func(db *gorm.DB) *gorm.DB {
		return db.Order("created_at DESC")
	})
	if status != "" {
		query = query.Where("status = ?", status)
	}
	if location != "" {
		query = query.Where("location = ?", location)
	}
	var plants []model.Plant
	return plants, query.Order("created_at DESC").Find(&plants).Error
}

func (r *PlantRepository) FindOwned(id, userID string) (*model.Plant, error) {
	var plant model.Plant
	err := r.db.Where("id = ? AND user_id = ?", id, userID).
		Preload("CareLogs", func(db *gorm.DB) *gorm.DB { return db.Order("created_at DESC") }).
		Preload("CareCycles").
		First(&plant).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &plant, err
}

func (r *PlantRepository) Create(plant *model.Plant) error {
	return r.db.Create(plant).Error
}

func (r *PlantRepository) Save(plant *model.Plant) error {
	return r.db.Select("*").Save(plant).Error
}

func (r *PlantRepository) Delete(id, userID string) error {
	return r.db.Where("id = ? AND user_id = ?", id, userID).Delete(&model.Plant{}).Error
}

func (r *PlantRepository) CountByStatus(userID string) (map[string]int64, error) {
	type result struct {
		Status string
		Count  int64
	}
	var rows []result
	err := r.db.Model(&model.Plant{}).Select("status, count(*) as count").
		Where("user_id = ?", userID).Group("status").Scan(&rows).Error
	counts := make(map[string]int64, len(rows))
	for _, row := range rows {
		counts[row.Status] = row.Count
	}
	return counts, err
}

func (r *PlantRepository) CountCreatedAfter(userID string, from interface{}) (int64, error) {
	var count int64
	err := r.db.Model(&model.Plant{}).Where("user_id = ? AND created_at >= ?", userID, from).Count(&count).Error
	return count, err
}
