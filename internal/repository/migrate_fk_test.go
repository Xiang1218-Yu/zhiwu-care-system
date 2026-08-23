package repository

import (
	"os"
	"testing"

	"plant-diary/internal/model"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Reproduce the real upgrade path: FK enabled, a care_cycles table that GORM
// itself created (correct FK schema, pre-fix non-unique index) and now carries a
// duplicate, then MigrateCycles. Because the table already matches the model's
// FK schema, the migration only needs to add the unique index — no rebuild, no
// FOREIGN KEY trip.
func TestMigrateCyclesWithForeignKeysOn(t *testing.T) {
	dbPath := "/tmp/pd_fk_test.db"
	os.Remove(dbPath)
	db, err := gorm.Open(sqlite.Open(dbPath+"?_foreign_keys=on"), &gorm.Config{Logger: logger.Default.LogMode(logger.Warn)})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := db.AutoMigrate(&model.User{}, &model.Plant{}, &model.CareLog{}, &model.CareCycle{}); err != nil {
		t.Fatalf("migrate (pre-fix schema): %v", err)
	}
	// drop the unique index the new model added, recreating the pre-fix non-unique
	// index, so the table looks exactly like an affected user's DB before the fix.
	if err := db.Exec(`DROP INDEX IF EXISTS idx_care_cycles_plant_type`).Error; err != nil {
		t.Fatalf("drop unique index: %v", err)
	}
	if err := db.Exec(`CREATE INDEX IF NOT EXISTS idx_cc_plant_id ON care_cycles(plant_id)`).Error; err != nil {
		t.Fatalf("create legacy index: %v", err)
	}
	db.Create(&model.User{ID: "u1", Email: "t@e", PasswordHash: "h", Name: "T"})
	db.Create(&model.Plant{ID: "p1", UserID: "u1", Name: "绿萝", Species: "E", Source: "market", Location: "balcony"})
	// the stale duplicate the race left behind
	db.Exec(`INSERT INTO care_cycles (id,plant_id,type,interval_days,last_date,next_date,created_at,updated_at) VALUES ('a','p1','water',3,'2026-01-01','2026-01-04','2026-01-01','2026-01-01')`)
	db.Exec(`INSERT INTO care_cycles (id,plant_id,type,interval_days,last_date,next_date,created_at,updated_at) VALUES ('b','p1','water',5,'2026-01-01','2026-01-06','2026-01-01','2026-01-01')`)

	care := NewCareRepository(db)
	if err := care.MigrateCycles(); err != nil {
		t.Fatalf("MigrateCycles with FK on: %v", err)
	}
	var rows []model.CareCycle
	db.Find(&rows)
	if len(rows) != 1 {
		t.Errorf("expected 1 survivor, got %d", len(rows))
	} else {
		t.Logf("survivor id=%s next=%s", rows[0].ID, rows[0].NextDate.Format("2006-01-02"))
	}
}
