package service

import (
	"time"

	"plant-diary/internal/model"
	"plant-diary/internal/repository"
	"plant-diary/pkg/utils"
)

type ReminderService struct {
	care *repository.CareRepository
}

type Reminder struct {
	PlantName string
	Type      string
	TypeLabel string
	NextDate  time.Time
	Days      int
}

func NewReminderService(care *repository.CareRepository) *ReminderService {
	return &ReminderService{care: care}
}

func (s *ReminderService) Due(userID string) ([]Reminder, error) {
	cycles, err := s.care.ListDueCycles(userID, utils.Today())
	if err != nil {
		return nil, err
	}
	reminders := make([]Reminder, 0, len(cycles))
	for _, cycle := range cycles {
		reminders = append(reminders, Reminder{
			PlantName: cycle.Plant.Name, Type: cycle.Type,
			TypeLabel: careTypeLabel(cycle.Type), NextDate: cycle.NextDate,
			Days: utils.DaysFromToday(cycle.NextDate),
		})
	}
	return reminders, nil
}

func careTypeLabel(value string) string {
	if value == model.CareWater {
		return "浇水"
	}
	return "施肥"
}
