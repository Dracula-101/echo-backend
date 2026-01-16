package domain

type Mention struct {
	UserID string `json:"user_id"`
	Offset int    `json:"offset"`
	Length int    `json:"length"`
}
