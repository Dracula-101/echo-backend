package service

import (
	"auth-service/api/dto"
	authErrors "auth-service/internal/errors"
	"auth-service/internal/model"
	repository "auth-service/internal/repo"
	"context"
	"strings"
	"time"

	serviceModels "auth-service/internal/service/models"
	"encoding/base64"
	"shared/pkg/database/postgres"
	pkgErrors "shared/pkg/errors"
	"shared/pkg/logger"
	"shared/server/common/token"
)

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

	exists, err := s.repo.ExistsByEmail(ctx, email)
	return exists, err
}

// ============================================================================
// User Retrieval
// ============================================================================

func (s *AuthService) GetUserByEmail(ctx context.Context, email string) (*model.User, error) {
	s.log.Info("Fetching user by email",
		logger.String("service", authErrors.ServiceName),
		logger.String("email", email),
	)

	user, err := s.repo.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, err.WithService(authErrors.ServiceName)
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

func (s *AuthService) RegisterUser(ctx context.Context, input serviceModels.RegisterUserInput) (*serviceModels.RegisterUserOutput, error) {
	s.log.Info("Registering new user",
		logger.String("service", authErrors.ServiceName),
		logger.String("email", input.Email),
	)
	input.Email = normalizeEmail(input.Email)

	result, err := s.hashingService.HashPassword(ctx, input.Password)
	if err != nil {
		return nil, pkgErrors.FromError(err, authErrors.CodePasswordHashingFailed, "failed to hash password").
			WithService(authErrors.ServiceName).
			WithDetail("email", input.Email)
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
		return nil, pkgErrors.FromError(err, authErrors.CodeTokenGenerationFailed, "failed to generate verification token").
			WithService(authErrors.ServiceName).
			WithDetail("email", input.Email)
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
		if appErr, ok := err.(pkgErrors.AppError); ok {
			return nil, appErr.WithService(authErrors.ServiceName)
		}
		return nil, pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "failed to create user").
			WithService(authErrors.ServiceName).
			WithDetail("email", input.Email)
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

func (s *AuthService) Login(ctx context.Context, email, password string) (*dto.LoginResponse, error) {
	s.log.Info("User login",
		logger.String("service", authErrors.ServiceName),
		logger.String("email", email),
	)

	user, err := s.repo.GetUserByEmail(ctx, email)
	if err != nil && !postgres.IsNoRowsError(err) {
		return nil, err.WithService(authErrors.ServiceName)
	}
	if user == nil {
		s.log.Warn("Login attempt for non-existent user",
			logger.String("service", authErrors.ServiceName),
			logger.String("email", email),
		)
		return nil, pkgErrors.New(authErrors.CodeUserNotFound, "invalid credentials").
			WithService(authErrors.ServiceName).
			WithDetail("email", email)
	}
	s.log.Debug("User fetched for login",
		logger.String("service", authErrors.ServiceName),
		logger.String("email", email),
	)

	success, algo, verifyErr := s.hashingService.VerifyPassword(ctx, password, user.PasswordHash)
	if verifyErr != nil {
		return nil, pkgErrors.FromError(verifyErr, authErrors.CodeInvalidCredentials, "password verification failed").
			WithService(authErrors.ServiceName).
			WithDetail("email", email).
			WithDetail("algorithm", algo)
	}
	if !success {
		s.log.Warn("Invalid password attempt",
			logger.String("service", authErrors.ServiceName),
			logger.String("email", email),
		)
		return nil, pkgErrors.New(authErrors.CodeInvalidCredentials, "invalid credentials").
			WithService(authErrors.ServiceName).
			WithDetail("email", email)
	}

	accessToken, tokenErr := s.tokenService.IssueAccessToken(ctx, user.ID, token.IssueOptions{
		ExpiresIn: s.cfg.JWT.AccessTokenTTL,
		Metadata: map[string]interface{}{
			"purpose": "access_token",
			"user_id": user.ID,
			"email":   user.Email,
		},
		Audience: []string{s.cfg.JWT.Audience},
	})
	if tokenErr != nil {
		return nil, pkgErrors.FromError(tokenErr, authErrors.CodeTokenGenerationFailed, "failed to generate access token").
			WithService(authErrors.ServiceName).
			WithDetail("user_id", user.ID)
	}

	expiresAt := accessToken.Claims.IssuedAt.Add(s.cfg.JWT.AccessTokenTTL)
	refreshToken, refreshErr := s.tokenService.IssueRefreshToken(ctx, user.ID, token.IssueOptions{
		ExpiresIn: s.cfg.JWT.RefreshTokenTTL,
		Metadata: map[string]interface{}{
			"purpose": "refresh_token",
		},
		Audience: []string{s.cfg.JWT.Audience},
	})
	if refreshErr != nil {
		return nil, pkgErrors.FromError(refreshErr, authErrors.CodeTokenGenerationFailed, "failed to generate refresh token").
			WithService(authErrors.ServiceName).
			WithDetail("user_id", user.ID)
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
