package service

import (
	"time"

	"plant-diary/internal/repository"
)

type Stats struct {
	TotalPlants      int64
	Healthy          int64
	Yellowing        int64
	Pests            int64
	Gone             int64
	Dead             int64
	RecentCareLogs   int64
	MonthlyNewPlants int64
}

type StatsService struct {
	plants *repository.PlantRepository
	care   *repository.CareRepository
}

func NewStatsService(plants *repository.PlantRepository, care *repository.CareRepository) *StatsService {
	return &StatsService{plants: plants, care: care}
}

func (s *StatsService) Get(userID string) (Stats, error) {
	counts, err := s.plants.CountByStatus(userID)
	if err != nil {
		return Stats{}, err
	}
	stats := Stats{
		Healthy: counts["healthy"], Yellowing: counts["yellowing"],
		Pests: counts["pests"], Gone: counts["gone"], Dead: counts["dead"],
	}
	stats.TotalPlants = stats.Healthy + stats.Yellowing + stats.Pests + stats.Gone + stats.Dead
	now := time.Now()
	stats.RecentCareLogs, err = s.care.CountLogsAfter(userID, now.AddDate(0, 0, -7))
	if err != nil {
		return Stats{}, err
	}
	stats.MonthlyNewPlants, err = s.plants.CountCreatedAfter(userID, time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location()))
	return stats, err
}
