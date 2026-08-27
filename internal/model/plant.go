package model

import "time"

const (
	StatusHealthy   = "healthy"
	StatusYellowing = "yellowing"
	StatusPests     = "pests"
	StatusGone      = "gone"
	StatusDead      = "dead"
)

type Plant struct {
	ID           string    `gorm:"primaryKey;size:36"`
	UserID       string    `gorm:"index;size:36;not null"`
	Name         string    `gorm:"size:100;not null"`
	Species      string    `gorm:"size:100;not null"`
	Source       string    `gorm:"size:50;not null"`
	AcquiredDate time.Time `gorm:"type:date;not null"`
	Location     string    `gorm:"size:50;not null"`
	Status       string    `gorm:"size:20;not null;default:healthy"`
	Difficulty   string    `gorm:"size:20;not null;default:easy"`
	AvatarURL    string    `gorm:"size:255"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
	CareLogs     []CareLog   `gorm:"foreignKey:PlantID;constraint:OnDelete:CASCADE"`
	CareCycles   []CareCycle `gorm:"foreignKey:PlantID;constraint:OnDelete:CASCADE"`
}

func (p Plant) StatusLabel() string {
	return map[string]string{
		StatusHealthy:   "健康",
		StatusYellowing: "黄叶",
		StatusPests:     "虫害",
		StatusGone:      "已送人",
		StatusDead:      "已枯萎",
	}[p.Status]
}

func (p Plant) DifficultyLabel() string {
	return map[string]string{"easy": "好养", "medium": "一般", "hard": "需要细心"}[p.Difficulty]
}

func (p Plant) SourceLabel() string {
	return map[string]string{"market": "花市", "online": "网购", "friend": "朋友送的"}[p.Source]
}

func (p Plant) LocationLabel() string {
	return map[string]string{"balcony": "阳台", "living-room": "客厅", "bedroom": "卧室", "study": "书房"}[p.Location]
}

func (p Plant) Cycle(cycleType string) *CareCycle {
	for index := range p.CareCycles {
		if p.CareCycles[index].Type == cycleType {
			return &p.CareCycles[index]
		}
	}
	return nil
}
