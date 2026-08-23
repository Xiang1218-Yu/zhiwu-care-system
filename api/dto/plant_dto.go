package dto

type PlantInput struct {
	Name         string `json:"name" form:"name"`
	Species      string `json:"species" form:"species"`
	Source       string `json:"source" form:"source"`
	AcquiredDate string `json:"acquired_date" form:"acquired_date"`
	Location     string `json:"location" form:"location"`
	Status       string `json:"status" form:"status"`
	Difficulty   string `json:"difficulty" form:"difficulty"`
}

type CycleInput struct {
	Type         string `json:"type" form:"type"`
	IntervalDays int    `json:"interval_days" form:"interval_days"`
	LastDate     string `json:"last_date" form:"last_date"`
}
