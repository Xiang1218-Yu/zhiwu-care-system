package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"time"

	"plant-diary/api"
	"plant-diary/config"
	"plant-diary/internal/handler"
	"plant-diary/internal/model"
	"plant-diary/internal/repository"
	"plant-diary/internal/service"
	"plant-diary/pkg/utils"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}
	db, err := gorm.Open(sqlite.Open(cfg.Database+"?_foreign_keys=on"), &gorm.Config{})
	if err != nil {
		log.Fatalf("open database: %v", err)
	}
	if err := db.AutoMigrate(&model.User{}, &model.Plant{}, &model.CareLog{}); err != nil {
		log.Fatalf("migrate database: %v", err)
	}
	if err := repository.NewCareRepository(db).MigrateCycles(); err != nil {
		log.Fatalf("migrate care cycles: %v", err)
	}

	users := repository.NewUserRepository(db)
	plants := repository.NewPlantRepository(db)
	care := repository.NewCareRepository(db)
	insightRepo := repository.NewInsightRepository(db)
	auth := service.NewAuthService(users, cfg.JWTSecret, cfg.JWTExpires)
	plantService := service.NewPlantService(plants, care)
	reminderService := service.NewReminderService(care)
	statsService := service.NewStatsService(plants, care)
	insightService := service.NewInsightService(plants, care, insightRepo)
	views := handler.NewViewHandler(users, plantService, reminderService, statsService, cfg.UploadDir)
	apiHandler := handler.NewAPIHandler(plantService, reminderService, statsService)
	insightHandler := handler.NewInsightHandler(insightService, plantService)
	router := api.NewRouter(auth, views, apiHandler, insightHandler, loadTemplates())

	server := &http.Server{
		Addr:              fmt.Sprintf(":%d", cfg.Port),
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
	}
	log.Printf("plant diary is running at http://localhost:%d", cfg.Port)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}

func loadTemplates() *template.Template {
	return template.Must(template.New("").Funcs(template.FuncMap{
		"date":  utils.FormatDate,
		"today": func() string { return utils.FormatDate(utils.Today()) },
		"daysText": func(days int) string {
			switch {
			case days < 0:
				return fmt.Sprintf("逾期 %d 天", -days)
			case days == 0:
				return "今天"
			default:
				return fmt.Sprintf("%d 天后", days)
			}
		},
		"statusClass": func(status string) string {
			switch status {
			case "healthy":
				return "success"
			case "yellowing", "pests":
				return "warning"
			default:
				return "muted"
			}
		},
		"careLabel": func(value string) string {
			return map[string]string{
				"water": "浇水", "fertilizer": "施肥", "repot": "换盆",
				"prune": "修剪", "spray": "喷药", "clean": "擦拭叶片",
			}[value]
		},
		"trendWidth": func(count, max int) int {
			if max <= 0 || count <= 0 {
				return 4
			}
			return count * 100 / max
		},
		"maxTrend": func(points []service.TrendPoint) int {
			maximum := 0
			for _, point := range points {
				if point.Count > maximum {
					maximum = point.Count
				}
			}
			return maximum
		},
	}).ParseGlob("templates/*.html"))
}
