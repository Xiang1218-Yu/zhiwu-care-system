package verification_test

import (
	"encoding/json"
	"net/http/httptest"
	"testing"
	"time"

	"plant-diary/internal/handler"
	"plant-diary/internal/middleware"
	"plant-diary/internal/model"
	"plant-diary/internal/repository"
	"plant-diary/internal/service"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// GET /api/v1/insights -> InsightHandler.API -> InsightService.Analyze -> InsightRepository.ListCareRecords
func TestInsightAPIKeepsUserRecordsIsolated(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:bug004?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.User{}, &model.Plant{}, &model.CareLog{}, &model.CareCycle{}); err != nil {
		t.Fatal(err)
	}
	today := time.Now()
	day := time.Date(today.Year(), today.Month(), today.Day(), 12, 0, 0, 0, today.Location())
	plants := []model.Plant{
		{ID: "plant-a", UserID: "user-a", Name: "用户A的绿萝", Species: "pothos", Source: "market", Location: "balcony", Status: model.StatusHealthy, Difficulty: "easy"},
		{ID: "plant-b", UserID: "user-b", Name: "用户B的龟背竹", Species: "monstera", Source: "online", Location: "living-room", Status: model.StatusHealthy, Difficulty: "medium"},
	}
	for index := range plants {
		if err := db.Create(&plants[index]).Error; err != nil {
			t.Fatal(err)
		}
	}
	for _, log := range []model.CareLog{
		{ID: "log-a", PlantID: "plant-a", Type: model.CareWater, Note: "A 的记录", PhotoURL: "/uploads/a.png", CreatedAt: day},
		{ID: "log-b", PlantID: "plant-b", Type: model.CareWater, Note: "B 的记录", PhotoURL: "/uploads/b.png", CreatedAt: day},
	} {
		if err := db.Create(&log).Error; err != nil {
			t.Fatal(err)
		}
	}

	plantRepo := repository.NewPlantRepository(db)
	careRepo := repository.NewCareRepository(db)
	insightRepo := repository.NewInsightRepository(db)
	insights := service.NewInsightService(plantRepo, careRepo, insightRepo)
	insightHandler := handler.NewInsightHandler(insights, nil)

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Set(middleware.UserIDKey, "user-a")
	ctx.Request = httptest.NewRequest("GET", "/api/v1/insights?from="+day.Format("2006-01-02")+"&to="+day.Format("2006-01-02"), nil)
	insightHandler.API(ctx)
	if recorder.Code != 200 {
		t.Fatalf("insight API returned %d: %s", recorder.Code, recorder.Body.String())
	}

	var report service.InsightReport
	if err := json.Unmarshal(recorder.Body.Bytes(), &report); err != nil {
		t.Fatal(err)
	}
	if len(report.Records) != 1 || report.Records[0].PlantID != "plant-a" {
		t.Fatalf("user-a received records outside the account boundary: %#v", report.Records)
	}
	if len(report.Gallery) != 1 || report.Gallery[0].PlantID != "plant-a" {
		t.Fatalf("user-a received another user's photo: %#v", report.Gallery)
	}
}
