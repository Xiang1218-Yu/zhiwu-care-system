package handler

import (
	"net/http"

	"plant-diary/api/dto"
	"plant-diary/internal/middleware"
	"plant-diary/internal/service"
	"plant-diary/pkg/utils"

	"github.com/gin-gonic/gin"
)

type APIHandler struct {
	plants    *service.PlantService
	reminders *service.ReminderService
	stats     *service.StatsService
}

func NewAPIHandler(plants *service.PlantService, reminders *service.ReminderService, stats *service.StatsService) *APIHandler {
	return &APIHandler{plants: plants, reminders: reminders, stats: stats}
}

func (h *APIHandler) ListPlants(c *gin.Context) {
	plants, err := h.plants.List(middleware.UserID(c), c.Query("status"), c.Query("location"))
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, plants)
}

func (h *APIHandler) CreatePlant(c *gin.Context) {
	var input dto.PlantInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(400, gin.H{"error": "请求格式不正确"})
		return
	}
	plant, err := h.plants.Create(middleware.UserID(c), input, "")
	if err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	c.JSON(201, plant)
}

func (h *APIHandler) GetPlant(c *gin.Context) {
	plant, err := h.plants.Get(middleware.UserID(c), c.Param("id"))
	if err != nil {
		c.JSON(404, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, plant)
}

func (h *APIHandler) UpdatePlant(c *gin.Context) {
	var input dto.PlantInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(400, gin.H{"error": "请求格式不正确"})
		return
	}
	if err := h.plants.Update(middleware.UserID(c), c.Param("id"), input, ""); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	h.GetPlant(c)
}

func (h *APIHandler) DeletePlant(c *gin.Context) {
	if err := h.plants.Delete(middleware.UserID(c), c.Param("id")); err != nil {
		c.JSON(404, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *APIHandler) Timeline(c *gin.Context) {
	plant, err := h.plants.Get(middleware.UserID(c), c.Param("id"))
	if err != nil {
		c.JSON(404, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"plant": plant, "logs": plant.CareLogs})
}

func (h *APIHandler) AddCare(c *gin.Context) {
	var input dto.CareInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(400, gin.H{"error": "请求格式不正确"})
		return
	}
	// 周期校验统一交由 service 处理：浇水/施肥要求 1..365，避免记录与待办分叉。
	if err := h.plants.AddCare(middleware.UserID(c), c.Param("id"), input, ""); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	c.JSON(201, gin.H{"message": "养护记录已保存", "date": utils.FormatDate(utils.Today())})
}

func (h *APIHandler) SetCycle(c *gin.Context) {
	var input dto.CycleInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(400, gin.H{"error": "请求格式不正确"})
		return
	}
	cycle, err := h.plants.SetCycle(middleware.UserID(c), c.Param("id"), input)
	if err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, cycle)
}

func (h *APIHandler) DeleteCycle(c *gin.Context) {
	if err := h.plants.DeleteCycle(middleware.UserID(c), c.Param("id"), c.Param("type")); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *APIHandler) Reminders(c *gin.Context) {
	reminders, err := h.reminders.Due(middleware.UserID(c))
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, reminders)
}

func (h *APIHandler) Stats(c *gin.Context) {
	stats, err := h.stats.Get(middleware.UserID(c))
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, stats)
}
