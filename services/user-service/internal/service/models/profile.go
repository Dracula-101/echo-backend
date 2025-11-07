package models

type Profile struct {
	UserID       string
	Username     string
	DisplayName  string
	Bio          *string
	AvatarURL    *string
	FirstName    *string
	LastName     *string
	LanguageCode *string
	Timezone     *string
	CountryCode  *string
	IsVerified   bool
}
