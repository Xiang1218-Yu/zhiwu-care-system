package repository

import (
	"testing"
	"time"

	"plant-diary/internal/model"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newInsightTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared&_foreign_keys=on"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	if err := db.AutoMigrate(&model.Plant{}, &model.CareLog{}, &model.CareCycle{}); err != nil {
		t.Fatalf("migrate database: %v", err)
	}
	// Ensure a clean slate: file::memory:?cache=shared is shared across opens
	// within a process, so drop any rows left from a previous test.
	db.Exec("DELETE FROM care_logs")
	db.Exec("DELETE FROM plants")
	return db
}

func seedInsightRecords(t *testing.T, db *gorm.DB, userID string) (plantID string) {
	t.Helper()
	plantID = "plant-1"
	plant := &model.Plant{
		ID: plantID, UserID: userID, Name: "绿萝", Species: "Epipremnum",
		Source: "market", AcquiredDate: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		Location: "balcony", Status: model.StatusHealthy, Difficulty: "easy",
	}
	if err := db.Create(plant).Error; err != nil {
		t.Fatalf("seed plant: %v", err)
	}
	return plantID
}

func insertCareLog(t *testing.T, db *gorm.DB, plantID string, at time.Time) {
	t.Helper()
	log := &model.CareLog{
		ID: at.Format("20060102-150405"), PlantID: plantID, Type: model.CareWater,
	}
	if err := db.Create(log).Error; err != nil {
		t.Fatalf("seed care log at %s: %v", at, err)
	}
	// GORM auto-fills CreatedAt on Create; overwrite it to the exact boundary we want.
	if err := db.Model(&model.CareLog{}).Where("id = ?", log.ID).
		Update("created_at", at).Error; err != nil {
		t.Fatalf("set created_at: %v", err)
	}
}

// The insight range is half-open [from, to) where `to` already points at the
// start of the day AFTER the user's selected end date (the handler adds one
// day so that `< to` includes the end date itself and excludes anything after).
func TestListCareRecordsExcludesRecordsAfterEndDate(t *testing.T) {
	db := newInsightTestDB(t)
	const userID = "user-1"
	plantID := seedInsightRecords(t, db, userID)

	loc, _ := time.LoadLocation("Asia/Shanghai")

	// User picks from=2026-08-20, to=2026-08-22 (end date is the 22nd).
	from := time.Date(2026, 8, 20, 0, 0, 0, 0, loc)
	// `to` arrives here already advanced by one day: start of the 23rd, the
	// exclusive upper bound. This mirrors what InsightHandler.analyze passes in.
	to := time.Date(2026, 8, 23, 0, 0, 0, 0, loc)

	// Exactly one record on the selected end date — this is what we want kept.
	insertCareLog(t, db, plantID, time.Date(2026, 8, 22, 9, 0, 0, 0, loc))
	// One record the day after the end date — must NOT appear.
	insertCareLog(t, db, plantID, time.Date(2026, 8, 23, 10, 0, 0, 0, loc))
	// One more record two days after the end date — must NOT appear either.
	insertCareLog(t, db, plantID, time.Date(2026, 8, 24, 11, 0, 0, 0, loc))

	repo := NewInsightRepository(db)
	records, err := repo.ListCareRecords(userID, from, to, "", "")
	if err != nil {
		t.Fatalf("ListCareRecords: %v", err)
	}

	if len(records) != 1 {
		t.Fatalf("selected end date included %d records, want 1", len(records))
	}
	got := records[0].CreatedAt.In(loc).Format("2006-01-02")
	if got != "2026-08-22" {
		t.Fatalf("expected the single end-date record, got %s", got)
	}
}
