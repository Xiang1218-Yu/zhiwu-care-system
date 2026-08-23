package model

import "time"

type User struct {
	ID           string `gorm:"primaryKey;size:36"`
	Email        string `gorm:"uniqueIndex;size:120;not null"`
	PasswordHash string `gorm:"size:255;not null"`
	Name         string `gorm:"size:80;not null"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
	Plants       []Plant `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`
}
