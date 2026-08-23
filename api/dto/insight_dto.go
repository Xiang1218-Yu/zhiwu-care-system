package dto

type InsightQuery struct {
	From     string `form:"from" json:"from"`
	To       string `form:"to" json:"to"`
	PlantID  string `form:"plant_id" json:"plant_id"`
	CareType string `form:"care_type" json:"care_type"`
}
