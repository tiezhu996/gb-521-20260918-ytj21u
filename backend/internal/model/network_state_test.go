package model

import "testing"

func fingerprintFixture() ([]VentilationNode, []AirwayEdge) {
	nodes := []VentilationNode{
		{ID: 1, Code: "INT-01", NodeType: "intake", ElevationM: 12, PressurePa: 1250, Status: "active"},
		{ID: 2, Code: "JCT-12", NodeType: "junction", ElevationM: -85, PressurePa: 680, Status: "active"},
		{ID: 3, Code: "WF-07", NodeType: "workface", ElevationM: -126, RequiredAirflowM3S: 18, PressurePa: 410, Status: "active"},
	}
	edges := []AirwayEdge{
		{ID: 1, Code: "AW-101", FromNodeID: 1, ToNodeID: 2, ResistanceNS2M8: 1.8, AreaM2: 8.2, MaxVelocityMS: 8, DoorState: "open", Enabled: true, CriticalPath: true, Version: 3},
		{ID: 2, Code: "AW-102", FromNodeID: 2, ToNodeID: 3, ResistanceNS2M8: 2.4, AreaM2: 6.4, MaxVelocityMS: 7, DoorState: "open", Enabled: true, Version: 1},
	}
	return nodes, edges
}

func TestNetworkFingerprintIsDeterministicAndOrderIndependent(t *testing.T) {
	nodes, edges := fingerprintFixture()
	first := ComputeNetworkFingerprint(nodes, edges)
	second := ComputeNetworkFingerprint(nodes, edges)
	if first != second {
		t.Fatalf("fingerprint not deterministic: %s vs %s", first, second)
	}
	reversedNodes := []VentilationNode{nodes[2], nodes[1], nodes[0]}
	reversedEdges := []AirwayEdge{edges[1], edges[0]}
	if got := ComputeNetworkFingerprint(reversedNodes, reversedEdges); got != first {
		t.Fatalf("fingerprint must not depend on input order: %s vs %s", got, first)
	}
	if len(first) != 64 {
		t.Fatalf("expected sha256 hex fingerprint, got %q", first)
	}
}

func TestNetworkFingerprintIgnoresTimestampsAndRowVersion(t *testing.T) {
	nodes, edges := fingerprintFixture()
	base := ComputeNetworkFingerprint(nodes, edges)
	edges[1].Version = 99
	if got := ComputeNetworkFingerprint(nodes, edges); got != base {
		t.Fatal("optimistic-lock version bump must not change the fingerprint")
	}
}

func TestNetworkFingerprintCapturesAddDisableAndParamChange(t *testing.T) {
	nodes, edges := fingerprintFixture()
	base := ComputeNetworkFingerprint(nodes, edges)

	cases := map[string]func([]VentilationNode, []AirwayEdge) ([]VentilationNode, []AirwayEdge){
		"新增节点": func(n []VentilationNode, e []AirwayEdge) ([]VentilationNode, []AirwayEdge) {
			return append(append([]VentilationNode(nil), n...), VentilationNode{ID: 9, Code: "JCT-99", NodeType: "junction", Status: "inactive"}), e
		},
		"停用节点": func(n []VentilationNode, e []AirwayEdge) ([]VentilationNode, []AirwayEdge) {
			copyN := append([]VentilationNode(nil), n...)
			copyN[1].Status = "inactive"
			return copyN, e
		},
		"节点参数变化": func(n []VentilationNode, e []AirwayEdge) ([]VentilationNode, []AirwayEdge) {
			copyN := append([]VentilationNode(nil), n...)
			copyN[2].PressurePa = 555
			return copyN, e
		},
		"新增巷道": func(n []VentilationNode, e []AirwayEdge) ([]VentilationNode, []AirwayEdge) {
			return n, append(append([]AirwayEdge(nil), e...), AirwayEdge{ID: 8, Code: "AW-108", FromNodeID: 1, ToNodeID: 3, ResistanceNS2M8: 3, AreaM2: 6, MaxVelocityMS: 8, DoorState: "open", Enabled: true})
		},
		"停用巷道": func(n []VentilationNode, e []AirwayEdge) ([]VentilationNode, []AirwayEdge) {
			copyE := append([]AirwayEdge(nil), e...)
			copyE[0].Enabled = false
			return n, copyE
		},
		"巷道参数变化": func(n []VentilationNode, e []AirwayEdge) ([]VentilationNode, []AirwayEdge) {
			copyE := append([]AirwayEdge(nil), e...)
			copyE[0].ResistanceNS2M8 = 2.2
			return n, copyE
		},
		"风门状态变化": func(n []VentilationNode, e []AirwayEdge) ([]VentilationNode, []AirwayEdge) {
			copyE := append([]AirwayEdge(nil), e...)
			copyE[0].DoorState = "closed"
			return n, copyE
		},
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			nodes, edges := fingerprintFixture()
			changedN, changedE := mutate(nodes, edges)
			if got := ComputeNetworkFingerprint(changedN, changedE); got == base {
				t.Fatalf("%s 必须改变网络指纹", name)
			}
		})
	}
}
