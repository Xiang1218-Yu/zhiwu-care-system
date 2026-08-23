package model

import "time"

const (
	CareWater      = "water"
	CareFertilizer = "fertilizer"
	CareRepot      = "repot"
	CarePrune      = "prune"
	CareSpray      = "spray"
	CareClean      = "clean"
)

type CareLog struct {
	ID        string `gorm:"primaryKey;size:36"`
	PlantID   string `gorm:"index;size:36;not null"`
	Type      string `gorm:"size:20;not null"`
	Note      string `gorm:"type:text"`
	PhotoURL  string `gorm:"size:255"`
	CreatedAt time.Time
}

func (l CareLog) TypeLabel() string {
	return map[string]string{
		CareWater:      "浇水",
		CareFertilizer: "施肥",
		CareRepot:      "换盆",
		CarePrune:      "修剪",
		CareSpray:      "喷药",
		CareClean:      "擦拭叶片",
	}[l.Type]
}
