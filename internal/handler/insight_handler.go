package handler

import (
	"encoding/csv"
	"fmt"
	"net/http"
	"time"

	"plant-diary/api/dto"
	"plant-diary/internal/middleware"
	"plant-diary/internal/model"
	"plant-diary/internal/service"
	"plant-diary/pkg/utils"

	"github.com/gin-gonic/gin"
)

type InsightHandler struct {
	insights *service.InsightService
	plants   *service.PlantService
}

type InsightPageData struct {
	Title  string
	Report service.InsightReport
	Plants []model.Plant
	Query  dto.InsightQuery
	Error  string
}

func NewInsightHandler(insights *service.InsightService, plants *service.PlantService) *InsightHandler {
	return &InsightHandler{insights: insights, plants: plants}
}

func (h *InsightHandler) Show(c *gin.Context) {
	query := insightQuery(c)
	report, err := h.analyze(c, query)
	if err != nil {
		c.HTML(http.StatusBadRequest, "insights.html", InsightPageData{Title: "成长分析", Query: query, Error: err.Error()})
		return
	}
	plants, err := h.plants.List(middleware.UserID(c), "", "")
	if err != nil {
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{"Title": "分析失败", "Error": err.Error()})
		return
	}
	c.HTML(http.StatusOK, "insights.html", InsightPageData{
		Title: "成长分析", Report: report, Plants: plants, Query: query,
	})
}

func (h *InsightHandler) ExportCSV(c *gin.Context) {
	report, err := h.analyze(c, insightQuery(c))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	filename := fmt.Sprintf("plant-care-%s-%s.csv", report.From, report.To)
	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Disposition", `attachment; filename="`+filename+`"`)
	c.Writer.WriteHeader(http.StatusOK)

	writer := csv.NewWriter(c.Writer)
	_, _ = c.Writer.Write([]byte{0xef, 0xbb, 0xbf})
	_ = writer.Write([]string{"日期", "植物", "操作类型", "备注", "照片地址"})
	for _, record := range report.Records {
		_ = writer.Write([]string{
			record.CreatedAt.Format("2006-01-02 15:04"),
			record.PlantName,
			record.TypeLabel(),
			record.Note,
			record.PhotoURL,
		})
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		c.Error(err)
	}
}

func (h *InsightHandler) API(c *gin.Context) {
	report, err := h.analyze(c, insightQuery(c))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, report)
}

func (h *InsightHandler) analyze(c *gin.Context, query dto.InsightQuery) (service.InsightReport, error) {
	from, err := parseInsightDate(query.From)
	if err != nil {
		return service.InsightReport{}, errorsWithField("起始日期", err)
	}
	to, err := parseInsightDate(query.To)
	if err != nil {
		return service.InsightReport{}, errorsWithField("结束日期", err)
	}
	return h.insights.Analyze(middleware.UserID(c), service.InsightFilter{
		From: from, To: to, PlantID: query.PlantID, CareType: query.CareType,
	})
}

func insightQuery(c *gin.Context) dto.InsightQuery {
	return dto.InsightQuery{
		From: c.Query("from"), To: c.Query("to"),
		PlantID: c.Query("plant_id"), CareType: c.Query("care_type"),
	}
}

func parseInsightDate(value string) (time.Time, error) {
	if value == "" {
		return time.Time{}, nil
	}
	return utils.ParseDate(value)
}

func errorsWithField(field string, err error) error {
	return fmt.Errorf("%s格式不正确: %w", field, err)
}
