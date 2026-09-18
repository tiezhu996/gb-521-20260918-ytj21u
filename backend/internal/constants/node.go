package constants

type NodeType string

const (
	NodeTypeIntake   NodeType = "intake"
	NodeTypeExhaust  NodeType = "exhaust"
	NodeTypeWorkface NodeType = "workface"
	NodeTypeJunction NodeType = "junction"
)

type NodeStatus string

const (
	NodeStatusActive   NodeStatus = "active"
	NodeStatusInactive NodeStatus = "inactive"
	NodeStatusBlocked  NodeStatus = "blocked"
)

func ValidNodeType(value string) bool {
	switch NodeType(value) {
	case NodeTypeIntake, NodeTypeExhaust, NodeTypeWorkface, NodeTypeJunction:
		return true
	default:
		return false
	}
}

func ValidNodeStatus(value string) bool {
	switch NodeStatus(value) {
	case NodeStatusActive, NodeStatusInactive, NodeStatusBlocked:
		return true
	default:
		return false
	}
}
