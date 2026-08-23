package verification_test

import (
	"bytes"
	"encoding/json"
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

// PUT /api/v1/plants/:id/cycles -> APIHandler.SetCycle -> PlantService.SetCycle -> CareRepository.UpsertCycle
func TestCycleUpdateKeepsCurrentCycleIdentity(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:bug001?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.User{}, &model.Plant{}, &model.CareLog{}, &model.CareCycle{}); err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&model.Plant{
		ID: "plant-1", UserID: "user-1", Name: "绿萝", Species: "pothos",
		Source: "market", Location: "balcony", Status: model.StatusHealthy, Difficulty: "easy",
	}).Error; err != nil {
		t.Fatal(err)
	}

	plants := repository.NewPlantRepository(db)
	care := repository.NewCareRepository(db)
	apiHandler := handler.NewAPIHandler(service.NewPlantService(plants, care), nil, nil)

	first := callCycleUpdate(t, apiHandler, "7")
	second := callCycleUpdate(t, apiHandler, "14")
	if first.ID == "" || second.ID == "" {
		t.Fatalf("cycle update responses must include an id: first=%q second=%q", first.ID, second.ID)
	}
	if first.ID != second.ID {
		t.Fatalf("updating one cycle changed its identity: first=%q second=%q", first.ID, second.ID)
	}

	var cycles []model.CareCycle
	if err := db.Where("plant_id = ? AND type = ?", "plant-1", model.CareWater).Find(&cycles).Error; err != nil {
		t.Fatal(err)
	}
	if len(cycles) != 1 || cycles[0].IntervalDays != 14 {
		t.Fatalf("expected one current 14-day cycle, got %#v", cycles)
	}
}

func callCycleUpdate(t *testing.T, apiHandler *handler.APIHandler, interval string) model.CareCycle {
	t.Helper()
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Set(middleware.UserIDKey, "user-1")
	ctx.Params = gin.Params{{Key: "id", Value: "plant-1"}}
	ctx.Request = httptest.NewRequest("PUT", "/api/v1/plants/plant-1/cycles", bytes.NewBufferString(
		`{"type":"water","interval_days":`+interval+`}`,
	))
	ctx.Request.Header.Set("Content-Type", "application/json")
	apiHandler.SetCycle(ctx)
	if recorder.Code != 200 {
		t.Fatalf("cycle update returned %d: %s", recorder.Code, recorder.Body.String())
	}
	var cycle model.CareCycle
	if err := json.Unmarshal(recorder.Body.Bytes(), &cycle); err != nil {
		t.Fatal(err)
	}
	return cycle
}
