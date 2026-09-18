package constants

type DoorState string

const (
	DoorStateOpen     DoorState = "open"
	DoorStateClosed   DoorState = "closed"
	DoorStateRegulate DoorState = "regulating"
)

func ValidDoorState(value string) bool {
	switch DoorState(value) {
	case DoorStateOpen, DoorStateClosed, DoorStateRegulate:
		return true
	default:
		return false
	}
}

func DoorResistanceMultiplier(value string) float64 {
	switch DoorState(value) {
	case DoorStateClosed:
		return 1000
	case DoorStateRegulate:
		return 2.5
	default:
		return 1
	}
}
