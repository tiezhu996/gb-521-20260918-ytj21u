package dto

type CreateAirwayEdgeRequest struct {
	Code            string  `json:"code" binding:"required,min=2,max=48"`
	FromNodeID      uint    `json:"from_node_id" binding:"required"`
	ToNodeID        uint    `json:"to_node_id" binding:"required"`
	ResistanceNS2M8 float64 `json:"resistance_ns2m8" binding:"required,gt=0,lte=10000"`
	AreaM2          float64 `json:"area_m2" binding:"required,gt=0,lte=1000"`
	MaxVelocityMS   float64 `json:"max_velocity_ms" binding:"required,gt=0,lte=100"`
	DoorState       string  `json:"door_state" binding:"required"`
	Enabled         *bool   `json:"enabled" binding:"required"`
	CriticalPath    bool    `json:"critical_path"`
}

type UpdateAirwayEdgeRequest struct {
	ResistanceNS2M8 float64 `json:"resistance_ns2m8" binding:"required,gt=0,lte=10000"`
	AreaM2          float64 `json:"area_m2" binding:"required,gt=0,lte=1000"`
	MaxVelocityMS   float64 `json:"max_velocity_ms" binding:"required,gt=0,lte=100"`
	DoorState       string  `json:"door_state" binding:"required"`
	Enabled         *bool   `json:"enabled" binding:"required"`
	CriticalPath    bool    `json:"critical_path"`
	Version         uint    `json:"version" binding:"required,gte=1"`
}

type EdgeListQuery struct {
	Page     int    `form:"page" binding:"omitempty,gte=1"`
	PageSize int    `form:"page_size" binding:"omitempty,gte=1,lte=100"`
	Enabled  string `form:"enabled"`
	Search   string `form:"search"`
}
