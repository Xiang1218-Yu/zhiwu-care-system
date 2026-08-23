package verification_test

import (
	"bytes"
	"net/http/httptest"
	"testing"

	"plant-diary/internal/handler"
	"plant-diary/internal/middleware"
	"plant-diary/internal/model"
	"plant-diary/internal/repository"
	"plant-diary/internal/service"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// POST /api/v1/plants/:id/care -> APIHandler.AddCare -> PlantService.AddCare -> CareRepository.Transaction
func TestCareAPIIsAtomicWhenCycleWriteFails(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:bug005?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.User{}, &model.Plant{}, &model.CareLog{}, &model.CareCycle{}); err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`CREATE TRIGGER reject_long_cycle BEFORE INSERT ON care_cycles
WHEN NEW.interval_days > 30
BEGIN SELECT RAISE(ABORT, 'cycle interval too long'); END;`).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&model.Plant{
		ID: "plant-1", UserID: "user-1", Name: "吊兰", Species: "spider plant",
		Source: "market", Location: "balcony", Status: model.StatusHealthy, Difficulty: "easy",
	}).Error; err != nil {
		t.Fatal(err)
	}

	plants := repository.NewPlantRepository(db)
	care := repository.NewCareRepository(db)
	apiHandler := handler.NewAPIHandler(service.NewPlantService(plants, care), nil, nil)
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Set(middleware.UserIDKey, "user-1")
	ctx.Params = gin.Params{{Key: "id", Value: "plant-1"}}
	ctx.Request = httptest.NewRequest("POST", "/api/v1/plants/plant-1/care", bytes.NewBufferString(
		`{"type":"water","water_cycle":31}`,
	))
	ctx.Request.Header.Set("Content-Type", "application/json")
	apiHandler.AddCare(ctx)

	if recorder.Code == 201 {
		t.Fatalf("care API reported success after the cycle write failed: %s", recorder.Body.String())
	}
	var logCount int64
	if err := db.Model(&model.CareLog{}).Where("plant_id = ?", "plant-1").Count(&logCount).Error; err != nil {
		t.Fatal(err)
	}
	if logCount != 0 {
		t.Fatalf("care log survived a failed cycle write; expected atomic rollback, got %d logs", logCount)
	}
}
