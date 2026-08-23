package service

import (
	"testing"
	"time"

	"plant-diary/internal/model"
	"plant-diary/internal/repository"
)

func TestNormalizeInsightFilterUsesThirtyDayWindow(t *testing.T) {
	filter, err := normalizeInsightFilter(InsightFilter{})
	if err != nil {
		t.Fatal(err)
	}
	if filter.To.Sub(filter.From).Hours()/24 != 29 {
		t.Fatalf("expected 30 calendar days, got %v", filter.To.Sub(filter.From))
	}
}

func TestNormalizeInsightFilterRejectsLargeRange(t *testing.T) {
	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.Local)
	to := from.AddDate(0, 0, 365)
	if _, err := normalizeInsightFilter(InsightFilter{From: from, To: to}); err == nil {
		t.Fatal("expected large range validation error")
	}
}

func TestCareStreakCountsConsecutiveCalendarDays(t *testing.T) {
	records := []repository.CareRecord{
		{CreatedAt: time.Date(2026, 8, 1, 9, 0, 0, 0, time.Local)},
		{CreatedAt: time.Date(2026, 8, 1, 18, 0, 0, 0, time.Local)},
		{CreatedAt: time.Date(2026, 8, 2, 9, 0, 0, 0, time.Local)},
		{CreatedAt: time.Date(2026, 8, 4, 9, 0, 0, 0, time.Local)},
		{CreatedAt: time.Date(2026, 8, 5, 9, 0, 0, 0, time.Local)},
		{CreatedAt: time.Date(2026, 8, 6, 9, 0, 0, 0, time.Local)},
	}
	if got := careStreak(records); got != 3 {
		t.Fatalf("expected streak 3, got %d", got)
	}
}

func TestPlantScoreRecognizesOverdueCycle(t *testing.T) {
	plant := model.Plant{Status: model.StatusHealthy}
	cycles := []model.CareCycle{{
		Type:     model.CareWater,
		NextDate: time.Now().AddDate(0, 0, -1),
	}}
	score, level, className := plantScore(plant, nil, cycles)
	if score >= 85 || level == "" || className == "" {
		t.Fatalf("expected overdue cycle to reduce score, got %d/%s/%s", score, level, className)
	}
}

func TestBuildTypeInsightsKeepsAllSupportedTypes(t *testing.T) {
	insights := buildTypeInsights(map[string]int{model.CareWater: 3}, 3)
	if len(insights) != 6 {
		t.Fatalf("expected 6 care types, got %d", len(insights))
	}
	if insights[0].Type != model.CareWater || insights[0].Percent != 100 {
		t.Fatalf("expected water to be first at 100%%, got %#v", insights[0])
	}
}
