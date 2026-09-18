package dto

type FanCurvePoint struct {
	FlowM3S    float64 `json:"flow_m3s" binding:"gte=0,lte=5000"`
	PressurePa float64 `json:"pressure_pa" binding:"gte=0,lte=200000"`
}

type CreateFanScenarioRequest struct {
	Name            string          `json:"name" binding:"required,min=2,max=120"`
	Description     string          `json:"description" binding:"required,min=4,max=600"`
	FanCurve        []FanCurvePoint `json:"fan_curve" binding:"required,min=2,max=20,dive"`
	OperatingMode   string          `json:"operating_mode" binding:"required,oneof=normal reduced emergency_test"`
	SolverTolerance float64         `json:"solver_tolerance" binding:"required,gt=0,lte=10"`
	MaxIterations   int             `json:"max_iterations" binding:"required,gte=10,lte=500"`
}

type TransitionScenarioRequest struct {
	TargetStatus string `json:"target_status" binding:"required"`
	Reason       string `json:"reason" binding:"omitempty,max=400"`
	Version      uint   `json:"version" binding:"required,gte=1"`
}

type ScenarioListQuery struct {
	Page     int    `form:"page" binding:"omitempty,gte=1"`
	PageSize int    `form:"page_size" binding:"omitempty,gte=1,lte=100"`
	Status   string `form:"status"`
	Search   string `form:"search"`
}
