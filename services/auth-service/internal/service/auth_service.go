package service

import (
	"auth-service/api/dto"
	authErrors "auth-service/internal/errors"
	"auth-service/internal/model"
	repository "auth-service/internal/repo"
	"context"
	"fmt"
	"strings"
	"time"

	serviceModels "auth-service/internal/service/models"
	"encoding/base64"
	"shared/pkg/database"
	pkgErrors "shared/pkg/errors"
	"shared/pkg/logger"
	"shared/server/common/token"
)

// ============================================================================
// Error Types
// ============================================================================

type AuthError struct {
	Message string
	Code    string
	Error   error
}

// ============================================================================
// Helper Functions
// ============================================================================

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

// ============================================================================
// Email Validation
// ============================================================================

func (s *AuthService) IsEmailTaken(ctx context.Context, email string) (bool, error) {
	s.log.Info("Checking if email is taken", logger.String("email", email))
	email = normalizeEmail(email)
	return s.repo.ExistsByEmail(ctx, email)
}

// ============================================================================
// User Retrieval
// ============================================================================

func (s *AuthService) GetUserByEmail(ctx context.Context, email string) (*model.User, *AuthError) {
	s.log.Info("Fetching user by email",
		logger.String("service", authErrors.ServiceName),
		logger.String("email", email),
	)
	user, err := s.repo.GetUserByEmail(ctx, email)
	if err != nil {
		s.log.Error("Failed to fetch user",
			logger.String("service", authErrors.ServiceName),
			logger.String("error_code", pkgErrors.CodeDatabaseError),
			logger.Error(err),
		)
		return nil, &AuthError{Message: "Failed to fetch user", Code: pkgErrors.CodeDatabaseError, Error: err}
	}
	if user == nil {
		s.log.Info("User not found",
			logger.String("service", authErrors.ServiceName),
			logger.String("email", email),
		)
		return nil, nil
	}
	s.log.Debug("User fetched successfully",
		logger.String("service", authErrors.ServiceName),
		logger.String("email", email),
		logger.String("user_id", user.ID),
	)

	return &model.User{
		ID:                     user.ID,
		Email:                  user.Email,
		PhoneNumber:            user.PhoneNumber,
		PhoneCountryCode:       user.PhoneCountryCode,
		EmailVerified:          user.EmailVerified,
		PhoneVerified:          user.PhoneVerified,
		AccountStatus:          user.AccountStatus,
		TwoFactorEnabled:       user.TwoFactorEnabled,
		PasswordHash:           user.PasswordHash,
		PasswordLastChanged:    user.PasswordLastChangedAt,
		AccountLockedUntil:     user.AccountLockedUntil,
		FailedLoginAttempts:    user.FailedLoginAttempts,
		RequiresPasswordChange: user.RequiresPasswordChange,
		DeletedAt:              user.DeletedAt,
		CreatedAt:              user.CreatedAt,
		UpdatedAt:              user.UpdatedAt,
	}, nil
}

// ============================================================================
// User Registration
// ============================================================================

func (s *AuthService) RegisterUser(ctx context.Context, input serviceModels.RegisterUserInput) (*serviceModels.RegisterUserOutput, *AuthError) {
	s.log.Info("Registering new user",
		logger.String("service", authErrors.ServiceName),
		logger.String("email", input.Email),
	)
	input.Email = normalizeEmail(input.Email)

	result, err := s.hashingService.HashPassword(ctx, input.Password)
	if err != nil {
		s.log.Error("Failed to hash password",
			logger.String("service", authErrors.ServiceName),
			logger.String("error_code", authErrors.CodePasswordHashingFailed),
			logger.Error(err),
		)
		return nil, &AuthError{Message: "Failed to process registration", Code: authErrors.CodePasswordHashingFailed, Error: err}
	}
	s.log.Debug("Password hashed successfully",
		logger.String("service", authErrors.ServiceName),
		logger.String("hash", result.Encoded),
		logger.String("algorithm", string(result.Algorithm)),
	)

	tokenResult, err := s.tokenService.IssueAccessToken(ctx, input.Email, token.IssueOptions{
		ExpiresIn: 24 * time.Hour,
		Metadata: map[string]interface{}{
			"purpose": "email_verification",
		},
		Audience: []string{"auth_service_email_verification"},
	})
	if err != nil {
		s.log.Error("Failed to generate verification token",
			logger.String("service", authErrors.ServiceName),
			logger.String("error_code", authErrors.CodeTokenGenerationFailed),
			logger.Error(err),
		)
		return nil, &AuthError{Message: "Failed to process registration", Code: authErrors.CodeTokenGenerationFailed, Error: err}
	}
	s.log.Debug("Email verification token generated successfully",
		logger.String("service", authErrors.ServiceName),
		logger.String("token", tokenResult.Token),
	)

	userID, err := s.repo.CreateUser(ctx, repository.CreateUserParams{
		Email:             input.Email,
		PasswordHash:      result.Encoded,
		PasswordSalt:      base64.StdEncoding.EncodeToString(result.Salt),
		PasswordAlgorithm: string(result.Algorithm),
		PhoneNumber:       input.PhoneNumber,
		PhoneCountryCode:  input.PhoneCountryCode,
		IPAddress:         input.IPAddress,
		UserAgent:         input.UserAgent,
	})
	if err != nil {
		s.log.Error("Failed to create user",
			logger.String("service", authErrors.ServiceName),
			logger.String("error_code", pkgErrors.CodeDatabaseError),
			logger.Error(err),
		)
		return nil, &AuthError{Message: "Failed to process registration", Code: pkgErrors.CodeDatabaseError, Error: err}
	}

	s.log.Info("User registered successfully",
		logger.String("service", authErrors.ServiceName),
		logger.String("user_id", userID),
		logger.String("email", input.Email),
	)

	return &serviceModels.RegisterUserOutput{
		UserID:                userID,
		Email:                 input.Email,
		EmailVerificationSent: true,
		VerificationToken:     tokenResult.Token,
	}, nil
}

// ============================================================================
// User Authentication
// ============================================================================

func (s *AuthService) Login(ctx context.Context, email, password string) (*dto.LoginResponse, *AuthError) {
	s.log.Info("User login",
		logger.String("service", authErrors.ServiceName),
		logger.String("email", email),
	)
	user, err := s.repo.GetUserByEmail(ctx, email)
	if err != nil && !database.IsNoRowsError(err) {
		s.log.Error("Database error during login",
			logger.String("service", authErrors.ServiceName),
			logger.String("error_code", pkgErrors.CodeDatabaseError),
			logger.Error(err),
		)
		return nil, &AuthError{Message: "Failed to process login", Code: pkgErrors.CodeDatabaseError, Error: err}
	}
	if user == nil {
		s.log.Warn("Login attempt for non-existent user",
			logger.String("service", authErrors.ServiceName),
			logger.String("email", email),
		)
		return nil, &AuthError{Message: "Invalid credentials", Code: authErrors.CodeUserNotFound, Error: nil}
	}
	s.log.Debug("User fetched for login",
		logger.String("service", authErrors.ServiceName),
		logger.String("email", email),
	)

	success, algo, verifyErr := s.hashingService.VerifyPassword(ctx, password, user.PasswordHash)
	if verifyErr != nil {
		s.log.Error(fmt.Sprintf("Password verification failed using algorithm %s", algo),
			logger.String("service", authErrors.ServiceName),
			logger.String("error_code", authErrors.CodeInvalidCredentials),
			logger.Error(verifyErr),
		)
		return nil, &AuthError{Message: "Error verifying credentials", Code: authErrors.CodeInvalidCredentials, Error: verifyErr}
	}
	if !success {
		s.log.Warn("Invalid password attempt",
			logger.String("service", authErrors.ServiceName),
			logger.String("email", email),
		)
		return nil, &AuthError{Message: "Invalid credentials", Code: authErrors.CodeInvalidCredentials, Error: nil}
	}

	accessToken, tokenErr := s.tokenService.IssueAccessToken(ctx, user.ID, token.IssueOptions{
		ExpiresIn: s.cfg.JWT.AccessTokenTTL,
		Metadata: map[string]interface{}{
			"purpose": "access_token",
		},
		Audience: []string{"auth_service_access"},
	})
	if tokenErr != nil {
		s.log.Error("Failed to generate access token",
			logger.String("service", authErrors.ServiceName),
			logger.String("error_code", authErrors.CodeTokenGenerationFailed),
			logger.Error(tokenErr),
		)
		return nil, &AuthError{Message: "Failed to generate access token", Code: authErrors.CodeTokenGenerationFailed, Error: tokenErr}
	}

	expiresAt := accessToken.Claims.IssuedAt.Add(s.cfg.JWT.AccessTokenTTL)
	refreshToken, refreshErr := s.tokenService.IssueRefreshToken(ctx, user.ID, token.IssueOptions{
		ExpiresIn: s.cfg.JWT.RefreshTokenTTL,
		Metadata: map[string]interface{}{
			"purpose": "refresh_token",
		},
		Audience: []string{"auth_service_refresh"},
	})
	if refreshErr != nil {
		s.log.Error("Failed to generate refresh token",
			logger.String("service", authErrors.ServiceName),
			logger.String("error_code", authErrors.CodeTokenGenerationFailed),
			logger.Error(refreshErr),
		)
		return nil, &AuthError{Message: "Failed to generate refresh token", Code: authErrors.CodeTokenGenerationFailed, Error: refreshErr}
	}

	return &dto.LoginResponse{
		User: dto.User{
			ID:               user.ID,
			Email:            user.Email,
			PhoneNumber:      *user.PhoneNumber,
			PhoneCountryCode: *user.PhoneCountryCode,
			EmailVerified:    user.EmailVerified,
			PhoneVerified:    user.PhoneVerified,
			AccountStatus:    user.AccountStatus,
			TFAEnabled:       user.TwoFactorEnabled,
			CreatedAt:        user.CreatedAt.Unix(),
			UpdatedAt:        user.UpdatedAt.Unix(),
		},
		Session: dto.Session{
			AccessToken:  accessToken.Token,
			RefreshToken: refreshToken.Token,
			ExpiresAt:    expiresAt.Unix(),
			TokenType:    "Bearer",
		},
	}, nil
}
