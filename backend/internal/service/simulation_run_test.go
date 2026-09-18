package service

import (
	"encoding/json"
	"reflect"
	"testing"

	"gorm.io/datatypes"

	"mine-ventilation-network-simulator/backend/internal/constants"
	"mine-ventilation-network-simulator/backend/internal/dto"
	"mine-ventilation-network-simulator/backend/internal/model"
)

func TestSolveNetworkIsDeterministic(t *testing.T) {
	nodes := []model.VentilationNode{
		{ID: 1, Code: "IN", NodeType: string(constants.NodeTypeIntake), Status: string(constants.NodeStatusActive), PressurePa: 1000},
		{ID: 2, Code: "WF", NodeType: string(constants.NodeTypeWorkface), Status: string(constants.NodeStatusActive), RequiredAirflowM3S: 8, PressurePa: 500},
		{ID: 3, Code: "OUT", NodeType: string(constants.NodeTypeExhaust), Status: string(constants.NodeStatusActive), PressurePa: 0},
	}
	edges := []model.AirwayEdge{
		{ID: 1, Code: "E1", FromNodeID: 1, ToNodeID: 2, ResistanceNS2M8: 2, AreaM2: 5, MaxVelocityMS: 5, DoorState: string(constants.DoorStateOpen), Enabled: true, CriticalPath: true},
		{ID: 2, Code: "E2", FromNodeID: 2, ToNodeID: 3, ResistanceNS2M8: 2, AreaM2: 5, MaxVelocityMS: 5, DoorState: string(constants.DoorStateOpen), Enabled: true, CriticalPath: true},
	}
	curve, _ := json.Marshal([]dto.FanCurvePoint{{FlowM3S: 0, PressurePa: 900}, {FlowM3S: 30, PressurePa: 600}, {FlowM3S: 60, PressurePa: 300}})
	scenario := model.FanScenario{ID: 1, FanCurveJSON: datatypes.JSON(curve), SolverTolerance: 0.02, MaxIterations: 120}
	first := solveNetwork(scenario, nodes, edges)
	second := solveNetwork(scenario, nodes, edges)
	if !reflect.DeepEqual(first, second) {
		t.Fatal("solver returned different results for identical input")
	}
	if first.Status != constants.SimulationStatusConverged {
		t.Fatalf("expected convergence, got %s residual %.6f", first.Status, first.Residual)
	}
	if first.Iterations == 0 || len(first.Residuals) != first.Iterations {
		t.Fatalf("residual history mismatch: iterations=%d history=%d", first.Iterations, len(first.Residuals))
	}
}

func TestValidateDirectedNetworkFindsUnreachableWorkface(t *testing.T) {
	nodes := []model.VentilationNode{
		{ID: 1, Code: "IN", NodeType: string(constants.NodeTypeIntake), Status: string(constants.NodeStatusActive)},
		{ID: 2, Code: "OUT", NodeType: string(constants.NodeTypeExhaust), Status: string(constants.NodeStatusActive)},
		{ID: 3, Code: "WF", NodeType: string(constants.NodeTypeWorkface), Status: string(constants.NodeStatusActive)},
	}
	edges := []model.AirwayEdge{{ID: 1, FromNodeID: 1, ToNodeID: 2, Enabled: true}}
	result := validateDirectedNetwork(nodes, edges)
	if result.Valid {
		t.Fatal("expected invalid network")
	}
	found := false
	for _, issue := range result.Issues {
		if issue.Code == "UNREACHABLE_WORKFACE" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected unreachable issue, got %#v", result.Issues)
	}
}

func TestEvaluateRisksCoversVelocityReverseDemandAndDisconnected(t *testing.T) {
	nodes := []model.VentilationNode{{ID: 2, NodeType: string(constants.NodeTypeWorkface), RequiredAirflowM3S: 20}}
	edges := []model.AirwayEdge{
		{ID: 1, FromNodeID: 1, ToNodeID: 2, AreaM2: 1, MaxVelocityMS: 2, Enabled: true, DoorState: string(constants.DoorStateOpen)},
		{ID: 2, FromNodeID: 2, ToNodeID: 3, AreaM2: 4, MaxVelocityMS: 8, Enabled: true, DoorState: string(constants.DoorStateOpen)},
		{ID: 3, FromNodeID: 2, ToNodeID: 4, AreaM2: 4, MaxVelocityMS: 8, Enabled: false, DoorState: string(constants.DoorStateClosed), CriticalPath: true},
	}
	risks := evaluateRisks(nodes, edges, map[uint]float64{1: 5, 2: -1, 3: 0})
	codes := map[string]bool{}
	for _, risk := range risks {
		codes[risk.RuleCode] = true
	}
	for _, code := range []string{constants.RiskRuleVelocity, constants.RiskRuleReverseFlow, constants.RiskRuleDemandGap, constants.RiskRuleDisconnected} {
		if !codes[code] {
			t.Errorf("missing risk rule %s in %#v", code, risks)
		}
	}
}
