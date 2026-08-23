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
func TestInsightEndDateExcludesFollowingDay(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:bug008?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.User{}, &model.Plant{}, &model.CareLog{}, &model.CareCycle{}); err != nil {
		t.Fatal(err)
	}

	day := time.Date(2026, 8, 20, 12, 0, 0, 0, time.Local)
	if err := db.Create(&model.Plant{
		ID: "plant-008", UserID: "user-008", Name: "绿萝", Species: "pothos",
		Source: "market", Location: "balcony", Status: model.StatusHealthy, Difficulty: "easy",
	}).Error; err != nil {
		t.Fatal(err)
	}
	for _, log := range []model.CareLog{
		{ID: "log-008-end", PlantID: "plant-008", Type: model.CareWater, CreatedAt: day},
		{ID: "log-008-next", PlantID: "plant-008", Type: model.CareWater, CreatedAt: day.AddDate(0, 0, 1)},
	} {
		if err := db.Create(&log).Error; err != nil {
			t.Fatal(err)
		}
	}

	plants := repository.NewPlantRepository(db)
	care := repository.NewCareRepository(db)
	insightRepo := repository.NewInsightRepository(db)
	insights := service.NewInsightService(plants, care, insightRepo)
	plantService := service.NewPlantService(plants, care)
	insightHandler := handler.NewInsightHandler(insights, plantService)

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Set(middleware.UserIDKey, "user-008")
	date := day.Format("2006-01-02")
	ctx.Request = httptest.NewRequest("GET", "/api/v1/insights?from="+date+"&to="+date, nil)
	insightHandler.API(ctx)
	if recorder.Code != 200 {
		t.Fatalf("insight API returned %d: %s", recorder.Code, recorder.Body.String())
	}

	var report service.InsightReport
	if err := json.Unmarshal(recorder.Body.Bytes(), &report); err != nil {
		t.Fatal(err)
	}
	if len(report.Records) != 1 || report.Records[0].ID != "log-008-end" {
		t.Fatalf("selected end date included %d records, want 1", len(report.Records))
	}
}
