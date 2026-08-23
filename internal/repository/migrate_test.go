package repository

import (
	"os"
	"testing"

	"plant-diary/internal/model"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// simulates an existing DB created with the old non-unique index, then re-migrated
func TestAutoMigrateUpgradesIndexToUnique(t *testing.T) {
	dbPath := "/tmp/pd_migrate_test.db"
	os.Remove(dbPath)
	sqlDB, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	// create table with the OLD schema (non-unique indexes on plant_id and type)
	if err := sqlDB.Exec(`CREATE TABLE care_cycles (
		id TEXT PRIMARY KEY,
		plant_id TEXT NOT NULL,
		type TEXT NOT NULL,
		interval_days INTEGER NOT NULL,
		last_date DATE NOT NULL,
		next_date DATE NOT NULL,
		created_at DATETIME,
		updated_at DATETIME)`).Error; err != nil {
		t.Fatalf("create old table: %v", err)
	}
	if err := sqlDB.Exec(`CREATE INDEX idx_care_cycles_plant_id ON care_cycles(plant_id)`).Error; err != nil {
		t.Fatalf("create old plant_id index: %v", err)
	}
	// insert a duplicate to mimic existing bad data
	sqlDB.Exec(`INSERT INTO care_cycles (id,plant_id,type,interval_days,last_date,next_date,created_at,updated_at) VALUES ('a','p','water',3,'2026-01-01','2026-01-04','2026-01-01','2026-01-01')`)
	sqlDB.Exec(`INSERT INTO care_cycles (id,plant_id,type,interval_days,last_date,next_date,created_at,updated_at) VALUES ('b','p','water',5,'2026-01-01','2026-01-06','2026-01-01','2026-01-01')`)

	care := NewCareRepository(sqlDB)
	// MigrateCycles is the path main.go takes: it tolerates the duplicate by
	// deduplicating then retrying, so the unique index lands cleanly.
	if err := care.MigrateCycles(); err != nil {
		t.Fatalf("MigrateCycles: %v", err)
	}
	var rows []model.CareCycle
	sqlDB.Find(&rows)
	if len(rows) != 1 {
		t.Errorf("expected 1 survivor row, got %d", len(rows))
	} else {
		t.Logf("survivor id=%s interval=%d next=%s", rows[0].ID, rows[0].IntervalDays, rows[0].NextDate.Format("2006-01-02"))
	}
	// confirm the unique index actually exists after migration
	var idxCount int64
	sqlDB.Raw(`SELECT count(*) FROM sqlite_master
		WHERE type='index' AND tbl_name='care_cycles'
		AND sql LIKE '%UNIQUE%plant_id%type%'`).Scan(&idxCount)
	if idxCount != 1 {
		t.Errorf("expected a UNIQUE(plant_id,type) index, found %d", idxCount)
	}
}
