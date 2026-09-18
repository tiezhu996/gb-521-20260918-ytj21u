package service

import (
	"testing"

	"mine-ventilation-network-simulator/backend/internal/constants"
	"mine-ventilation-network-simulator/backend/internal/dto"
)

func TestValidateFanCurveRejectsRisingPressure(t *testing.T) {
	err := validateFanCurve([]dto.FanCurvePoint{{FlowM3S: 0, PressurePa: 100}, {FlowM3S: 10, PressurePa: 120}})
	if err == nil {
		t.Fatal("expected rising pressure curve to be rejected")
	}
}

func TestAuthorizeTransitionRequiresReviewerApproval(t *testing.T) {
	err := authorizeTransition(constants.ScenarioStatusPendingReview, constants.ScenarioStatusApproved, string(constants.RoleEngineer), "")
	if err == nil {
		t.Fatal("engineer must not approve a scenario")
	}
	if err := authorizeTransition(constants.ScenarioStatusPendingReview, constants.ScenarioStatusApproved, string(constants.RoleReviewer), ""); err != nil {
		t.Fatalf("reviewer should approve: %v", err)
	}
}

func TestAuthorizeTransitionRequiresRejectReason(t *testing.T) {
	err := authorizeTransition(constants.ScenarioStatusPendingReview, constants.ScenarioStatusDraft, string(constants.RoleReviewer), "no")
	if err == nil {
		t.Fatal("short reject reason should fail")
	}
}
