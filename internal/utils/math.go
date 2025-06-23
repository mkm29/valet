package utils

// CalculateDelta calculates the delta between current and last values,
// handling counter resets gracefully. This is useful for monotonic counters
// that can only increase but may reset (e.g., after a restart).
func CalculateDelta(current, last int64) int64 {
	if current < last {
		// Counter reset detected - use current value as the delta
		// This assumes the counter was reset to 0 and then incremented to current
		return current
	}
	return current - last
}
