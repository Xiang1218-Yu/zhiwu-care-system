package dto

type CareInput struct {
	Type            string `json:"type" form:"type"`
	Note            string `json:"note" form:"note"`
	WaterCycle      int    `json:"water_cycle" form:"water_cycle"`
	FertilizerCycle int    `json:"fertilizer_cycle" form:"fertilizer_cycle"`
}
