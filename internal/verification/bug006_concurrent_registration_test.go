package verification

import (
	"fmt"
	"sync"
	"testing"

	"plant-diary/internal/model"
	"plant-diary/internal/repository"
	"plant-diary/internal/service"

	"github.com/google/uuid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestConcurrentRegistrationKeepsOneAccount(t *testing.T) {
	// Covers AuthService.Register, UserRepository.Create, and User.Email.
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared&_foreign_keys=on"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.User{}, &model.Plant{}); err != nil {
		t.Fatal(err)
	}

	users := repository.NewUserRepository(db)
	auth := service.NewAuthService(users, "verification-secret", 1)
	email := fmt.Sprintf("parallel-%s@example.com", uuid.NewString())

	const workers = 2
	start := make(chan struct{})
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			_, _, _ = auth.Register("并发用户", email, "password123")
		}()
	}
	close(start)
	wg.Wait()

	var count int64
	if err := db.Model(&model.User{}).Where("email = ?", email).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("expected exactly one account for email, got %d", count)
	}
}
