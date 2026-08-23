package verification_test

import (
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

// GET /api/v1/stats -> APIHandler.Stats -> StatsService.Get -> CareRepository.CountLogsAfter
func TestStatsAPIPropagatesRecentLogQueryError(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:bug002?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.User{}, &model.Plant{}, &model.CareLog{}); err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&model.Plant{
		ID: "plant-1", UserID: "user-1", Name: "龟背竹", Species: "monstera",
		Source: "online", Location: "living-room", Status: model.StatusHealthy, Difficulty: "medium",
	}).Error; err != nil {
		t.Fatal(err)
	}

	plants := repository.NewPlantRepository(db)
	care := repository.NewCareRepository(db)
	stats := service.NewStatsService(plants, care)
	apiHandler := handler.NewAPIHandler(nil, nil, stats)

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Set(middleware.UserIDKey, "user-1")
	ctx.Request = httptest.NewRequest("GET", "/api/v1/stats", nil)
	apiHandler.Stats(ctx)

	if recorder.Code != 500 {
		t.Fatalf("expected stats API to expose the storage error with HTTP 500, got %d: %s", recorder.Code, recorder.Body.String())
	}
}
