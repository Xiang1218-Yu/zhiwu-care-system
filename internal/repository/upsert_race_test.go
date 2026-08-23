package repository

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"plant-diary/internal/model"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestUpsertCycleConcurrentDuplicate(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	// shared in-memory db so all pooled connections see the same data
	sqlDB, _ := db.DB()
	sqlDB.SetMaxOpenConns(4)
	defer sqlDB.Close()
	if err := db.AutoMigrate(&model.Plant{}, &model.CareCycle{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	db.Create(&model.Plant{ID: "p1", UserID: "u1", Name: "T", Species: "S", Source: "market", Location: "balcony"})

	care := NewCareRepository(db)
	if err := care.UpsertCycle(&model.CareCycle{ID: "c0", PlantID: "p1", Type: "water", IntervalDays: 3, LastDate: time.Now(), NextDate: time.Now().AddDate(0, 0, 3)}); err != nil {
		t.Fatalf("seed: %v", err)
	}

	const n = 10
	var wg sync.WaitGroup
	for i := 1; i <= n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_ = care.UpsertCycle(&model.CareCycle{
				ID: fmt.Sprintf("c%d", i), PlantID: "p1", Type: "water",
				IntervalDays: i + 10, LastDate: time.Now(), NextDate: time.Now().AddDate(0, 0, i),
			})
		}(i)
	}
	wg.Wait()

	var cycles []model.CareCycle
	db.Where("plant_id = ? AND type = ?", "p1", "water").Find(&cycles)
	for _, c := range cycles {
		t.Logf("  row id=%s interval=%d next=%s", c.ID, c.IntervalDays, c.NextDate.Format("2006-01-02"))
	}
	if len(cycles) != 1 {
		t.Errorf("expected 1 cycle row, got %d (duplicate cycles -> duplicate reminders)", len(cycles))
	}
}
