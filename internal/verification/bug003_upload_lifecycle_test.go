package verification_test

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
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

// POST /plants/:id/care -> ViewHandler.AddCare -> PlantService.AddCare -> utils.SaveUpload -> utils.DeleteUploadAfterUse
func TestSuccessfulCarePhotoRemainsAvailable(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:bug003?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.User{}, &model.Plant{}, &model.CareLog{}, &model.CareCycle{}); err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&model.Plant{
		ID: "plant-1", UserID: "user-1", Name: "琴叶榕", Species: "fiddle-leaf fig",
		Source: "friend", Location: "study", Status: model.StatusHealthy, Difficulty: "hard",
	}).Error; err != nil {
		t.Fatal(err)
	}

	plants := repository.NewPlantRepository(db)
	care := repository.NewCareRepository(db)
	uploadDir := t.TempDir()
	views := handler.NewViewHandler(nil, service.NewPlantService(plants, care), nil, nil, uploadDir)
	recorder := httptest.NewRecorder()
	body := newUploadBody(t, uploadDir)
	request := httptest.NewRequest(http.MethodPost, "/plants/plant-1/care", bytes.NewReader(body.Bytes()))
	request.Header.Set("Content-Type", body.contentType)
	gin.SetMode(gin.TestMode)
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Set(middleware.UserIDKey, "user-1")
	ctx.Params = gin.Params{{Key: "id", Value: "plant-1"}}
	ctx.Request = request
	views.AddCare(ctx)
	ctx.Writer.WriteHeaderNow()

	if recorder.Code != http.StatusFound {
		t.Fatalf("expected successful care submission redirect, got %d: %s", recorder.Code, recorder.Body.String())
	}
	var log model.CareLog
	if err := db.Where("plant_id = ?", "plant-1").First(&log).Error; err != nil {
		t.Fatal(err)
	}
	if log.PhotoURL == "" {
		t.Fatal("successful care record did not retain a photo reference")
	}
	if _, err := os.Stat(filepath.Join(body.directory, filepath.Base(log.PhotoURL))); err != nil {
		t.Fatalf("photo referenced by a successful care record is unavailable: %v", err)
	}
}

type uploadBody struct {
	bytes.Buffer
	contentType string
	directory   string
}

func newUploadBody(t *testing.T, directory string) *uploadBody {
	t.Helper()
	result := &uploadBody{directory: directory}
	writer := multipart.NewWriter(&result.Buffer)
	file, err := writer.CreateFormFile("photo", "leaf.png")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := file.Write([]byte{
		0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a,
		0x00, 0x00, 0x00, 0x0d, 0x49, 0x48, 0x44, 0x52,
	}); err != nil {
		t.Fatal(err)
	}
	_ = writer.WriteField("type", model.CareWater)
	_ = writer.WriteField("note", "拍照记录")
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	result.contentType = writer.FormDataContentType()
	return result
}
