package validator

import (
	"testing"
)

func TestIsRequired(t *testing.T) {
	tests := []struct {
		name    string
		field   string
		value   string
		wantErr bool
	}{
		{"valid non-empty", "username", "john_doe", false},
		{"invalid empty", "username", "", true},
		{"invalid whitespace", "username", "   ", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := IsRequired(tt.field, tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("IsRequired() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestIsEmail(t *testing.T) {
	tests := []struct {
		name    string
		field   string
		value   string
		wantErr bool
	}{
		{"valid email", "email", "test@example.com", false},
		{"valid with subdomain", "email", "test@mail.example.com", false},
		{"invalid no @", "email", "testexample.com", true},
		{"invalid no domain", "email", "test@", true},
		{"invalid no TLD", "email", "test@example", true},
		{"empty", "email", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := IsEmail(tt.field, tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("IsEmail() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestIsPhone(t *testing.T) {
	tests := []struct {
		name    string
		field   string
		value   string
		wantErr bool
	}{
		{"valid E.164", "phone", "+14155552671", false},
		{"valid international", "phone", "+442071838750", false},
		{"invalid no plus", "phone", "14155552671", true},
		{"invalid letters", "phone", "+1415555ABCD", true},
		{"invalid too short", "phone", "+1", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := IsPhone(tt.field, tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("IsPhone() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestIsUUID(t *testing.T) {
	tests := []struct {
		name    string
		field   string
		value   string
		wantErr bool
	}{
		{"valid UUID", "id", "550e8400-e29b-41d4-a716-446655440000", false},
		{"valid uppercase", "id", "550E8400-E29B-41D4-A716-446655440000", false},
		{"invalid format", "id", "550e8400-e29b-41d4-a716", true},
		{"invalid chars", "id", "550e8400-e29b-41d4-a716-44665544000g", true},
		{"empty", "id", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := IsUUID(tt.field, tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("IsUUID() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestIsStrongPassword(t *testing.T) {
	tests := []struct {
		name    string
		field   string
		value   string
		wantErr bool
	}{
		{"valid strong", "password", "MyP@ssw0rd", false},
		{"valid complex", "password", "Abcd123!@#", false},
		{"invalid too short", "password", "Ab1!", true},
		{"invalid no uppercase", "password", "myp@ssw0rd", true},
		{"invalid no lowercase", "password", "MYP@SSW0RD", true},
		{"invalid no number", "password", "MyP@ssword", true},
		{"invalid no special", "password", "MyPassword0", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := IsStrongPassword(tt.field, tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("IsStrongPassword() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestMinLength(t *testing.T) {
	tests := []struct {
		name    string
		field   string
		value   string
		min     int
		wantErr bool
	}{
		{"valid exact min", "name", "john", 4, false},
		{"valid above min", "name", "johnny", 4, false},
		{"invalid below min", "name", "joe", 4, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := MinLength(tt.field, tt.value, tt.min)
			if (err != nil) != tt.wantErr {
				t.Errorf("MinLength() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestMaxLength(t *testing.T) {
	tests := []struct {
		name    string
		field   string
		value   string
		max     int
		wantErr bool
	}{
		{"valid below max", "name", "john", 10, false},
		{"valid exact max", "name", "johnathan", 9, false},
		{"invalid above max", "name", "johnathan", 5, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := MaxLength(tt.field, tt.value, tt.max)
			if (err != nil) != tt.wantErr {
				t.Errorf("MaxLength() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestIsPositive(t *testing.T) {
	tests := []struct {
		name    string
		field   string
		value   int
		wantErr bool
	}{
		{"valid positive", "count", 1, false},
		{"valid large", "count", 1000, false},
		{"invalid zero", "count", 0, true},
		{"invalid negative", "count", -1, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := IsPositive(tt.field, tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("IsPositive() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestIsOneOf(t *testing.T) {
	allowed := []string{"admin", "user", "guest"}

	tests := []struct {
		name    string
		field   string
		value   string
		wantErr bool
	}{
		{"valid admin", "role", "admin", false},
		{"valid user", "role", "user", false},
		{"invalid role", "role", "superuser", true},
		{"empty", "role", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := IsOneOf(tt.field, tt.value, allowed)
			if (err != nil) != tt.wantErr {
				t.Errorf("IsOneOf() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestChainValidator(t *testing.T) {
	t.Run("no errors", func(t *testing.T) {
		validator := NewChainValidator()
		validator.Add(IsRequired("username", "john"))
		validator.Add(MinLength("username", "john", 3))

		if validator.HasErrors() {
			t.Errorf("expected no errors, got %v", validator.Errors())
		}
	})

	t.Run("multiple errors", func(t *testing.T) {
		validator := NewChainValidator()
		validator.Add(IsRequired("username", ""))
		validator.Add(IsEmail("email", "invalid"))

		if !validator.HasErrors() {
			t.Error("expected errors, got none")
		}

		if len(validator.Errors()) != 2 {
			t.Errorf("expected 2 errors, got %d", len(validator.Errors()))
		}
	})

	t.Run("validate returns error", func(t *testing.T) {
		validator := NewChainValidator()
		validator.Add(IsRequired("name", ""))

		err := validator.Validate()
		if err == nil {
			t.Error("expected validation error, got nil")
		}
	})
}

func TestIsSlug(t *testing.T) {
	tests := []struct {
		name    string
		field   string
		value   string
		wantErr bool
	}{
		{"valid slug", "slug", "hello-world", false},
		{"valid with numbers", "slug", "hello-world-123", false},
		{"invalid uppercase", "slug", "Hello-World", true},
		{"invalid spaces", "slug", "hello world", true},
		{"invalid underscore", "slug", "hello_world", true},
		{"invalid special chars", "slug", "hello@world", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := IsSlug(tt.field, tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("IsSlug() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestInRange(t *testing.T) {
	tests := []struct {
		name    string
		field   string
		value   int
		min     int
		max     int
		wantErr bool
	}{
		{"valid in range", "age", 25, 18, 65, false},
		{"valid min boundary", "age", 18, 18, 65, false},
		{"valid max boundary", "age", 65, 18, 65, false},
		{"invalid below min", "age", 17, 18, 65, true},
		{"invalid above max", "age", 66, 18, 65, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := InRange(tt.field, tt.value, tt.min, tt.max)
			if (err != nil) != tt.wantErr {
				t.Errorf("InRange() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
