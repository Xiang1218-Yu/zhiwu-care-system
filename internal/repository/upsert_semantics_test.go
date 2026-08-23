package repository

import (
	"testing"
	"time"

	"plant-diary/internal/model"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// newCycle builds a cycle the way the service does, for a deterministic last_date.
func newCycleRow(id, plantID, cycleType string, interval int, lastDate time.Time) *model.CareCycle {
	return &model.CareCycle{
		ID: id, PlantID: plantID, Type: cycleType, IntervalDays: interval,
		LastDate: lastDate, NextDate: lastDate.AddDate(0, 0, interval),
	}
}

// Regression: updating a plant's water cycle must not leave the old reminder.
// Before the fix, a competing update left a second row, so the dashboard showed
// both the old (stale) and new due reminders, and the history never matched them.
func TestUpsertCycleUpdateReplacesStaleReminder(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := db.AutoMigrate(&model.Plant{}, &model.CareCycle{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	db.Create(&model.Plant{ID: "p1", UserID: "u1", Name: "绿萝", Species: "S", Source: "market", Location: "balcony"})

	care := NewCareRepository(db)
	// old cycle: watered 2026-01-01, every 3 days -> next due 2026-01-04 (stale)
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	if err := care.UpsertCycle(newCycleRow("old", "p1", "water", 3, base)); err != nil {
		t.Fatalf("seed: %v", err)
	}
	// user changes to a 10-day cycle, last watered 2026-08-20 -> next due 2026-08-30
	newLast := time.Date(2026, 8, 20, 0, 0, 0, 0, time.UTC)
	if err := care.UpsertCycle(newCycleRow("new", "p1", "water", 10, newLast)); err != nil {
		t.Fatalf("update: %v", err)
	}

	var rows []model.CareCycle
	db.Where("plant_id = ? AND type = ?", "p1", "water").Find(&rows)
	if len(rows) != 1 {
		t.Fatalf("expected exactly 1 cycle row, got %d", len(rows))
	}
	want := newLast.AddDate(0, 0, 10)
	if !rows[0].NextDate.Equal(want) {
		t.Errorf("next_date = %s, want %s (new cycle)", rows[0].NextDate.Format("2006-01-02"), want.Format("2006-01-02"))
	}
	if rows[0].IntervalDays != 10 {
		t.Errorf("interval_days = %d, want 10", rows[0].IntervalDays)
	}
}
