package service

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"plant-diary/api/dto"
	"plant-diary/internal/model"
	"plant-diary/internal/repository"
	"plant-diary/pkg/utils"

	"github.com/google/uuid"
)

type PlantService struct {
	plants *repository.PlantRepository
	care   *repository.CareRepository
}

func NewPlantService(plants *repository.PlantRepository, care *repository.CareRepository) *PlantService {
	return &PlantService{plants: plants, care: care}
}

func (s *PlantService) List(userID, status, location string) ([]model.Plant, error) {
	return s.plants.ListByUser(userID, status, location)
}

func (s *PlantService) Get(userID, id string) (*model.Plant, error) {
	plant, err := s.plants.FindOwned(id, userID)
	if err != nil {
		return nil, err
	}
	if plant == nil {
		return nil, errors.New("植物不存在")
	}
	return plant, nil
}

func (s *PlantService) Create(userID string, input dto.PlantInput, avatarURL string) (*model.Plant, error) {
	acquiredDate, err := utils.ParseDate(input.AcquiredDate)
	if err != nil {
		return nil, errors.New("入手日期格式不正确")
	}
	if strings.TrimSpace(input.Name) == "" || strings.TrimSpace(input.Species) == "" {
		return nil, errors.New("昵称和品种不能为空")
	}
	if !validPlantValues(input) {
		return nil, errors.New("植物档案包含不支持的选项")
	}
	plant := &model.Plant{
		ID: uuid.NewString(), UserID: userID,
		Name: strings.TrimSpace(input.Name), Species: strings.TrimSpace(input.Species),
		Source: input.Source, AcquiredDate: acquiredDate, Location: input.Location,
		Status: input.Status, Difficulty: input.Difficulty, AvatarURL: avatarURL,
	}
	if err := s.plants.Create(plant); err != nil {
		return nil, fmt.Errorf("创建植物: %w", err)
	}
	return plant, nil
}

func (s *PlantService) Update(userID, id string, input dto.PlantInput, avatarURL string) error {
	plant, err := s.Get(userID, id)
	if err != nil {
		return err
	}
	acquiredDate, err := utils.ParseDate(input.AcquiredDate)
	if err != nil {
		return errors.New("入手日期格式不正确")
	}
	if strings.TrimSpace(input.Name) == "" || strings.TrimSpace(input.Species) == "" || !validPlantValues(input) {
		return errors.New("植物档案不完整或包含不支持的选项")
	}
	plant.Name, plant.Species = strings.TrimSpace(input.Name), strings.TrimSpace(input.Species)
	plant.Source, plant.AcquiredDate, plant.Location = input.Source, acquiredDate, input.Location
	plant.Status, plant.Difficulty = input.Status, input.Difficulty
	if avatarURL != "" {
		plant.AvatarURL = avatarURL
	}
	return s.plants.Save(plant)
}

func (s *PlantService) Delete(userID, id string) error {
	if _, err := s.Get(userID, id); err != nil {
		return err
	}
	return s.plants.Delete(id, userID)
}

func (s *PlantService) AddCare(userID, plantID string, input dto.CareInput, photoURL string) error {
	if _, err := s.Get(userID, plantID); err != nil {
		return err
	}
	if !validCareType(input.Type) {
		return errors.New("不支持的养护类型")
	}
	switch input.Type {
	case model.CareWater:
		if !cycleWithinRange(input.WaterCycle) {
			return errors.New("养护周期需在 1 到 365 天之间")
		}
	case model.CareFertilizer:
		if !cycleWithinRange(input.FertilizerCycle) {
			return errors.New("养护周期需在 1 到 365 天之间")
		}
	}
	log := &model.CareLog{
		ID: uuid.NewString(), PlantID: plantID, Type: input.Type,
		Note: strings.TrimSpace(input.Note), PhotoURL: photoURL,
	}
	today := utils.Today()
	return s.care.Transaction(func(tx *repository.CareRepository) error {
		if err := tx.CreateLog(log); err != nil {
			return fmt.Errorf("创建养护记录: %w", err)
		}
		switch input.Type {
		case model.CareWater:
			if input.WaterCycle > 0 {
				return s.saveCycle(tx, plantID, model.CareWater, input.WaterCycle, today)
			}
		case model.CareFertilizer:
			if input.FertilizerCycle > 0 {
				return s.saveCycle(tx, plantID, model.CareFertilizer, input.FertilizerCycle, today)
			}
		}
		return nil
	})
}

func cycleWithinRange(days int) bool {
	return days >= 0 && days <= 365
}

func (s *PlantService) SetCycle(userID, plantID string, input dto.CycleInput) (*model.CareCycle, error) {
	if _, err := s.Get(userID, plantID); err != nil {
		return nil, err
	}
	lastDate := utils.Today()
	if input.LastDate != "" {
		parsed, err := utils.ParseDate(input.LastDate)
		if err != nil {
			return nil, errors.New("周期起算日期格式不正确")
		}
		lastDate = parsed
	}
	cycle, err := newCycle(plantID, input.Type, input.IntervalDays, lastDate)
	if err != nil {
		return nil, err
	}
	if err := s.care.UpsertCycle(cycle); err != nil {
		return nil, fmt.Errorf("保存养护周期: %w", err)
	}
	return cycle, nil
}

func (s *PlantService) DeleteCycle(userID, plantID, cycleType string) error {
	if _, err := s.Get(userID, plantID); err != nil {
		return err
	}
	if !validCycleType(cycleType) {
		return errors.New("不支持的养护周期类型")
	}
	return s.care.DeleteCycle(plantID, cycleType)
}

func (s *PlantService) saveCycle(care *repository.CareRepository, plantID, cycleType string, interval int, lastDate time.Time) error {
	cycle, err := newCycle(plantID, cycleType, interval, lastDate)
	if err != nil {
		return err
	}
	return care.UpsertCycle(cycle)
}

func newCycle(plantID, cycleType string, interval int, lastDate time.Time) (*model.CareCycle, error) {
	if interval < 1 || interval > 365 {
		return nil, errors.New("养护周期需在 1 到 365 天之间")
	}
	if !validCycleType(cycleType) {
		return nil, errors.New("不支持的养护周期类型")
	}
	return &model.CareCycle{
		PlantID: plantID, Type: cycleType, IntervalDays: interval,
		LastDate: lastDate, NextDate: lastDate.AddDate(0, 0, interval),
	}, nil
}

func validPlantValues(input dto.PlantInput) bool {
	return contains([]string{"market", "online", "friend"}, input.Source) &&
		contains([]string{"balcony", "living-room", "bedroom", "study"}, input.Location) &&
		contains([]string{model.StatusHealthy, model.StatusYellowing, model.StatusPests, model.StatusGone, model.StatusDead}, input.Status) &&
		contains([]string{"easy", "medium", "hard"}, input.Difficulty)
}

func validCareType(value string) bool {
	return contains([]string{model.CareWater, model.CareFertilizer, model.CareRepot, model.CarePrune, model.CareSpray, model.CareClean}, value)
}

func validCycleType(value string) bool {
	return contains([]string{model.CareWater, model.CareFertilizer}, value)
}

func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
