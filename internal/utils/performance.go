package utils

import "time"

// CategorizePerformance categorizes the performance based on duration
func CategorizePerformance(duration time.Duration) string {
	ms := duration.Milliseconds()
	switch {
	case ms < 100:
		return "fast"
	case ms < 500:
		return "normal"
	case ms < 1000:
		return "slow"
	default:
		return "very_slow"
	}
}

// ServerState represents the state of a server
type ServerState int

const (
	// ServerStateStopped indicates the server is stopped
	ServerStateStopped ServerState = 0
	// ServerStateStarting indicates the server is starting
	ServerStateStarting ServerState = 1
	// ServerStateRunning indicates the server is running
	ServerStateRunning ServerState = 2
	// ServerStateShuttingDown indicates the server is shutting down
	ServerStateShuttingDown ServerState = 3
)

// ServerStateToString converts a numeric server state to its string representation
func ServerStateToString(state float64) string {
	switch ServerState(state) {
	case ServerStateStopped:
		return "stopped"
	case ServerStateStarting:
		return "starting"
	case ServerStateRunning:
		return "running"
	case ServerStateShuttingDown:
		return "shutting_down"
	default:
		return "unknown"
	}
}
