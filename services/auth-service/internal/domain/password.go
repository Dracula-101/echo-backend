package domain

import "time"

type PasswordHistoryEntry struct {
	Hash      string    `json:"hash"`
	Salt      string    `json:"salt"`
	Algorithm string    `json:"algorithm"`
	ChangedAt time.Time `json:"changed_at"`
}
