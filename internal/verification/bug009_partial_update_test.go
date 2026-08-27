package verification

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"plant-diary/api/dto"
	"plant-diary/internal/handler"
	"plant-diary/internal/middleware"
	"plant-diary/internal/model"
	"plant-diary/internal/repository"
	"plant-diary/internal/service"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestPartialPlantUpdatePreservesOmittedFields(t *testing.T) {
	// Covers APIHandler.UpdatePlant, PlantService.Update, and PlantRepository.Save.
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared&_foreign_keys=on"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.User{}, &model.Plant{}, &model.CareLog{}, &model.CareCycle{}); err != nil {
		t.Fatal(err)
	}

	const userID = "user-partial"
	const plantID = "plant-partial"
	original := &model.Plant{
		ID: plantID, UserID: userID, Name: "旧昵称", Species: "Monstera",
		Source: "online", Location: "living-room", Status: model.StatusYellowing,
		Difficulty: "medium", AcquiredDate: time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC),
	}
	if err := db.Create(&model.User{
		ID: userID, Email: "partial@example.com", Name: "部分更新用户",
		PasswordHash: "hash",
	}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(original).Error; err != nil {
		t.Fatal(err)
	}

	plants := repository.NewPlantRepository(db)
	care := repository.NewCareRepository(db)
	plantService := service.NewPlantService(plants, care)
	api := handler.NewAPIHandler(plantService, nil, nil)

	body, err := json.Marshal(dto.PlantInput{Name: "新昵称"})
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPut, "/plants/"+plantID, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = req
	c.Params = gin.Params{{Key: "id", Value: plantID}}
	c.Set(middleware.UserIDKey, userID)

	api.UpdatePlant(c)
	if recorder.Code != http.StatusOK {
		t.Fatalf("partial update was not accepted: status %d body %s", recorder.Code, recorder.Body.String())
	}

	var saved model.Plant
	if err := db.First(&saved, "id = ?", plantID).Error; err != nil {
		t.Fatal(err)
	}
	if saved.Name != "新昵称" ||
		saved.Species != original.Species ||
		saved.Location != original.Location ||
		saved.Status != original.Status {
		t.Fatalf("omitted plant fields were not preserved: %#v", saved)
	}
}
