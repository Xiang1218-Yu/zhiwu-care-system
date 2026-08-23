package handler

import (
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"html/template"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"plant-diary/internal/middleware"
	"plant-diary/internal/model"
	"plant-diary/internal/repository"
	"plant-diary/internal/service"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// setupTestRouter wires a minimal view handler stack backed by an in-memory
// SQLite database and a real on-disk upload directory, so we can exercise the
// full care-photo lifecycle (save -> persist -> clean up) end to end.
func setupTestRouter(t *testing.T) (*gin.Engine, *ViewHandler, string, *gorm.DB) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	// Give each test its own named in-memory database so seeded rows don't
	// leak between cases (cache=shared keeps one DB per DSN across the process).
	sum := sha1.Sum([]byte(t.Name()))
	dsn := "file:" + hex.EncodeToString(sum[:]) + "?mode=memory&cache=shared&_foreign_keys=on"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&model.Plant{}, &model.CareLog{}, &model.CareCycle{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	uploadDir := t.TempDir()
	plants := repository.NewPlantRepository(db)
	care := repository.NewCareRepository(db)
	plantService := service.NewPlantService(plants, care)
	views := NewViewHandler(nil, plantService, nil, nil, uploadDir)

	const userID = "user-1"
	plant := &model.Plant{
		ID: "plant-1", UserID: userID, Name: "绿萝", Species: "Epipremnum",
		Source: "market", Location: "balcony", Status: model.StatusHealthy, Difficulty: "easy",
	}
	if err := db.Create(plant).Error; err != nil {
		t.Fatalf("seed plant: %v", err)
	}

	r := gin.New()
	// The failure path renders error.html, so load just that template
	// (the full glob needs main.go's func map; AddCare only touches error.html).
	tmpl := template.Must(template.ParseFiles(filepath.Join("..", "..", "templates", "error.html")))
	r.SetHTMLTemplate(tmpl)
	// Inject the auth user id the same way middleware.UserID reads it back,
	// so the handler sees an authenticated owner without a real JWT.
	r.Use(func(c *gin.Context) { c.Set(middleware.UserIDKey, userID); c.Next() })
	r.POST("/plants/:id/care", views.AddCare)
	return r, views, uploadDir, db
}

// photoRequest builds a multipart/form-data POST for AddCare.
func photoRequest(t *testing.T, url string, plantID string, photo []byte, photoName, careType string) *http.Request {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	if photo != nil {
		part, err := writer.CreateFormFile("photo", photoName)
		if err != nil {
			t.Fatalf("create form file: %v", err)
		}
		if _, err := part.Write(photo); err != nil {
			t.Fatalf("write photo: %v", err)
		}
	}
	_ = writer.WriteField("type", careType)
	_ = writer.WriteField("note", "今天长新叶了")
	if err := writer.Close(); err != nil {
		t.Fatalf("close writer: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/plants/"+plantID+"/care", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	return req
}

// validPNG is a minimal 1x1 PNG so SaveUpload's content-type sniffing passes.
var validPNG = []byte{
	0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a,
	0x00, 0x00, 0x00, 0x0d, 0x49, 0x48, 0x44, 0x52,
	0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
	0x08, 0x06, 0x00, 0x00, 0x00, 0x1f, 0x15, 0xc4,
	0x89, 0x00, 0x00, 0x00, 0x0d, 0x49, 0x44, 0x41,
	0x54, 0x78, 0x9c, 0x63, 0x00, 0x01, 0x00, 0x00,
	0x05, 0x00, 0x01, 0x0d, 0x0a, 0x2d, 0xb4, 0x00,
	0x00, 0x00, 0x00, 0x49, 0x45, 0x4e, 0x44, 0xae,
	0x42, 0x60, 0x82,
}

func uploadedFiles(t *testing.T, dir string) []string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read upload dir: %v", err)
	}
	var names []string
	for _, e := range entries {
		if !e.IsDir() {
			names = append(names, e.Name())
		}
	}
	return names
}

// TestAddCare_KeepsPhotoOnSuccess is the regression test for the original bug:
// a deferred "delete after use" wiped the photo even on the success path, so
// the DB kept a URL pointing at a file that no longer existed on disk.
func TestAddCare_KeepsPhotoOnSuccess(t *testing.T) {
	r, _, uploadDir, db := setupTestRouter(t)

	req := photoRequest(t, "/plants/plant-1/care", "plant-1", validPNG, "leaf.png", "water")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusFound {
		t.Fatalf("expected 302 redirect on success, got %d (body=%q)", w.Code, w.Body.String())
	}

	// The photo must survive on disk.
	files := uploadedFiles(t, uploadDir)
	if len(files) != 1 {
		t.Fatalf("expected photo to remain on disk after success, found %d: %v", len(files), files)
	}

	// And the stored care_log row must reference it with the /uploads/ prefix
	// the templates render as <img src="...">.
	var log model.CareLog
	if err := db.First(&log, "plant_id = ?", "plant-1").Error; err != nil {
		t.Fatalf("load care log: %v", err)
	}
	if !strings.HasPrefix(log.PhotoURL, "/uploads/") {
		t.Fatalf("stored PhotoURL %q must start with /uploads/ so it resolves through the static route", log.PhotoURL)
	}
	expected := filepath.Join(uploadDir, strings.TrimPrefix(log.PhotoURL, "/uploads/"))
	if _, err := os.Stat(expected); err != nil {
		t.Fatalf("stored URL %q does not point at an existing file (%v): %v", log.PhotoURL, expected, err)
	}
}

// TestAddCare_DeletesPhotoOnFailure guards the inverse case: when the service
// rejects the record (here: an unsupported care type), the just-written upload
// must be cleaned up so it doesn't linger as an unreferenced orphan.
func TestAddCare_DeletesPhotoOnFailure(t *testing.T) {
	r, _, uploadDir, _ := setupTestRouter(t)

	// "repot" is valid, so use a bogus type to force a service-level rejection
	// after the photo has already landed on disk.
	req := photoRequest(t, "/plants/plant-1/care", "plant-1", validPNG, "leaf.png", "not-a-care-type")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 on invalid care type, got %d", w.Code)
	}

	files := uploadedFiles(t, uploadDir)
	if len(files) != 0 {
		t.Fatalf("expected orphan photo to be cleaned up on failure, found %d: %v", len(files), files)
	}
}

// TestAddCare_NoPhotoStillSaves makes sure removing the cleanup-on-success path
// didn't break the no-photo happy path.
func TestAddCare_NoPhotoStillSaves(t *testing.T) {
	r, _, uploadDir, db := setupTestRouter(t)

	req := photoRequest(t, "/plants/plant-1/care", "plant-1", nil, "", "water")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusFound {
		t.Fatalf("expected 302 on success, got %d", w.Code)
	}
	if files := uploadedFiles(t, uploadDir); len(files) != 0 {
		t.Fatalf("upload dir should stay empty when no photo was sent, found %v", files)
	}
	var log model.CareLog
	if err := db.First(&log, "plant_id = ?", "plant-1").Error; err != nil {
		t.Fatalf("load care log: %v", err)
	}
	if log.PhotoURL != "" {
		t.Fatalf("expected empty PhotoURL when no photo sent, got %q", log.PhotoURL)
	}
}
