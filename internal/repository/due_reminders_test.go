package repository

import (
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"plant-diary/internal/model"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// End-to-end of the dashboard reminder query: after the migration repairs a DB
// that had a stale duplicate, and after concurrent cycle updates, ListDueCycles
// returns exactly one reminder per (plant, type) — never the old one beside the new.
func TestListDueCyclesNoDuplicatesAfterMigration(t *testing.T) {
	dbPath := "/tmp/pd_due_test.db"
	removeFile(dbPath)
	sqlDB, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	// old schema with non-unique index + a duplicate (the bug's fingerprint)
	if err := sqlDB.Exec(`CREATE TABLE care_cycles (
		id TEXT PRIMARY KEY, plant_id TEXT NOT NULL, type TEXT NOT NULL,
		interval_days INTEGER NOT NULL, last_date DATE NOT NULL, next_date DATE NOT NULL,
		created_at DATETIME, updated_at DATETIME)`).Error; err != nil {
		t.Fatalf("create table: %v", err)
	}
	if err := sqlDB.AutoMigrate(&model.User{}, &model.Plant{}); err != nil {
		t.Fatalf("migrate base: %v", err)
	}
	sqlDB.Create(&model.Plant{ID: "p1", UserID: "u1", Name: "绿萝", Species: "S", Source: "market", Location: "balcony"})
	sqlDB.Exec(`INSERT INTO care_cycles (id,plant_id,type,interval_days,last_date,next_date,created_at,updated_at) VALUES ('a','p1','water',3,'2026-01-01','2026-01-04','2026-01-01','2026-01-01')`)
	sqlDB.Exec(`INSERT INTO care_cycles (id,plant_id,type,interval_days,last_date,next_date,created_at,updated_at) VALUES ('b','p1','water',5,'2026-01-01','2026-01-06','2026-01-01','2026-01-01')`)

	care := NewCareRepository(sqlDB)
	if err := care.MigrateCycles(); err != nil {
		t.Fatalf("MigrateCycles: %v", err)
	}

	// before repair, dashboard would have shown 2 water reminders for the same plant
	due, err := care.ListDueCycles("u1", time.Now())
	if err != nil {
		t.Fatalf("ListDueCycles pre: %v", err)
	}
	if len(due) != 1 {
		t.Errorf("after migration expected 1 due reminder, got %d", len(due))
	}

	// concurrent cycle updates still leave exactly one row
	var wg sync.WaitGroup
	for i := 1; i <= 8; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			last := time.Now().AddDate(0, 0, -i)
			_ = care.UpsertCycle(&model.CareCycle{
				ID: fmt.Sprintf("c%d", i), PlantID: "p1", Type: "water",
				IntervalDays: i, LastDate: last, NextDate: last.AddDate(0, 0, i),
			})
		}(i)
	}
	wg.Wait()

	due, err = care.ListDueCycles("u1", time.Now())
	if err != nil {
		t.Fatalf("ListDueCycles post: %v", err)
	}
	if len(due) != 1 {
		t.Errorf("after concurrent updates expected 1 due reminder, got %d (old+new surfacing together)", len(due))
	} else {
		t.Logf("final reminder: plant=%s type=%s next=%s", due[0].Plant.Name, due[0].Type, due[0].NextDate.Format("2006-01-02"))
	}
}

func removeFile(path string) { os.Remove(path) }
