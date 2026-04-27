package domain

import "github.com/google/uuid"

type Mention struct {
	UserID   uuid.UUID `json:"user_id"`
	Username string    `json:"username,omitempty"`
	Offset   int       `json:"offset"`
	Length   int       `json:"length"`
}

type Location struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Name      string  `json:"name,omitempty"`
	Address   string  `json:"address,omitempty"`
}

type Contact struct {
	Name  string `json:"name"`
	Phone string `json:"phone,omitempty"`
	Email string `json:"email,omitempty"`
}
