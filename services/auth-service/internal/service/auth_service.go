package service

import (
	"auth-service/internal/domain"
	authErrors "auth-service/internal/error"
	repoModels "auth-service/internal/repo/model"
	"context"
	"encoding/base64"
	"time"

	"auth-service/internal/error"
	"shared/pkg/database/postgres"
	pkgErrors "shared/pkg/errors"
	"shared/pkg/logger"
	"shared/pkg/utils"
	"shared/server/common/hashing"
	"shared/server/common/token"
)

// AuthServiceInterface defines the contract for authentication service operations
type AuthServiceInterface interface {
	// Email validation
	IsEmailTaken(ctx context.Context, email string) (bool, *error.AuthError)
	// User operations
	GetUserByEmail(ctx context.Context, email string) (*domain.User, *error.AuthError)
	GetUserByID(ctx context.Context, userID string) (*domain.User, *error.AuthError)
	RegisterUser(ctx context.Context, input domain.RegisterUserInput) (*domain.RegisterUserOutput, *error.AuthError)
	Login(ctx context.Context, input domain.LoginInput) (*domain.LoginResult, *error.AuthError)
	RecordFailedLoginAttempt(ctx context.Context, input domain.FailedLoginAttemptInput) *error.AuthError
	RefreshToken(ctx context.Context, refreshToken string) (*domain.LoginResult, *error.AuthError)

	// Service accessors
	TokenService() token.JWTTokenService
	HashingService() hashing.HashingService
}

// ============================================================================
// Email Validation
// ============================================================================

func (s *AuthService) IsEmailTaken(ctx context.Context, email string) (bool, *error.AuthError) {
	s.log.Info("Checking if email is taken", logger.String("email", email))
	email = normalizeEmail(email)

	exists, err := s.repo.ExistsByEmail(ctx, email)
	if err != nil {
		return false, &error.AuthError{
			Message: "Failed to check email existence",
			Code:    authErrors.CodeUserNotFound,
			Error: pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "failed to check email existence").
				WithService(authErrors.ServiceName).
				WithDetail("email", email),
		}
	}
	return exists, nil
}

// ============================================================================
// User Retrieval
// ============================================================================

func (s *AuthService) GetUserByEmail(ctx context.Context, email string) (*domain.User, *error.AuthError) {
	s.log.Info("Fetching user by email",
		logger.String("service", authErrors.ServiceName),
		logger.String("email", email),
	)

	user, err := s.repo.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, &error.AuthError{
			Message: "Failed to get user by email",
			Code:    authErrors.CodeDatabaseError,
			Error: pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "failed to get user by email").
				WithService(authErrors.ServiceName).
				WithDetail("email", email),
		}
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

	return &domain.User{
		ID:                     user.ID,
		Email:                  user.Email,
		PhoneNumber:            user.PhoneNumber,
		PhoneCountryCode:       user.PhoneCountryCode,
		EmailVerified:          user.EmailVerified,
		PhoneVerified:          user.PhoneVerified,
		AccountStatus:          user.AccountStatus,
		TwoFactorEnabled:       user.TwoFactorEnabled,
		PasswordHash:           user.PasswordHash,
		PasswordLastChanged:    user.PasswordLastChanged,
		AccountLockedUntil:     user.AccountLockedUntil,
		FailedLoginAttempts:    user.FailedLoginAttempts,
		RequiresPasswordChange: user.RequiresPasswordChange,
		CreatedAt:              user.CreatedAt,
		DeletedAt:              user.DeletedAt,
		UpdatedAt:              user.UpdatedAt,
	}, nil
}

func (s *AuthService) GetUserByID(ctx context.Context, userID string) (*domain.User, *error.AuthError) {
	s.log.Info("Fetching user by ID",
		logger.String("service", authErrors.ServiceName),
		logger.String("user_id", userID),
	)

	user, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		return nil, &error.AuthError{
			Message: "Failed to get user by ID",
			Code:    authErrors.CodeDatabaseError,
			Error: pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "failed to get user by ID").
				WithService(authErrors.ServiceName).
				WithDetail("user_id", userID),
		}
	}
	if user == nil {
		s.log.Info("User not found",
			logger.String("service", authErrors.ServiceName),
			logger.String("user_id", userID),
		)
		return nil, nil
	}
	s.log.Debug("User fetched successfully",
		logger.String("service", authErrors.ServiceName),
		logger.String("user_id", user.ID),
	)

	return &domain.User{
		ID:                     user.ID,
		Email:                  user.Email,
		PhoneNumber:            user.PhoneNumber,
		PhoneCountryCode:       user.PhoneCountryCode,
		EmailVerified:          user.EmailVerified,
		PhoneVerified:          user.PhoneVerified,
		AccountStatus:          user.AccountStatus,
		TwoFactorEnabled:       user.TwoFactorEnabled,
		PasswordHash:           user.PasswordHash,
		PasswordLastChanged:    user.PasswordLastChanged,
		AccountLockedUntil:     user.AccountLockedUntil,
		FailedLoginAttempts:    user.FailedLoginAttempts,
		RequiresPasswordChange: user.RequiresPasswordChange,
		CreatedAt:              user.CreatedAt,
		DeletedAt:              user.DeletedAt,
		UpdatedAt:              user.UpdatedAt,
	}, nil
}

// ============================================================================
// User Registration
// ============================================================================

func (s *AuthService) RegisterUser(ctx context.Context, input domain.RegisterUserInput) (*domain.RegisterUserOutput, *error.AuthError) {
	s.log.Info("Registering new user",
		logger.String("service", authErrors.ServiceName),
		logger.String("email", input.Email),
	)
	input.Email = normalizeEmail(input.Email)

	result, err := s.hashingService.HashPassword(ctx, input.Password)
	if err != nil {
		return nil, &error.AuthError{
			Message: "Failed to register user",
			Code:    authErrors.CodePasswordHashingFailed,
			Error: pkgErrors.FromError(err, authErrors.CodePasswordHashingFailed, "failed to hash password").
				WithService(authErrors.ServiceName).
				WithDetail("email", input.Email),
		}
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
		return nil, &error.AuthError{
			Message: "Failed to generate token",
			Code:    authErrors.CodeTokenGenerationFailed,
			Error: pkgErrors.FromError(err, authErrors.CodeTokenGenerationFailed, "failed to generate verification token").
				WithService(authErrors.ServiceName).
				WithDetail("email", input.Email).
				WithDetail("purpose", "email_verification"),
		}
	}
	s.log.Debug("Email verification token generated successfully",
		logger.String("service", authErrors.ServiceName),
		logger.String("token", tokenResult.Token),
	)

	userID, err := s.repo.CreateUser(ctx, repoModels.CreateUserParams{
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
		return nil, &error.AuthError{
			Message: "Failed to create user",
			Code:    authErrors.CodeDatabaseError,
			Error: pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "failed to create user").
				WithService(authErrors.ServiceName).
				WithDetail("email", input.Email),
		}
	}

	s.log.Info("User registered successfully",
		logger.String("service", authErrors.ServiceName),
		logger.String("user_id", userID),
		logger.String("email", input.Email),
	)

	return &domain.RegisterUserOutput{
		UserID:                userID,
		Email:                 input.Email,
		EmailVerificationSent: true,
		VerificationToken:     tokenResult.Token,
	}, nil
}

// ============================================================================
// User Authentication
// ============================================================================

func (s *AuthService) Login(ctx context.Context, input domain.LoginInput) (*domain.LoginResult, *error.AuthError) {
	email := normalizeEmail(input.Email)
	s.log.Info("User login",
		logger.String("service", authErrors.ServiceName),
		logger.String("email", email),
	)

	user, err := s.repo.GetUserByEmail(ctx, email)
	if err != nil && !postgres.IsNoRowsError(err) {
		return nil, &error.AuthError{
			Message: "Failed to get user by email",
			Code:    authErrors.CodeDatabaseError,
			Error: pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "failed to get user by email").
				WithService(authErrors.ServiceName).
				WithDetail("email", email).
				WithDetail("operation", "login"),
		}
	}
	if user == nil {
		s.log.Warn("Login attempt for non-existent user",
			logger.String("service", authErrors.ServiceName),
			logger.String("email", email),
		)
		return nil, &error.AuthError{
			Message: "User not found",
			Code:    authErrors.CodeUserNotFound,
			Error: pkgErrors.New(authErrors.CodeUserNotFound, "invalid credentials").
				WithService(authErrors.ServiceName).
				WithDetail("email", email),
		}
	}
	s.log.Debug("User fetched for login",
		logger.String("service", authErrors.ServiceName),
		logger.String("email", email),
	)

	if user.AccountLockedUntil != nil {
		if user.AccountLockedUntil.After(time.Now()) {
			s.log.Warn("Login attempt for locked account",
				logger.String("service", authErrors.ServiceName),
				logger.String("email", email),
				logger.Time("locked_until", *user.AccountLockedUntil),
			)
			return nil, &error.AuthError{
				Message: "Account is locked",
				Code:    authErrors.CodeAccountLocked,
				Error: pkgErrors.New(authErrors.CodeAccountLocked, "account is locked").
					WithService(authErrors.ServiceName).
					WithDetail("email", email).
					WithDetail("locked_until", user.AccountLockedUntil),
			}
		} else {
			unlockErr := s.repo.UnlockUserAccount(ctx, user.ID)
			if unlockErr != nil {
				return nil, &error.AuthError{
					Message: "Failed to unlock user account",
					Code:    authErrors.CodeDatabaseError,
					Error: pkgErrors.FromError(unlockErr, pkgErrors.CodeDatabaseError, "failed to unlock user account").
						WithService(authErrors.ServiceName).
						WithDetail("email", email).
						WithDetail("user_id", user.ID),
				}
			}
			s.log.Info("User account unlocked after lock period expired",
				logger.String("service", authErrors.ServiceName),
				logger.String("email", email),
				logger.String("user_id", user.ID),
			)
		}
	}

	success, algo, verifyErr := s.hashingService.VerifyPassword(ctx, input.Password, user.PasswordHash)
	if verifyErr != nil {
		return nil, &error.AuthError{
			Message: "Password verification failed",
			Code:    authErrors.CodeInvalidCredentials,
			Error: pkgErrors.FromError(verifyErr, authErrors.CodeInvalidCredentials, "password verification failed").
				WithService(authErrors.ServiceName).
				WithDetail("email", email).
				WithDetail("algorithm", algo),
		}
	}
	if !success {
		s.log.Warn("Invalid password attempt",
			logger.String("service", authErrors.ServiceName),
			logger.String("email", email),
		)

		return nil, &error.AuthError{
			Message: "Wrong email or password",
			Code:    authErrors.CodeInvalidCredentials,
			Error: pkgErrors.New(authErrors.CodeInvalidCredentials, "Wrong email or password").
				WithService(authErrors.ServiceName).
				WithDetail("email", email).
				WithDetail("algorithm", algo),
		}
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
		return nil, &error.AuthError{
			Message: "Failed to generate token",
			Code:    authErrors.CodeTokenGenerationFailed,
			Error: pkgErrors.FromError(tokenErr, authErrors.CodeTokenGenerationFailed, "failed to generate access token").
				WithService(authErrors.ServiceName).
				WithDetail("user_id", user.ID).
				WithDetail("email", user.Email).
				WithDetail("purpose", "access_token"),
		}
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
		return nil, &error.AuthError{
			Message: "Failed to generate token",
			Code:    authErrors.CodeTokenGenerationFailed,
			Error: pkgErrors.FromError(refreshErr, authErrors.CodeTokenGenerationFailed, "failed to generate refresh token").
				WithService(authErrors.ServiceName).
				WithDetail("user_id", user.ID).
				WithDetail("email", user.Email).
				WithDetail("purpose", "refresh_token"),
		}
	}

	return &domain.LoginResult{
		User: &domain.User{
			ID:                     user.ID,
			Email:                  user.Email,
			PhoneNumber:            user.PhoneNumber,
			PhoneCountryCode:       user.PhoneCountryCode,
			EmailVerified:          user.EmailVerified,
			PhoneVerified:          user.PhoneVerified,
			AccountStatus:          user.AccountStatus,
			TwoFactorEnabled:       user.TwoFactorEnabled,
			PasswordHash:           user.PasswordHash,
			PasswordSalt:           user.PasswordSalt,
			PasswordAlgorithm:      user.PasswordAlgorithm,
			PasswordLastChanged:    user.PasswordLastChanged,
			AccountLockedUntil:     user.AccountLockedUntil,
			FailedLoginAttempts:    user.FailedLoginAttempts,
			RequiresPasswordChange: user.RequiresPasswordChange,
			LastFailedLoginAt:      user.LastFailedLoginAt,
			LastSuccessfulLoginAt:  user.LastSuccessfulLoginAt,
			CreatedAt:              user.CreatedAt,
			UpdatedAt:              user.UpdatedAt,
			DeletedAt:              user.DeletedAt,
		},
		AccessToken:  accessToken.Token,
		RefreshToken: refreshToken.Token,
		ExpiresAt:    expiresAt,
	}, nil
}

func (s *AuthService) RecordFailedLoginAttempt(ctx context.Context, input domain.FailedLoginAttemptInput) *error.AuthError {

	record := repoModels.CreateLoginHistoryInput{
		DeviceInfo: input.Device,
		UserID:     input.UserID,
		SessionID:  nil,
	}

	if input.Location != nil {
		record.IPInfo = *input.Location
	}

	loginMethod := input.LoginMethod
	status := "failure"
	record.LoginMethod = &loginMethod
	record.Status = &status
	record.FailureReason = utils.PtrString(input.Reason)
	record.UserAgent = utils.PtrString(input.UserAgent)
	record.IsNewDevice = utils.PtrBool(input.IsNewDevice)
	record.IsNewLocation = utils.PtrBool(input.IsNewLocation)

	_ = s.loginHistoryRepo.CreateLoginHistory(ctx, record)
	return nil
}

// ============================================================================
// Token Refresh
// ============================================================================

func (s *AuthService) RefreshToken(ctx context.Context, refreshToken string) (*domain.LoginResult, *error.AuthError) {
	s.log.Info("Refreshing token",
		logger.String("service", authErrors.ServiceName),
	)

	claims, err := s.tokenService.Validate(ctx, refreshToken, token.TokenTypeRefresh)
	if err != nil {
		return nil, &error.AuthError{
			Message: "Invalid or expired refresh token",
			Code:    authErrors.CodeInvalidToken,
			Error: pkgErrors.FromError(err, authErrors.CodeInvalidToken, "invalid or expired refresh token").
				WithService(authErrors.ServiceName),
		}
	}

	userID := claims.Subject
	s.log.Debug("Refresh token validated",
		logger.String("service", authErrors.ServiceName),
		logger.String("user_id", userID),
	)

	user, err2 := s.repo.GetUserByID(ctx, userID)
	if err2 != nil && !postgres.IsNoRowsError(err2) {
		return nil, &error.AuthError{
			Message: "Failed to get user by ID",
			Code:    authErrors.CodeDatabaseError,
			Error: pkgErrors.FromError(err2, pkgErrors.CodeDatabaseError, "failed to get user by ID").
				WithService(authErrors.ServiceName).
				WithDetail("user_id", userID),
		}
	}
	if user == nil {
		s.log.Warn("Token refresh attempt for non-existent user",
			logger.String("service", authErrors.ServiceName),
			logger.String("user_id", userID),
		)
		return nil, &error.AuthError{
			Message: "User not found",
			Code:    authErrors.CodeUserNotFound,
			Error: pkgErrors.New(authErrors.CodeUserNotFound, "user not found").
				WithService(authErrors.ServiceName).
				WithDetail("user_id", userID),
		}
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
		return nil, &error.AuthError{
			Message: "Failed to generate token",
			Code:    authErrors.CodeTokenGenerationFailed,
			Error: pkgErrors.FromError(tokenErr, authErrors.CodeTokenGenerationFailed, "failed to generate access token").
				WithService(authErrors.ServiceName).
				WithDetail("user_id", user.ID).
				WithDetail("email", user.Email).
				WithDetail("purpose", "access_token"),
		}
	}

	expiresAt := accessToken.Claims.IssuedAt.Add(s.cfg.JWT.AccessTokenTTL)
	newRefreshToken, refreshErr := s.tokenService.IssueRefreshToken(ctx, user.ID, token.IssueOptions{
		ExpiresIn: s.cfg.JWT.RefreshTokenTTL,
		Metadata: map[string]interface{}{
			"purpose": "refresh_token",
		},
		Audience: []string{s.cfg.JWT.Audience},
	})
	if refreshErr != nil {
		return nil, &error.AuthError{
			Message: "Failed to generate token",
			Code:    authErrors.CodeTokenGenerationFailed,
			Error: pkgErrors.FromError(refreshErr, authErrors.CodeTokenGenerationFailed, "failed to generate refresh token").
				WithService(authErrors.ServiceName).
				WithDetail("user_id", user.ID).
				WithDetail("email", user.Email).
				WithDetail("purpose", "refresh_token"),
		}
	}

	return &domain.LoginResult{
		User: &domain.User{
			ID:                     user.ID,
			Email:                  user.Email,
			PhoneNumber:            user.PhoneNumber,
			PhoneCountryCode:       user.PhoneCountryCode,
			EmailVerified:          user.EmailVerified,
			PhoneVerified:          user.PhoneVerified,
			AccountStatus:          user.AccountStatus,
			TwoFactorEnabled:       user.TwoFactorEnabled,
			PasswordHash:           user.PasswordHash,
			PasswordSalt:           user.PasswordSalt,
			PasswordAlgorithm:      user.PasswordAlgorithm,
			PasswordLastChanged:    user.PasswordLastChanged,
			AccountLockedUntil:     user.AccountLockedUntil,
			FailedLoginAttempts:    user.FailedLoginAttempts,
			RequiresPasswordChange: user.RequiresPasswordChange,
			LastFailedLoginAt:      user.LastFailedLoginAt,
			LastSuccessfulLoginAt:  user.LastSuccessfulLoginAt,
			CreatedAt:              user.CreatedAt,
			UpdatedAt:              user.UpdatedAt,
			DeletedAt:              user.DeletedAt,
		},
		AccessToken:  accessToken.Token,
		RefreshToken: newRefreshToken.Token,
		ExpiresAt:    expiresAt,
	}, nil
}
