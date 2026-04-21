package utils

import (
	"errors"

	"github.com/nyaruka/phonenumbers"
)

type PhoneInfo struct {
	E164           string // +14155552671
	National       string // (415) 555-2671
	CountryISO     string // US
	CountryCode    int    // 1
	NationalNumber uint64
}

func ValidatePhoneNumber(phone string, countryISO string) (*PhoneInfo, error) {
	if phone == "" {
		return nil, errors.New("phone number is required")
	}
	if countryISO == "" {
		return nil, errors.New("country code is required")
	}

	num, err := phonenumbers.Parse(phone, countryISO)
	if err != nil || !phonenumbers.IsValidNumber(num) {
		return nil, errors.New("invalid phone number")
	}

	region := phonenumbers.GetRegionCodeForNumber(num)
	if region != countryISO {
		return nil, errors.New("phone number does not match country code")
	}

	return &PhoneInfo{
		E164:           phonenumbers.Format(num, phonenumbers.E164),
		National:       phonenumbers.Format(num, phonenumbers.NATIONAL),
		CountryISO:     region,
		CountryCode:    int(num.GetCountryCode()),
		NationalNumber: num.GetNationalNumber(),
	}, nil
}

func ValidatePhoneNumberE164(phone string) (*PhoneInfo, error) {
	if phone == "" {
		return nil, errors.New("phone number is required")
	}

	num, err := phonenumbers.Parse(phone, "")
	if err != nil || !phonenumbers.IsValidNumber(num) {
		return nil, errors.New("invalid phone number")
	}

	region := phonenumbers.GetRegionCodeForNumber(num)

	return &PhoneInfo{
		E164:           phonenumbers.Format(num, phonenumbers.E164),
		National:       phonenumbers.Format(num, phonenumbers.NATIONAL),
		CountryISO:     region,
		CountryCode:    int(num.GetCountryCode()),
		NationalNumber: num.GetNationalNumber(),
	}, nil
}
