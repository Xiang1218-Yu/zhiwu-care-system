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
func TestZeroCycleCareRequestDoesNotCreateHistory(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:bug010?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.User{}, &model.Plant{}, &model.CareLog{}, &model.CareCycle{}); err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&model.Plant{
		ID: "plant-010", UserID: "user-010", Name: "吊兰", Species: "spider plant",
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
	ctx.Set(middleware.UserIDKey, "user-010")
	ctx.Params = gin.Params{{Key: "id", Value: "plant-010"}}
	ctx.Request = httptest.NewRequest("POST", "/api/v1/plants/plant-010/care", bytes.NewBufferString(
		`{"type":"water","water_cycle":0}`,
	))
	ctx.Request.Header.Set("Content-Type", "application/json")
	apiHandler.AddCare(ctx)

	var logCount int64
	if err := db.Model(&model.CareLog{}).Where("plant_id = ?", "plant-010").Count(&logCount).Error; err != nil {
		t.Fatal(err)
	}
	if recorder.Code == 201 || logCount != 0 {
		t.Fatalf("zero-day care request was accepted and stored %d log(s)", logCount)
	}
}
