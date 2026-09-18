package dto

type StartSimulationRequest struct {
	ScenarioID uint `json:"scenario_id" binding:"required"`
}

type ConfirmRisksRequest struct {
	Note string `json:"note" binding:"required,min=4,max=500"`
}

type SimulationListQuery struct {
	Page       int    `form:"page" binding:"omitempty,gte=1"`
	PageSize   int    `form:"page_size" binding:"omitempty,gte=1,lte=100"`
	Status     string `form:"status"`
	ScenarioID uint   `form:"scenario_id"`
}

type NetworkIssue struct {
	Code       string `json:"code"`
	EntityType string `json:"entity_type"`
	EntityID   uint   `json:"entity_id"`
	Message    string `json:"message"`
	Severity   string `json:"severity"`
}

type RiskEvidence struct {
	RuleCode    string  `json:"rule_code"`
	Level       string  `json:"level"`
	EntityType  string  `json:"entity_type"`
	EntityID    uint    `json:"entity_id"`
	Evidence    float64 `json:"evidence"`
	Threshold   float64 `json:"threshold"`
	Unit        string  `json:"unit"`
	Description string  `json:"description"`
}
