package utils

import (
	"time"

	"golang.org/x/text/language"
)

// language code
func IsValidLanguageCode(code string) bool {
	if code == "" {
		return false
	}

	_, err := language.Parse(code)
	return err == nil
}

// country code
func IsValidCountryCode(code string) bool {
	if code == "" {
		return false
	}

	_, err := language.ParseRegion(code)
	return err == nil
}

// timezone
func IsValidTimezone(tz string) bool {
	if tz == "" {
		return false
	}

	_, err := time.LoadLocation(tz)
	return err == nil
}
