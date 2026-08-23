package service

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"plant-diary/internal/model"
	"plant-diary/internal/repository"
	"plant-diary/pkg/utils"
)

const maxInsightRangeDays = 365

type InsightFilter struct {
	From     time.Time
	To       time.Time
	PlantID  string
	CareType string
}

type TrendPoint struct {
	Date  string `json:"date"`
	Label string `json:"label"`
	Count int    `json:"count"`
}

type CareTypeInsight struct {
	Type    string `json:"type"`
	Label   string `json:"label"`
	Count   int    `json:"count"`
	Percent int    `json:"percent"`
}

type PlantInsight struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	Species        string `json:"species"`
	Status         string `json:"status"`
	StatusLabel    string `json:"status_label"`
	Score          int    `json:"score"`
	Level          string `json:"level"`
	LevelClass     string `json:"level_class"`
	TotalCare      int    `json:"total_care"`
	PhotoCount     int    `json:"photo_count"`
	Streak         int    `json:"streak"`
	LastCareDate   string `json:"last_care_date"`
	NextDueDate    string `json:"next_due_date"`
	NeedsAttention bool   `json:"needs_attention"`
	AttentionText  string `json:"attention_text"`
}

type GalleryItem struct {
	PlantID   string `json:"plant_id"`
	PlantName string `json:"plant_name"`
	Type      string `json:"type"`
	TypeLabel string `json:"type_label"`
	PhotoURL  string `json:"photo_url"`
	Note      string `json:"note"`
	Date      string `json:"date"`
}

type MonthlySummary struct {
	Month      string  `json:"month"`
	Label      string  `json:"label"`
	CareCount  int     `json:"care_count"`
	PhotoCount int     `json:"photo_count"`
	ActiveDays int     `json:"active_days"`
	Average    float64 `json:"average"`
}

type InsightRecommendation struct {
	PlantID   string `json:"plant_id"`
	PlantName string `json:"plant_name"`
	Priority  string `json:"priority"`
	Title     string `json:"title"`
	Message   string `json:"message"`
}

type InsightReport struct {
	From            string                  `json:"from"`
	To              string                  `json:"to"`
	PlantID         string                  `json:"plant_id"`
	CareType        string                  `json:"care_type"`
	TotalCare       int                     `json:"total_care"`
	ActiveDays      int                     `json:"active_days"`
	AveragePerDay   float64                 `json:"average_per_day"`
	BestDay         string                  `json:"best_day"`
	BestDayCount    int                     `json:"best_day_count"`
	TopPlant        string                  `json:"top_plant"`
	HealthAverage   int                     `json:"health_average"`
	Trend           []TrendPoint            `json:"trend"`
	CareTypes       []CareTypeInsight       `json:"care_types"`
	Plants          []PlantInsight          `json:"plants"`
	Gallery         []GalleryItem           `json:"gallery"`
	Monthly         []MonthlySummary        `json:"monthly"`
	Recommendations []InsightRecommendation `json:"recommendations"`
	Records         []repository.CareRecord `json:"records"`
}

type InsightService struct {
	plants  *repository.PlantRepository
	care    *repository.CareRepository
	insight *repository.InsightRepository
}

func NewInsightService(plants *repository.PlantRepository, care *repository.CareRepository, insight *repository.InsightRepository) *InsightService {
	return &InsightService{plants: plants, care: care, insight: insight}
}

func (s *InsightService) Analyze(userID string, filter InsightFilter) (InsightReport, error) {
	filter, err := normalizeInsightFilter(filter)
	if err != nil {
		return InsightReport{}, err
	}
	plants, err := s.plants.ListByUser(userID, "", "")
	if err != nil {
		return InsightReport{}, fmt.Errorf("读取植物档案: %w", err)
	}
	if filter.PlantID != "" {
		plant, err := s.insight.FindPlant(userID, filter.PlantID)
		if err != nil {
			return InsightReport{}, fmt.Errorf("读取筛选植物: %w", err)
		}
		if plant == nil {
			return InsightReport{}, errors.New("筛选的植物不存在")
		}
	}
	records, err := s.insight.ListCareRecords(userID, filter.From, filter.To, filter.PlantID, filter.CareType)
	if err != nil {
		return InsightReport{}, fmt.Errorf("读取养护记录: %w", err)
	}
	cycles, err := s.insight.ListCycles(userID)
	if err != nil {
		return InsightReport{}, fmt.Errorf("读取养护周期: %w", err)
	}

	plantMap := make(map[string]model.Plant, len(plants))
	for _, plant := range plants {
		if filter.PlantID == "" || plant.ID == filter.PlantID {
			plantMap[plant.ID] = plant
		}
	}
	recordsByPlant := groupRecordsByPlant(records)
	cyclesByPlant := groupCyclesByPlant(cycles, plantMap)
	dailyCounts := countByDay(records)
	typeCounts := countByType(records)

	report := InsightReport{
		From:          filter.From.Format("2006-01-02"),
		To:            filter.To.Format("2006-01-02"),
		PlantID:       filter.PlantID,
		CareType:      filter.CareType,
		TotalCare:     len(records),
		ActiveDays:    len(dailyCounts),
		AveragePerDay: averagePerDay(len(records), filter),
		Trend:         buildTrend(filter, dailyCounts),
		CareTypes:     buildTypeInsights(typeCounts, len(records)),
		Gallery:       buildGallery(records),
		Records:       records,
	}
	report.BestDay, report.BestDayCount = bestDay(dailyCounts)
	report.Plants = buildPlantInsights(plantMap, recordsByPlant, cyclesByPlant)
	report.HealthAverage = averageHealth(report.Plants)
	report.TopPlant = topPlant(report.Plants, recordsByPlant)
	report.Monthly = buildMonthlySummary(records, filter)
	report.Recommendations = buildRecommendations(report.Plants, recordsByPlant, cyclesByPlant)
	return report, nil
}

func normalizeInsightFilter(filter InsightFilter) (InsightFilter, error) {
	today := utils.Today()
	if filter.To.IsZero() {
		filter.To = today
	}
	if filter.From.IsZero() {
		filter.From = filter.To.AddDate(0, 0, -29)
	}
	filter = normalizeInsightDates(filter)
	if filter.From.After(filter.To) {
		return InsightFilter{}, errors.New("分析起始日期不能晚于结束日期")
	}
	if filter.To.Sub(filter.From).Hours()/24 >= maxInsightRangeDays {
		return InsightFilter{}, fmt.Errorf("分析范围不能超过 %d 天", maxInsightRangeDays)
	}
	if filter.CareType != "" && !validCareType(filter.CareType) {
		return InsightFilter{}, errors.New("不支持的分析操作类型")
	}
	return filter, nil
}

func normalizeInsightDates(filter InsightFilter) InsightFilter {
	filter.From = dateStart(filter.From)
	filter.To = dateStart(filter.To)
	return filter
}

func dateStart(value time.Time) time.Time {
	return time.Date(value.Year(), value.Month(), value.Day(), 0, 0, 0, 0, value.Location())
}

func averagePerDay(total int, filter InsightFilter) float64 {
	days := int(filter.To.Sub(filter.From).Hours()/24) + 1
	if days < 1 {
		return 0
	}
	return float64(total) / float64(days)
}

func groupRecordsByPlant(records []repository.CareRecord) map[string][]repository.CareRecord {
	grouped := make(map[string][]repository.CareRecord)
	for _, record := range records {
		grouped[record.PlantID] = append(grouped[record.PlantID], record)
	}
	return grouped
}

func groupCyclesByPlant(cycles []model.CareCycle, plants map[string]model.Plant) map[string][]model.CareCycle {
	grouped := make(map[string][]model.CareCycle)
	for _, cycle := range cycles {
		if _, ok := plants[cycle.PlantID]; ok {
			grouped[cycle.PlantID] = append(grouped[cycle.PlantID], cycle)
		}
	}
	return grouped
}

func countByDay(records []repository.CareRecord) map[string]int {
	counts := make(map[string]int)
	for _, record := range records {
		counts[record.CreatedAt.Format("2006-01-02")]++
	}
	return counts
}

func countByType(records []repository.CareRecord) map[string]int {
	counts := make(map[string]int)
	for _, record := range records {
		counts[record.Type]++
	}
	return counts
}

func buildTrend(filter InsightFilter, counts map[string]int) []TrendPoint {
	days := int(filter.To.Sub(filter.From).Hours()/24) + 1
	trend := make([]TrendPoint, 0, days)
	for index := 0; index < days; index++ {
		day := filter.From.AddDate(0, 0, index)
		date := day.Format("2006-01-02")
		trend = append(trend, TrendPoint{Date: date, Label: day.Format("01/02"), Count: counts[date]})
	}
	return trend
}

func buildTypeInsights(counts map[string]int, total int) []CareTypeInsight {
	types := []string{model.CareWater, model.CareFertilizer, model.CareRepot, model.CarePrune, model.CareSpray, model.CareClean}
	result := make([]CareTypeInsight, 0, len(types))
	for _, careType := range types {
		count := counts[careType]
		percent := 0
		if total > 0 {
			percent = count * 100 / total
		}
		result = append(result, CareTypeInsight{Type: careType, Label: careTypeLabel(careType), Count: count, Percent: percent})
	}
	sort.SliceStable(result, func(i, j int) bool {
		return result[i].Count > result[j].Count
	})
	return result
}

func buildGallery(records []repository.CareRecord) []GalleryItem {
	gallery := make([]GalleryItem, 0)
	for index := len(records) - 1; index >= 0; index-- {
		record := records[index]
		if strings.TrimSpace(record.PhotoURL) == "" {
			continue
		}
		gallery = append(gallery, GalleryItem{
			PlantID: record.PlantID, PlantName: record.PlantName,
			Type: record.Type, TypeLabel: careTypeLabel(record.Type),
			PhotoURL: record.PhotoURL, Note: record.Note,
			Date: record.CreatedAt.Format("2006-01-02"),
		})
	}
	return gallery
}

func buildPlantInsights(plants map[string]model.Plant, records map[string][]repository.CareRecord, cycles map[string][]model.CareCycle) []PlantInsight {
	result := make([]PlantInsight, 0, len(plants))
	for id, plant := range plants {
		plantRecords := records[id]
		plantCycles := cycles[id]
		insight := PlantInsight{
			ID: id, Name: plant.Name, Species: plant.Species,
			Status: plant.Status, StatusLabel: plant.StatusLabel(),
			TotalCare: len(plantRecords), PhotoCount: photoCount(plantRecords),
			Streak: careStreak(plantRecords),
		}
		if len(plantRecords) > 0 {
			insight.LastCareDate = plantRecords[len(plantRecords)-1].CreatedAt.Format("2006-01-02")
		}
		insight.NextDueDate = nextDueDate(plantCycles)
		insight.Score, insight.Level, insight.LevelClass = plantScore(plant, plantRecords, plantCycles)
		insight.NeedsAttention, insight.AttentionText = attentionFor(insight, plantCycles)
		result = append(result, insight)
	}
	sort.SliceStable(result, func(i, j int) bool {
		if result[i].NeedsAttention != result[j].NeedsAttention {
			return result[i].NeedsAttention
		}
		return result[i].Score < result[j].Score
	})
	return result
}

func photoCount(records []repository.CareRecord) int {
	count := 0
	for _, record := range records {
		if record.PhotoURL != "" {
			count++
		}
	}
	return count
}

func careStreak(records []repository.CareRecord) int {
	dates := make(map[string]struct{})
	for _, record := range records {
		dates[record.CreatedAt.Format("2006-01-02")] = struct{}{}
	}
	if len(dates) == 0 {
		return 0
	}
	ordered := make([]string, 0, len(dates))
	for date := range dates {
		ordered = append(ordered, date)
	}
	sort.Strings(ordered)
	best, current := 1, 1
	for index := 1; index < len(ordered); index++ {
		left, _ := time.Parse("2006-01-02", ordered[index-1])
		right, _ := time.Parse("2006-01-02", ordered[index])
		if right.Sub(left).Hours() == 24 {
			current++
			if current > best {
				best = current
			}
		} else {
			current = 1
		}
	}
	return best
}

func nextDueDate(cycles []model.CareCycle) string {
	var next time.Time
	for _, cycle := range cycles {
		if next.IsZero() || cycle.NextDate.Before(next) {
			next = cycle.NextDate
		}
	}
	if next.IsZero() {
		return ""
	}
	return next.Format("2006-01-02")
}

func plantScore(plant model.Plant, records []repository.CareRecord, cycles []model.CareCycle) (int, string, string) {
	score := 60
	switch plant.Status {
	case model.StatusHealthy:
		score += 25
	case model.StatusYellowing:
		score -= 8
	case model.StatusPests:
		score -= 18
	case model.StatusGone, model.StatusDead:
		score -= 30
	}
	score += minInt(len(records)*2, 12)
	score += minInt(photoCount(records)*2, 8)
	score += minInt(careStreak(records), 10)
	if len(records) == 0 {
		score -= 15
	}
	for _, cycle := range cycles {
		if !cycle.NextDate.After(utils.Today()) {
			score -= 8
		}
	}
	score = maxInt(0, minInt(score, 100))
	switch {
	case score >= 85:
		return score, "状态很好", "success"
	case score >= 70:
		return score, "保持稳定", "success"
	case score >= 50:
		return score, "需要关注", "warning"
	default:
		return score, "优先处理", "danger"
	}
}

func attentionFor(insight PlantInsight, cycles []model.CareCycle) (bool, string) {
	for _, cycle := range cycles {
		if !cycle.NextDate.After(utils.Today()) {
			return true, fmt.Sprintf("%s已到期", careTypeLabel(cycle.Type))
		}
	}
	if insight.Score < 50 {
		return true, "健康评分偏低"
	}
	if insight.TotalCare == 0 {
		return true, "当前周期没有养护记录"
	}
	return false, ""
}

func averageHealth(plants []PlantInsight) int {
	if len(plants) == 0 {
		return 0
	}
	total := 0
	for _, plant := range plants {
		total += plant.Score
	}
	return total / len(plants)
}

func topPlant(plants []PlantInsight, records map[string][]repository.CareRecord) string {
	var name string
	maxCount := 0
	for _, plant := range plants {
		if len(records[plant.ID]) > maxCount {
			name, maxCount = plant.Name, len(records[plant.ID])
		}
	}
	return name
}

func bestDay(counts map[string]int) (string, int) {
	var bestDate string
	bestCount := 0
	for date, count := range counts {
		if count > bestCount || (count == bestCount && date < bestDate) {
			bestDate, bestCount = date, count
		}
	}
	return bestDate, bestCount
}

func buildMonthlySummary(records []repository.CareRecord, filter InsightFilter) []MonthlySummary {
	type monthData struct {
		count  int
		photos int
		days   map[string]struct{}
	}
	months := make(map[string]*monthData)
	for _, record := range records {
		month := record.CreatedAt.Format("2006-01")
		if months[month] == nil {
			months[month] = &monthData{days: make(map[string]struct{})}
		}
		data := months[month]
		data.count++
		data.days[record.CreatedAt.Format("2006-01-02")] = struct{}{}
		if record.PhotoURL != "" {
			data.photos++
		}
	}
	result := make([]MonthlySummary, 0)
	month := time.Date(filter.From.Year(), filter.From.Month(), 1, 0, 0, 0, 0, filter.From.Location())
	last := time.Date(filter.To.Year(), filter.To.Month(), 1, 0, 0, 0, 0, filter.To.Location())
	for !month.After(last) {
		key := month.Format("2006-01")
		data := months[key]
		item := MonthlySummary{Month: key, Label: month.Format("2006年01月")}
		if data != nil {
			item.CareCount = data.count
			item.PhotoCount = data.photos
			item.ActiveDays = len(data.days)
			item.Average = float64(item.CareCount) / float64(daysInMonth(month))
		}
		result = append(result, item)
		month = month.AddDate(0, 1, 0)
	}
	return result
}

func daysInMonth(month time.Time) int {
	return month.AddDate(0, 1, 0).Add(-time.Nanosecond).Day()
}

func buildRecommendations(plants []PlantInsight, records map[string][]repository.CareRecord, cycles map[string][]model.CareCycle) []InsightRecommendation {
	recommendations := make([]InsightRecommendation, 0)
	for _, plant := range plants {
		plantRecords := records[plant.ID]
		plantCycles := cycles[plant.ID]
		if plant.NeedsAttention {
			recommendations = append(recommendations, InsightRecommendation{
				PlantID: plant.ID, PlantName: plant.Name, Priority: "high",
				Title:   "优先安排一次检查",
				Message: plant.AttentionText + "，建议查看叶片、土壤湿度并补充一条记录。",
			})
			continue
		}
		if len(plantRecords) == 0 {
			recommendations = append(recommendations, InsightRecommendation{
				PlantID: plant.ID, PlantName: plant.Name, Priority: "medium",
				Title:   "建立第一条成长记录",
				Message: "完成一次养护并拍照，后续才能形成连续的成长趋势。",
			})
			continue
		}
		if photoCount(plantRecords) == 0 {
			recommendations = append(recommendations, InsightRecommendation{
				PlantID: plant.ID, PlantName: plant.Name, Priority: "low",
				Title:   "补充一张成长照片",
				Message: "记录叶片或整体状态，方便下次对比变化。",
			})
			continue
		}
		if len(plantCycles) == 0 {
			recommendations = append(recommendations, InsightRecommendation{
				PlantID: plant.ID, PlantName: plant.Name, Priority: "low",
				Title:   "设置养护周期",
				Message: "为浇水或施肥设置周期，可以减少忘记养护的情况。",
			})
		}
	}
	sort.SliceStable(recommendations, func(i, j int) bool {
		return recommendationRank(recommendations[i].Priority) < recommendationRank(recommendations[j].Priority)
	})
	return recommendations
}

func recommendationRank(priority string) int {
	switch priority {
	case "high":
		return 1
	case "medium":
		return 2
	default:
		return 3
	}
}

func minInt(left, right int) int {
	if left < right {
		return left
	}
	return right
}

func maxInt(left, right int) int {
	if left > right {
		return left
	}
	return right
}
