package verification

import (
	"bytes"
	"html/template"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
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

func TestEditWithoutNewAvatarPreservesExistingAsset(t *testing.T) {
	// Covers ViewHandler.UpdatePlant, PlantService.Update, PlantRepository.Save,
	// and utils.DeleteUpload.
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared&_foreign_keys=on"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.User{}, &model.Plant{}, &model.CareLog{}, &model.CareCycle{}); err != nil {
		t.Fatal(err)
	}

	const userID = "user-avatar"
	const plantID = "plant-avatar"
	const avatarURL = "/uploads/existing-avatar.png"
	uploadDir := t.TempDir()
	avatarPath := filepath.Join(uploadDir, "existing-avatar.png")
	if err := os.WriteFile(avatarPath, []byte("avatar"), 0600); err != nil {
		t.Fatal(err)
	}
	plant := &model.Plant{
		ID: plantID, UserID: userID, Name: "绿萝", Species: "Epipremnum",
		Source: "market", Location: "balcony", Status: model.StatusHealthy,
		Difficulty: "easy", AcquiredDate: time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC),
		AvatarURL: avatarURL,
	}
	if err := db.Create(&model.User{
		ID: userID, Email: "avatar@example.com", Name: "头像用户",
		PasswordHash: "hash",
	}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(plant).Error; err != nil {
		t.Fatal(err)
	}

	plants := repository.NewPlantRepository(db)
	care := repository.NewCareRepository(db)
	plantService := service.NewPlantService(plants, care)
	view := handler.NewViewHandler(nil, plantService, nil, nil, uploadDir)

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	fields := map[string]string{
		"name": "绿萝更新", "species": "Epipremnum", "source": "market",
		"acquired_date": "2026-08-23", "location": "balcony",
		"status": model.StatusHealthy, "difficulty": "easy",
	}
	for key, value := range fields {
		if err := writer.WriteField(key, value); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/plants/"+plantID+"/update", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	recorder := httptest.NewRecorder()
	c, engine := gin.CreateTestContext(recorder)
	engine.SetHTMLTemplate(template.Must(template.New("plant_form.html").Parse("{{.Error}}")))
	c.Request = req
	c.Params = gin.Params{{Key: "id", Value: plantID}}
	c.Set(middleware.UserIDKey, userID)

	view.UpdatePlant(c)
	if recorder.Header().Get("Location") == "" {
		t.Fatalf("update did not complete successfully: status %d body %s", recorder.Code, recorder.Body.String())
	}

	var saved model.Plant
	if err := db.First(&saved, "id = ?", plantID).Error; err != nil {
		t.Fatal(err)
	}
	if saved.AvatarURL != avatarURL {
		t.Fatalf("existing avatar reference was lost: got %q", saved.AvatarURL)
	}
	if _, err := os.Stat(avatarPath); err != nil {
		t.Fatalf("existing avatar file was removed: %v", err)
	}
}
