package repository

import (
	"errors"
	"testing"

	"plant-diary/internal/model"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestCareRepositoryTransactionRollsBackLog(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.Plant{}, &model.CareLog{}, &model.CareCycle{}); err != nil {
		t.Fatal(err)
	}
	repo := NewCareRepository(db)

	err = repo.Transaction(func(tx *CareRepository) error {
		if err := tx.CreateLog(&model.CareLog{
			ID: "log-1", PlantID: "plant-1", Type: model.CareWater,
		}); err != nil {
			return err
		}
		return errors.New("force rollback")
	})
	if err == nil {
		t.Fatal("expected transaction error")
	}

	var count int64
	if err := db.Model(&model.CareLog{}).Where("id = ?", "log-1").Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("expected rollback to remove log, got %d rows", count)
	}
}
