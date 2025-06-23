package utils

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"path/filepath"
	"strings"
	"time"
)

// MaskString returns a masked representation for logging
func MaskString(value, whenPresent, whenEmpty string) string {
	if value != "" {
		return whenPresent
	}
	return whenEmpty
}

// SanitizePath removes sensitive information from file paths
// It returns only the filename and immediate parent directory
func SanitizePath(path string) string {
	if path == "" {
		return ""
	}

	// Clean the path
	path = filepath.Clean(path)

	// Remove any home directory references
	if strings.HasPrefix(path, "~/") {
		path = strings.TrimPrefix(path, "~/")
	}

	// Get the base name and parent directory
	dir := filepath.Dir(path)
	base := filepath.Base(path)

	// If we have a parent directory, include just the immediate parent
	if dir != "." && dir != "/" && dir != "" {
		parentDir := filepath.Base(dir)
		return filepath.Join(parentDir, base)
	}

	return base
}

// FormatBytes converts bytes to human-readable format
func FormatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// GenerateRequestID generates a unique request ID for correlation
func GenerateRequestID() string {
	// Generate 8 random bytes
	bytes := make([]byte, 8)
	if _, err := rand.Read(bytes); err != nil {
		// Fallback to timestamp-based ID if random generation fails
		return fmt.Sprintf("req-%d", time.Now().UnixNano())
	}
	// Convert to hex string with prefix
	return fmt.Sprintf("req-%s", hex.EncodeToString(bytes))
}
