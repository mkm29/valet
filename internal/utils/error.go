package utils

import (
	"errors"
	"strings"
	"syscall"
)

// ErrorType returns a simplified error type for metrics and logging
func ErrorType(err error) string {
	if err == nil {
		return ""
	}
	// You can add more specific error type detection here
	return "generic"
}

// IsIgnorableSyncError checks if an error from syncing file descriptors should be ignored
func IsIgnorableSyncError(err error) bool {
	if err == nil {
		return true
	}

	errStr := err.Error()

	// Check for common sync errors that are safe to ignore
	ignorablePatterns := []string{
		"sync /dev/stdout",
		"sync /dev/stderr",
		"sync /dev/stdin",
		"bad file descriptor",
		"invalid argument",
	}

	for _, pattern := range ignorablePatterns {
		if strings.Contains(errStr, pattern) {
			return true
		}
	}

	// Check for specific system errors
	var errno syscall.Errno
	if errors.As(err, &errno) {
		switch errno {
		case syscall.EBADF, // Bad file descriptor
			syscall.EINVAL, // Invalid argument
			syscall.ENOTTY: // Not a typewriter (common for stdout/stderr)
			return true
		}
	}

	return false
}
