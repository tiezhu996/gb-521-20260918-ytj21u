package dto

type CreateVentilationNodeRequest struct {
	Code               string  `json:"code" binding:"required,min=2,max=48"`
	NodeType           string  `json:"node_type" binding:"required"`
	ElevationM         float64 `json:"elevation_m" binding:"gte=-2000,lte=9000"`
	RequiredAirflowM3S float64 `json:"required_airflow_m3s" binding:"gte=0,lte=1000"`
	PressurePa         float64 `json:"pressure_pa" binding:"gte=-100000,lte=100000"`
	Status             string  `json:"status" binding:"required"`
}

type UpdateVentilationNodeRequest struct {
	NodeType           string  `json:"node_type" binding:"required"`
	ElevationM         float64 `json:"elevation_m" binding:"gte=-2000,lte=9000"`
	RequiredAirflowM3S float64 `json:"required_airflow_m3s" binding:"gte=0,lte=1000"`
	PressurePa         float64 `json:"pressure_pa" binding:"gte=-100000,lte=100000"`
	Status             string  `json:"status" binding:"required"`
}

type NodeListQuery struct {
	Page     int    `form:"page" binding:"omitempty,gte=1"`
	PageSize int    `form:"page_size" binding:"omitempty,gte=1,lte=100"`
	Type     string `form:"type"`
	Status   string `form:"status"`
	Search   string `form:"search"`
}
