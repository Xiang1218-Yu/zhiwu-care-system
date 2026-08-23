package handler

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"plant-diary/api/dto"
	"plant-diary/internal/middleware"
	"plant-diary/internal/model"
	"plant-diary/internal/repository"
	"plant-diary/internal/service"
	"plant-diary/pkg/utils"

	"github.com/gin-gonic/gin"
)

type ViewHandler struct {
	users     *repository.UserRepository
	plants    *service.PlantService
	reminders *service.ReminderService
	stats     *service.StatsService
	uploadDir string
}

type DashboardData struct {
	Title     string
	User      *model.User
	Plants    []model.Plant
	Reminders []service.Reminder
	Stats     service.Stats
	Status    string
	Location  string
	Error     string
}

func NewViewHandler(users *repository.UserRepository, plants *service.PlantService, reminders *service.ReminderService, stats *service.StatsService, uploadDir string) *ViewHandler {
	return &ViewHandler{users: users, plants: plants, reminders: reminders, stats: stats, uploadDir: uploadDir}
}

func (h *ViewHandler) Dashboard(c *gin.Context) {
	userID := middleware.UserID(c)
	user, err := h.users.FindByID(userID)
	if err != nil || user == nil {
		c.Redirect(http.StatusFound, "/login")
		return
	}
	status, location := c.Query("status"), c.Query("location")
	plants, err := h.plants.List(userID, status, location)
	if err != nil {
		h.renderDashboard(c, DashboardData{Title: "我的植物", User: user, Error: "读取植物列表失败"})
		return
	}
	reminders, _ := h.reminders.Due(userID)
	stats, _ := h.stats.Get(userID)
	h.renderDashboard(c, DashboardData{
		Title: "我的植物", User: user, Plants: plants, Reminders: reminders,
		Stats: stats, Status: status, Location: location,
	})
}

func (h *ViewHandler) renderDashboard(c *gin.Context, data DashboardData) {
	c.HTML(http.StatusOK, "dashboard.html", data)
}

func (h *ViewHandler) NewPlant(c *gin.Context) {
	c.HTML(http.StatusOK, "plant_form.html", gin.H{
		"Title": "添加植物", "Action": "/plants", "Submit": "保存植物",
		"Plant": model.Plant{Status: model.StatusHealthy, Difficulty: "easy"},
	})
}

func (h *ViewHandler) CreatePlant(c *gin.Context) {
	avatarURL, err := h.saveFile(c, "avatar")
	if err != nil {
		h.formError(c, "添加植物", "/plants", err)
		return
	}
	input := plantInput(c)
	if _, err := h.plants.Create(middleware.UserID(c), input, avatarURL); err != nil {
		_ = utils.DeleteUpload(avatarURL, h.uploadDir)
		h.formError(c, "添加植物", "/plants", err)
		return
	}
	c.Redirect(http.StatusFound, "/")
}

func (h *ViewHandler) ShowPlant(c *gin.Context) {
	plant, err := h.plants.Get(middleware.UserID(c), c.Param("id"))
	if err != nil {
		c.HTML(http.StatusNotFound, "error.html", gin.H{"Title": "找不到植物", "Error": err.Error()})
		return
	}
	c.HTML(http.StatusOK, "plant_detail.html", gin.H{"Title": plant.Name, "Plant": plant})
}

func (h *ViewHandler) EditPlant(c *gin.Context) {
	plant, err := h.plants.Get(middleware.UserID(c), c.Param("id"))
	if err != nil {
		c.HTML(http.StatusNotFound, "error.html", gin.H{"Title": "找不到植物", "Error": err.Error()})
		return
	}
	c.HTML(http.StatusOK, "plant_form.html", gin.H{
		"Title": "编辑植物", "Action": "/plants/" + plant.ID + "/update",
		"Submit": "更新档案", "Plant": plant,
	})
}

func (h *ViewHandler) UpdatePlant(c *gin.Context) {
	existing, err := h.plants.Get(middleware.UserID(c), c.Param("id"))
	if err != nil {
		c.HTML(http.StatusNotFound, "error.html", gin.H{"Title": "找不到植物", "Error": err.Error()})
		return
	}
	avatarURL, err := h.saveFile(c, "avatar")
	if err != nil {
		h.formError(c, "编辑植物", "/plants/"+c.Param("id")+"/edit", err)
		return
	}
	if err := h.plants.Update(middleware.UserID(c), c.Param("id"), plantInput(c), avatarURL); err != nil {
		_ = utils.DeleteUpload(avatarURL, h.uploadDir)
		h.formError(c, "编辑植物", "/plants/"+c.Param("id")+"/edit", err)
		return
	}
	if avatarURL != "" && existing.AvatarURL != "" {
		_ = utils.DeleteUpload(existing.AvatarURL, h.uploadDir)
	}
	c.Redirect(http.StatusFound, "/plants/"+c.Param("id"))
}

func (h *ViewHandler) DeletePlant(c *gin.Context) {
	plant, err := h.plants.Get(middleware.UserID(c), c.Param("id"))
	if err != nil {
		c.HTML(http.StatusNotFound, "error.html", gin.H{"Title": "删除失败", "Error": err.Error()})
		return
	}
	if err := h.plants.Delete(middleware.UserID(c), c.Param("id")); err != nil {
		c.HTML(http.StatusNotFound, "error.html", gin.H{"Title": "删除失败", "Error": err.Error()})
		return
	}
	_ = utils.DeleteUpload(plant.AvatarURL, h.uploadDir)
	for _, log := range plant.CareLogs {
		_ = utils.DeleteUpload(log.PhotoURL, h.uploadDir)
	}
	c.Redirect(http.StatusFound, "/")
}

func (h *ViewHandler) AddCare(c *gin.Context) {
	photoURL, err := h.saveFile(c, "photo")
	if err != nil {
		c.HTML(http.StatusBadRequest, "error.html", gin.H{"Title": "记录失败", "Error": err.Error()})
		return
	}
	input := dto.CareInput{
		Type: c.PostForm("type"), Note: c.PostForm("note"),
		WaterCycle:      parseInt(c.PostForm("water_cycle")),
		FertilizerCycle: parseInt(c.PostForm("fertilizer_cycle")),
	}
	err = h.plants.AddCare(middleware.UserID(c), c.Param("id"), input, photoURL)
	if err != nil {
		// The record wasn't saved, so the uploaded photo would otherwise be
		// left on disk with nothing referencing it. Only clean up on failure;
		// on success the file is owned by the new care_log row.
		if photoURL != "" {
			_ = utils.DeleteUpload(photoURL, h.uploadDir)
		}
		c.HTML(http.StatusBadRequest, "error.html", gin.H{"Title": "记录失败", "Error": err.Error()})
		return
	}
	c.Redirect(http.StatusFound, "/plants/"+c.Param("id"))
}

func (h *ViewHandler) SetCycle(c *gin.Context) {
	input := dto.CycleInput{
		Type:         c.PostForm("type"),
		IntervalDays: parseInt(c.PostForm("interval_days")),
		LastDate:     c.PostForm("last_date"),
	}
	if _, err := h.plants.SetCycle(middleware.UserID(c), c.Param("id"), input); err != nil {
		c.HTML(http.StatusBadRequest, "error.html", gin.H{"Title": "周期保存失败", "Error": err.Error()})
		return
	}
	c.Redirect(http.StatusFound, "/plants/"+c.Param("id"))
}

func (h *ViewHandler) DeleteCycle(c *gin.Context) {
	if err := h.plants.DeleteCycle(middleware.UserID(c), c.Param("id"), c.Param("type")); err != nil {
		c.HTML(http.StatusBadRequest, "error.html", gin.H{"Title": "周期删除失败", "Error": err.Error()})
		return
	}
	c.Redirect(http.StatusFound, "/plants/"+c.Param("id"))
}

func plantInput(c *gin.Context) dto.PlantInput {
	date := c.PostForm("acquired_date")
	if strings.TrimSpace(date) == "" {
		date = utils.FormatDate(utils.Today())
	}
	return dto.PlantInput{
		Name: c.PostForm("name"), Species: c.PostForm("species"),
		Source: c.PostForm("source"), AcquiredDate: date,
		Location: c.PostForm("location"), Status: c.PostForm("status"),
		Difficulty: c.PostForm("difficulty"),
	}
}

func (h *ViewHandler) formError(c *gin.Context, title, action string, err error) {
	c.HTML(http.StatusBadRequest, "plant_form.html", gin.H{
		"Title": title, "Action": action, "Submit": "保存植物",
		"Plant": plantFromInput(plantInput(c)), "Error": err.Error(),
	})
}

func plantFromInput(input dto.PlantInput) model.Plant {
	date, _ := utils.ParseDate(input.AcquiredDate)
	return model.Plant{Name: input.Name, Species: input.Species, Source: input.Source, AcquiredDate: date, Location: input.Location, Status: input.Status, Difficulty: input.Difficulty}
}

func (h *ViewHandler) saveFile(c *gin.Context, field string) (string, error) {
	file, header, err := c.Request.FormFile(field)
	if errors.Is(err, http.ErrMissingFile) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return utils.SaveUpload(file, header, h.uploadDir)
}

func parseInt(value string) int {
	parsed, _ := strconv.Atoi(value)
	return parsed
}
