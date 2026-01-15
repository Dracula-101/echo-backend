package model

import (
	"database/sql"
	"time"
)

type Profile struct {
	ID                string       `db:"id"`
	UserID            string       `db:"user_id"`
	Username          string       `db:"username"`
	DisplayName       *string      `db:"display_name"`
	FirstName         *string      `db:"first_name"`
	LastName          *string      `db:"last_name"`
	Bio               *string      `db:"bio"`
	AvatarURL         *string      `db:"avatar_url"`
	LanguageCode      string       `db:"language_code"`
	Timezone          *string      `db:"timezone"`
	CountryCode       *string      `db:"country_code"`
	PhoneVisible      bool         `db:"phone_visible"`
	EmailVisible      bool         `db:"email_visible"`
	OnlineStatus      string       `db:"online_status"`
	LastSeenAt        *time.Time   `db:"last_seen_at"`
	ProfileVisibility string       `db:"profile_visibility"`
	SearchVisibility  bool         `db:"search_visibility"`
	IsVerified        bool         `db:"is_verified"`
	CreatedAt         time.Time    `db:"created_at"`
	UpdatedAt         time.Time    `db:"updated_at"`
	DeactivatedAt     sql.NullTime `db:"deactivated_at"`
}

func (p *Profile) TableName() string {
	return "users.profiles"
}

func (p *Profile) PrimaryKey() interface{} {
	return p.ID
}
