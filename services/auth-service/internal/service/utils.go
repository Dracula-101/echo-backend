package service

import "strings"

// ============================================================================
// Helper Functions
// ============================================================================

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}
