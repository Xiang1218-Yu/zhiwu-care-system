package model

import (
	"time"

	"gorm.io/gorm"
)

type CareCycle struct {
	ID           string    `gorm:"primaryKey;size:36"`
	// One cycle per (plant, type): the unique index backs the upsert so a
	// competing update can never leave a second, stale row that would surface
	// as a duplicate reminder.
	PlantID      string    `gorm:"size:36;not null;uniqueIndex:idx_care_cycles_plant_type"`
	Type         string    `gorm:"size:20;not null;uniqueIndex:idx_care_cycles_plant_type"`
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
