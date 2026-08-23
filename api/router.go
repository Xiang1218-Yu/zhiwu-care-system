package api

import (
	"html/template"
	"net/http"

	"plant-diary/internal/handler"
	"plant-diary/internal/middleware"
	"plant-diary/internal/service"

	"github.com/gin-gonic/gin"
)

func NewRouter(auth *service.AuthService, views *handler.ViewHandler, apiHandler *handler.APIHandler, insightHandler *handler.InsightHandler, templates *template.Template) *gin.Engine {
	router := gin.New()
	router.Use(gin.Logger(), gin.Recovery(), middleware.CORS())
	router.SetHTMLTemplate(templates)
	router.Static("/static", "./static")
	router.Static("/uploads", "./uploads")

	authHandler := handler.NewAuthHandler(auth)
	router.GET("/login", authHandler.ShowLogin)
	router.POST("/login", authHandler.LoginPage)
	router.GET("/register", authHandler.ShowRegister)
	router.POST("/register", authHandler.RegisterPage)
	router.GET("/logout", authHandler.Logout)

	pageAuth := middleware.RequireAuth(auth, true)
	router.GET("/", pageAuth, views.Dashboard)
	router.GET("/plants/new", pageAuth, views.NewPlant)
	router.POST("/plants", pageAuth, views.CreatePlant)
	router.GET("/plants/:id", pageAuth, views.ShowPlant)
	router.GET("/plants/:id/edit", pageAuth, views.EditPlant)
	router.POST("/plants/:id/update", pageAuth, views.UpdatePlant)
	router.POST("/plants/:id/delete", pageAuth, views.DeletePlant)
	router.POST("/plants/:id/care", pageAuth, views.AddCare)
	router.POST("/plants/:id/cycles", pageAuth, views.SetCycle)
	router.POST("/plants/:id/cycles/:type/delete", pageAuth, views.DeleteCycle)
	router.GET("/insights", pageAuth, insightHandler.Show)
	router.GET("/insights/export.csv", pageAuth, insightHandler.ExportCSV)

	api := router.Group("/api/v1")
	api.POST("/auth/login", authHandler.LoginAPI)
	api.POST("/auth/register", authHandler.RegisterAPI)
	protected := api.Group("")
	protected.Use(middleware.RequireAuth(auth, false))
	protected.GET("/plants", apiHandler.ListPlants)
	protected.POST("/plants", apiHandler.CreatePlant)
	protected.GET("/plants/:id", apiHandler.GetPlant)
	protected.PUT("/plants/:id", apiHandler.UpdatePlant)
	protected.DELETE("/plants/:id", apiHandler.DeletePlant)
	protected.GET("/plants/:id/timeline", apiHandler.Timeline)
	protected.POST("/plants/:id/care", apiHandler.AddCare)
	protected.PUT("/plants/:id/cycles", apiHandler.SetCycle)
	protected.DELETE("/plants/:id/cycles/:type", apiHandler.DeleteCycle)
	protected.GET("/today/reminders", apiHandler.Reminders)
	protected.GET("/stats", apiHandler.Stats)
	protected.GET("/insights", insightHandler.API)

	router.NoRoute(func(c *gin.Context) {
		c.HTML(http.StatusNotFound, "error.html", gin.H{"Title": "页面不存在", "Error": "你访问的页面不存在"})
	})
	return router
}
