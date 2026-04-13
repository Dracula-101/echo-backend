package service

import (
	"context"
	"encoding/base64"
	"time"

	"auth-service/internal/domain"
	"auth-service/internal/error"
	authErrors "auth-service/internal/error"
	repoModels "auth-service/internal/repo/model"

	"shared/pkg/database/postgres"
	"shared/pkg/database/postgres/models"
	pkgErrors "shared/pkg/errors"
	"shared/pkg/logger"
	"shared/pkg/utils"
	"shared/server/common/token"
)

const (
	MaxLoginAttempts = 10
)

// ============================================================================
// User Registration
// ============================================================================

func (s *AuthService) RegisterUser(ctx context.Context, input domain.RegisterUserInput) (*domain.RegisterUserOutput, *error.AuthError) {
	s.log.Info("Registering new user",
		logger.String("service", authErrors.ServiceName),
		logger.String("email", input.Email),
	)
	input.Email = utils.NormalizeEmail(input.Email)

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

	userID, err := s.repo.CreateUser(ctx, repoModels.CreateUserParams{
		Email:             input.Email,
		PasswordHash:      result.Encoded,
		PasswordSalt:      base64.StdEncoding.EncodeToString(result.Salt),
		PasswordAlgorithm: string(result.Algorithm),
		PhoneNumber:       input.PhoneNumber,
		PhoneCountryCode:  input.PhoneCountryCode,
		IPAddress:         input.IPAddress,
		UserAgent:         input.UserAgent,
		Country:           input.Country,
		AppVersion:        input.AppVersion,
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
		UserID:                  userID,
		Email:                   input.Email,
		EmailVerified:           false,
		PhoneVerified:           false,
		AccountStatus:           models.AccountStatusPending.String(),
		NextStep:                "verify_email",
		VerificationEmailSentTo: &input.Email,
	}, nil
}

// ============================================================================
// User Authentication
// ============================================================================

func (s *AuthService) Login(ctx context.Context, input domain.LoginInput) (*domain.LoginResult, *error.AuthError) {
	email := utils.NormalizeEmail(input.Email)
	s.log.Info("User login",
		logger.String("service", authErrors.ServiceName),
		logger.String("email", email),
	)

	user, err := s.repo.GetUserByEmail(ctx, email)
	if err != nil && !postgres.IsNotFoundError(err) {
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

		s.loginHistoryRepo.CreateLoginHistory(ctx, repoModels.CreateLoginHistoryInput{
			UserID:        user.ID,
			DeviceInfo:    input.DeviceInfo,
			IPInfo:        *input.LocationInfo,
			SessionID:     nil,
			LoginMethod:   utils.PtrString("password"),
			Status:        utils.PtrString("failure"),
			IsNewDevice:   utils.PtrBool(false),
			IsNewLocation: utils.PtrBool(false),
			FailureReason: utils.PtrString("invalid_password"),
			UserAgent:     utils.PtrString(input.DeviceInfo.UserAgent),
		})

		// get attempts and lock account if necessary
		attempts, attemptsErr := s.repo.GetLoginAttempts(ctx, user.ID)
		if attemptsErr != nil {
			s.log.Error("Failed to get login attempts",
				logger.String("service", authErrors.ServiceName),
				logger.String("email", email),
				logger.String("user_id", user.ID),
				logger.Error(attemptsErr),
			)
		} else {
			if attempts+1 >= MaxLoginAttempts {
				lockErr := s.repo.LockUserAccount(ctx, user.ID)
				if lockErr != nil {
					s.log.Error("Failed to lock user account after max login attempts",
						logger.String("service", authErrors.ServiceName),
						logger.String("email", email),
						logger.String("user_id", user.ID),
						logger.Error(lockErr),
					)
				} else {
					s.log.Warn("User account locked due to too many failed login attempts",
						logger.String("service", authErrors.ServiceName),
						logger.String("email", email),
						logger.String("user_id", user.ID),
					)
					return nil, &error.AuthError{
						Message: "Account is locked due to too many failed login attempts",
						Code:    authErrors.CodeAccountRecentlyLocked,
						Error: pkgErrors.New(authErrors.CodeAccountRecentlyLocked, "account is locked due to too many failed login attempts").
							WithService(authErrors.ServiceName).
							WithDetail("email", email).
							WithDetail("user_id", user.ID),
					}
				}
			}
		}

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

	s.log.Info("User logged in successfully",
		logger.String("service", authErrors.ServiceName),
		logger.String("email", email),
		logger.String("user_id", user.ID),
	)

	isNewLocation, locationErr := s.repo.IsNewLocation(ctx, user.ID, input.LocationInfo.City)
	if locationErr != nil {
		s.log.Error("Failed to determine login location",
			logger.String("service", authErrors.ServiceName),
			logger.String("email", email),
			logger.String("user_id", user.ID),
			logger.String("city", input.LocationInfo.City),
		)
		isNewLocation = false
	}
	s.loginHistoryRepo.CreateLoginHistory(ctx, repoModels.CreateLoginHistoryInput{
		UserID:        user.ID,
		DeviceInfo:    input.DeviceInfo,
		IPInfo:        *input.LocationInfo,
		SessionID:     nil,
		LoginMethod:   utils.PtrString("password"),
		Status:        utils.PtrString("success"),
		IsNewDevice:   utils.PtrBool(false),
		IsNewLocation: utils.PtrBool(isNewLocation),
		FailureReason: nil,

		UserAgent: utils.PtrString(input.DeviceInfo.UserAgent),
	})

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

func (s *AuthService) Logout(ctx context.Context, sessionID string, userID string, ipAddress string, userAgent string) *authErrors.AuthError {
	s.log.Info("Processing logout",
		logger.String("service", authErrors.ServiceName),
		logger.String("session_id", sessionID),
		logger.String("user_id", userID),
	)

	err := s.sessionRepo.RevokeSession(ctx, sessionID, "user_logout")
	if err != nil {
		return &authErrors.AuthError{
			Message: "Failed to revoke session",
			Code:    authErrors.CodeDatabaseError,
			Error: pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "failed to revoke session").
				WithService(authErrors.ServiceName).
				WithDetail("session_id", sessionID),
		}
	}

	if s.cache != nil {
		_ = s.cache.Delete(ctx, "session_token:"+sessionID)
	}

	s.log.Info("Logout successful",
		logger.String("service", authErrors.ServiceName),
		logger.String("session_id", sessionID),
		logger.String("user_id", userID),
	)
	return nil
}

func (s *AuthService) DeleteAccount(ctx context.Context, userID string, password string, ipAddress string, userAgent string) *authErrors.AuthError {
	s.log.Info("Processing account deletion",
		logger.String("service", authErrors.ServiceName),
		logger.String("user_id", userID),
	)

	user, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		return &authErrors.AuthError{
			Message: "Failed to get user",
			Code:    authErrors.CodeDatabaseError,
			Error: pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "failed to get user").
				WithService(authErrors.ServiceName).
				WithDetail("user_id", userID),
		}
	}
	if user == nil {
		return &authErrors.AuthError{
			Message: "User not found",
			Code:    authErrors.CodeUserNotFound,
			Error: pkgErrors.New(authErrors.CodeUserNotFound, "user not found").
				WithService(authErrors.ServiceName).
				WithDetail("user_id", userID),
		}
	}

	if user.DeletedAt != nil {
		return &authErrors.AuthError{
			Message: "Account has already been deleted",
			Code:    authErrors.CodeAccountAlreadyDeleted,
			Error: pkgErrors.New(authErrors.CodeAccountAlreadyDeleted, "account already deleted").
				WithService(authErrors.ServiceName).
				WithDetail("user_id", userID),
		}
	}

	success, _, verifyErr := s.hashingService.VerifyPassword(ctx, password, user.PasswordHash)
	if verifyErr != nil {
		return &authErrors.AuthError{
			Message: "Password verification failed",
			Code:    authErrors.CodeInvalidCredentials,
			Error: pkgErrors.FromError(verifyErr, authErrors.CodeInvalidCredentials, "password verification failed").
				WithService(authErrors.ServiceName).
				WithDetail("user_id", userID),
		}
	}
	if !success {
		return &authErrors.AuthError{
			Message: "Invalid password",
			Code:    authErrors.CodeInvalidCredentials,
			Error: pkgErrors.New(authErrors.CodeInvalidCredentials, "invalid password").
				WithService(authErrors.ServiceName).
				WithDetail("user_id", userID),
		}
	}

	deleteErr := s.repo.SoftDeleteUser(ctx, userID)
	if deleteErr != nil {
		return &authErrors.AuthError{
			Message: "Failed to delete account",
			Code:    authErrors.CodeDatabaseError,
			Error: pkgErrors.FromError(deleteErr, pkgErrors.CodeDatabaseError, "failed to soft delete user").
				WithService(authErrors.ServiceName).
				WithDetail("user_id", userID),
		}
	}

	_ = s.sessionRepo.RevokeAllUserSessions(ctx, userID, "account_deleted")

	s.log.Info("Account deleted successfully",
		logger.String("service", authErrors.ServiceName),
		logger.String("user_id", userID),
	)
	return nil
}

// ============================================================================
// Device Trust
// ============================================================================

func (s *AuthService) IsDeviceTrusted(ctx context.Context, userID string, deviceID string) (bool, *authErrors.AuthError) {
	trusted, err := s.repo.IsDeviceTrusted(ctx, userID, deviceID)
	if err != nil {
		return false, &authErrors.AuthError{
			Message: "Failed to determine device trust",
			Code:    authErrors.CodeDatabaseError,
			Error:   err,
		}
	}
	return trusted, nil
}

func (s *AuthService) IsNewLocation(ctx context.Context, userID string, city string) (bool, *authErrors.AuthError) {
	isNew, err := s.repo.IsNewLocation(ctx, userID, city)
	if err != nil {
		return false, &authErrors.AuthError{
			Message: "Failed to determine login location",
			Code:    authErrors.CodeDatabaseError,
			Error:   err,
		}
	}
	return isNew, nil
}
