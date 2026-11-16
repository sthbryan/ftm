package supervisor

import (
	"strings"

	"github.com/sthbryan/ftm/internal/provider"
)

const failureTailLines = 3

var failureMarkers = []string{"error", "fatal", "failed", "panic", "refused", "denied"}

func failureReason(lines []string, fallback string) string {
	cleaned := meaningfulLines(lines)
	if len(cleaned) == 0 {
		return fallback
	}

	if reported := reportedFailure(cleaned); reported != "" {
		return reported
	}

	return cleaned[len(cleaned)-1]
}

func reportedFailure(lines []string) string {
	cleaned := meaningfulLines(lines)

	start := -1
	for i := len(cleaned) - 1; i >= 0 && i >= len(cleaned)-failureTailLines; i-- {
		if hasFailureMarker(cleaned[i]) {
			start = i
		}
	}

	if start < 0 {
		return ""
	}

	return strings.Join(cleaned[start:], " ")
}

func meaningfulLines(lines []string) []string {
	cleaned := make([]string, 0, len(lines))

	for _, line := range lines {
		if trimmed := strings.TrimSpace(provider.StripANSI(line)); trimmed != "" {
			cleaned = append(cleaned, trimmed)
		}
	}

	return cleaned
}

func hasFailureMarker(line string) bool {
	lower := strings.ToLower(line)

	for _, marker := range failureMarkers {
		if strings.Contains(lower, marker) {
			return true
		}
	}

	return false
}
