package model

import (
	"time"

	"gorm.io/gorm"
)

type CareCycle struct {
	ID           string    `gorm:"primaryKey;size:36"`
	PlantID      string    `gorm:"uniqueIndex:plant_cycle_type;size:36;not null"`
	Type         string    `gorm:"uniqueIndex:plant_cycle_type;size:20;not null"`
	IntervalDays int       `gorm:"not null"`
	LastDate     time.Time `gorm:"type:date;not null"`
	NextDate     time.Time `gorm:"type:date;not null;index"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
	Plant        Plant `gorm:"foreignKey:PlantID"`
}

func (c *CareCycle) BeforeCreate(tx *gorm.DB) error {
	if c.ID == "" {
		c.ID = newID()
	}
	return nil
}
