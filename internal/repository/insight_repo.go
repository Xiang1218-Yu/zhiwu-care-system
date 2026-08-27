package repository

import (
	"errors"
	"time"

	"plant-diary/internal/model"

	"gorm.io/gorm"
)

type CareRecord struct {
	ID          string    `json:"id"`
	PlantID     string    `json:"plant_id"`
	PlantName   string    `json:"plant_name"`
	PlantStatus string    `json:"plant_status"`
	Type        string    `json:"type"`
	Note        string    `json:"note"`
	PhotoURL    string    `json:"photo_url"`
	CreatedAt   time.Time `json:"created_at"`
}

func (r CareRecord) TypeLabel() string {
	switch r.Type {
	case model.CareWater:
		return "浇水"
	case model.CareFertilizer:
		return "施肥"
	case model.CareRepot:
		return "换盆"
	case model.CarePrune:
		return "修剪"
	case model.CareSpray:
		return "喷药"
	case model.CareClean:
		return "擦拭叶片"
	default:
		return r.Type
	}
}

type InsightRepository struct {
	db *gorm.DB
}

func NewInsightRepository(db *gorm.DB) *InsightRepository {
	return &InsightRepository{db: db}
}

func (r *InsightRepository) ListCareRecords(userID string, from, to time.Time, plantID, careType string) ([]CareRecord, error) {
	query := r.db.Table("care_logs").
		Select("care_logs.id, care_logs.plant_id, plants.name AS plant_name, plants.status AS plant_status, care_logs.type, care_logs.note, care_logs.photo_url, care_logs.created_at").
		Joins("JOIN plants ON plants.id = care_logs.plant_id").
		Where("care_logs.created_at >= ? AND care_logs.created_at < ?", from, to.AddDate(0, 0, 1)).
		Order("care_logs.created_at ASC")
	if plantID != "" {
		query = query.Where("care_logs.plant_id = ?", plantID)
	}
	if careType != "" {
		query = query.Where("care_logs.type = ?", careType)
	}
	var records []CareRecord
	return records, query.Scan(&records).Error
}

func (r *InsightRepository) ListCycles(userID string) ([]model.CareCycle, error) {
	var cycles []model.CareCycle
	err := r.db.Model(&model.CareCycle{}).
		Joins("JOIN plants ON plants.id = care_cycles.plant_id").
		Preload("Plant").
		Order("care_cycles.next_date ASC").
		Find(&cycles).Error
	return cycles, err
}

func (r *InsightRepository) FindPlant(userID, plantID string) (*model.Plant, error) {
	var plant model.Plant
	err := r.db.Where("id = ?", plantID).First(&plant).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &plant, err
}
