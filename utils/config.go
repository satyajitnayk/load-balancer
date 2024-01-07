package utils

type LBStrategy int

// define enum
const (
	RoundRobin LBStrategy = iota
	LeastConnected
)

func GetLBStrategy(strategy string) LBStrategy {
	switch strategy {
	case "least-connected":
		return LeastConnected
	default:
		return RoundRobin
	}
}
