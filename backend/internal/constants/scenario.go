package constants

type ScenarioStatus string

const (
	ScenarioStatusDraft         ScenarioStatus = "draft"
	ScenarioStatusPendingReview ScenarioStatus = "pending_review"
	ScenarioStatusApproved      ScenarioStatus = "approved"
	ScenarioStatusArchived      ScenarioStatus = "archived"
)

type Role string

const (
	RoleEngineer Role = "engineer"
	RoleReviewer Role = "reviewer"
	RoleAdmin    Role = "admin"
)

func ValidScenarioStatus(value string) bool {
	switch ScenarioStatus(value) {
	case ScenarioStatusDraft, ScenarioStatusPendingReview, ScenarioStatusApproved, ScenarioStatusArchived:
		return true
	default:
		return false
	}
}

func CanTransitionScenario(from, to ScenarioStatus) bool {
	switch from {
	case ScenarioStatusDraft:
		return to == ScenarioStatusPendingReview
	case ScenarioStatusPendingReview:
		return to == ScenarioStatusApproved || to == ScenarioStatusDraft
	case ScenarioStatusApproved:
		return to == ScenarioStatusArchived
	default:
		return false
	}
}

func ValidRole(value string) bool {
	switch Role(value) {
	case RoleEngineer, RoleReviewer, RoleAdmin:
		return true
	default:
		return false
	}
}
