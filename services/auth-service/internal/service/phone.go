package service

import (
	"auth-service/internal/domain"
	"auth-service/internal/error"
	authErrors "auth-service/internal/error"
	"context"
	"fmt"
	"math/rand"
	pkgErrors "shared/pkg/errors"
	"shared/pkg/logger"
	"time"
)

func (s *authService) IsPhoneTaken(ctx context.Context, phone string) (*domain.User, *error.AuthError) {
	s.log.Info("Checking if phone number is taken", logger.String("phone", phone))

	user, err := s.repo.GetUserByPhone(ctx, phone)
	if err != nil {
		return nil, &error.AuthError{
			Message: "Failed to check phone number existence",
			Code:    authErrors.CodeDatabaseError,
			Error: pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "failed to check phone number existence").
				WithService(authErrors.ServiceName).
				WithDetail("phone", phone),
		}
	}
	return user, nil
}

//Inserts auth.otp_verifications with sent_via = "sms", hashed OTP, 10-min expiry. Rate-limit: 3 sends per 10 min per phone.

func (s *authService) generateOtp(ctx context.Context, phone string) (otp string, hash *string, err *error.AuthError) {
	otp = fmt.Sprintf("%06d", rand.Intn(1000000)) // Generate a random 6-digit OTP
	// handle edge case where generated OTP is less than 6 digits (e.g., 000123)
	if len(otp) < 6 {
		otp = fmt.Sprintf("%06s", otp)
	}

	hash, errHash := s.hashingService.SimpleHash(ctx, otp)
	if errHash != nil {
		return "", nil, &error.AuthError{
			Message: "Failed to hash OTP",
			Code:    authErrors.CodeOtpHashingFailed,
			Error: pkgErrors.FromError(errHash, authErrors.CodeOtpHashingFailed, "failed to hash OTP").
				WithService(authErrors.ServiceName).
				WithDetail("phone", phone),
		}
	}
	return otp, hash, nil
}

func (s *authService) SendOtpVerification(ctx context.Context, userID string, phone string, ipAddress string, userAgent string) *error.AuthError {
	s.log.Info("Sending OTP verification", logger.String("phone", phone))

	// Check if phone number is already registered
	existingUser, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		return &error.AuthError{
			Message: "Failed to check phone number existence",
			Code:    authErrors.CodeDatabaseError,
			Error: pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "failed to check phone number existence").
				WithService(authErrors.ServiceName).
				WithDetail("phone", phone),
		}
	}
	if existingUser.PhoneVerified {
		return &error.AuthError{
			Message: "Phone number is already verified",
			Code:    authErrors.CodePhoneAlreadyVerified,
			Error: pkgErrors.New("phone number is already verified", authErrors.CodePhoneAlreadyVerified).
				WithService(authErrors.ServiceName).
				WithDetail("phone", phone),
		}
	}

	// get already existing OTP attempts for this user
	otpAttempts, err := s.repo.GetOTPVerification(ctx, userID)
	if err != nil {
		return &error.AuthError{
			Message: "Failed to retrieve OTP attempts",
			Code:    authErrors.CodeDatabaseError,
			Error: pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "failed to retrieve OTP attempts").
				WithService(authErrors.ServiceName).
				WithDetail("user_id", userID),
		}
	}

	// Rate limit: allow max 3 OTP sends per 10 minutes per phone number
	if len(otpAttempts) >= 3 {
		return &error.AuthError{
			Message: "OTP send limit exceeded. Please try again later.",
			Code:    authErrors.CodeOTPTooManyVerificationAttempts,
			Error: pkgErrors.New("OTP send limit exceeded. Please try again later.", authErrors.CodeOTPTooManyVerificationAttempts).
				WithService(authErrors.ServiceName).
				WithDetail("phone", phone),
		}
	}

	// Generate and send OTP
	otp, hash, otpErr := s.generateOtp(ctx, phone)
	if otpErr != nil {
		return otpErr
	}
	if hash == nil {
		return &error.AuthError{
			Message: "Failed to generate OTP hash",
			Code:    authErrors.CodeOtpHashingFailed,
			Error: pkgErrors.New("failed to generate OTP hash", authErrors.CodeOtpHashingFailed).
				WithService(authErrors.ServiceName).
				WithDetail("phone", phone),
		}
	}

	// Store hashed OTP in database with 10-minute expiry
	err = s.repo.CreateOTPVerification(ctx, userID, phone, otp, *hash, ipAddress, userAgent)
	if err != nil {
		return &error.AuthError{
			Message: "Failed to create OTP verification",
			Code:    authErrors.CodeDatabaseError,
			Error: pkgErrors.FromError(err, authErrors.CodeDatabaseError, "failed to create OTP verification").
				WithService(authErrors.ServiceName).
				WithDetail("phone", phone),
		}
	}

	// Send OTP via SMS
	smsErr := s.smsService.SendOTP(ctx, phone, otp)
	if smsErr != nil {
		// delete OTP verification record from database since SMS sending failed
		deleteErr := s.repo.DeleteOTPVerification(ctx, userID, phone, otp)
		if deleteErr != nil {
			s.log.Error("Failed to delete OTP verification after SMS send failure",
				logger.String("user_id", userID),
				logger.String("phone", phone),
				logger.Error(deleteErr),
			)
		}
		return &error.AuthError{
			Message: "Failed to send OTP",
			Code:    authErrors.CodeSmsSendFailed,
			Error: pkgErrors.New(authErrors.CodeSmsSendFailed, "failed to send OTP").
				WithService(authErrors.ServiceName).
				WithDetail("phone", phone),
		}
	}

	s.log.Info("OTP sent successfully", logger.String("phone", phone))
	return nil

}

func (s *authService) VerifyPhone(ctx context.Context, userID string, otp string) *error.AuthError {
	s.log.Info("Verifying phone number with OTP",
		logger.String("user_id", userID))

	// Retrieve OTP verification record from database
	otpRecords, err := s.repo.GetOTPVerification(ctx, userID)
	if err != nil {
		return &error.AuthError{
			Message: "Failed to retrieve OTP verification",
			Code:    authErrors.CodeDatabaseError,
			Error: pkgErrors.FromError(err, authErrors.CodeDatabaseError, "failed to retrieve OTP verification").
				WithService(authErrors.ServiceName).
				WithDetail("user_id", userID),
		}
	}
	if otpRecords == nil {
		return &error.AuthError{
			Message: "No OTP verification record found",
			Code:    authErrors.CodeOTPNotFound,
			Error: pkgErrors.New("no OTP verification record found", authErrors.CodeOTPNotFound).
				WithService(authErrors.ServiceName).
				WithDetail("user_id", userID),
		}
	}

	// check if OTP is correct and not expired
	if len(otpRecords) == 0 {
		return &error.AuthError{
			Message: "Invalid OTP",
			Code:    authErrors.CodeInvalidOTP,
			Error: pkgErrors.New("invalid OTP", authErrors.CodeInvalidOTP).
				WithService(authErrors.ServiceName).
				WithDetail("user_id", userID),
		}
	}
	if len(otpRecords) == 1 {
		otpRecord := otpRecords[0]
		if time.Now().After(otpRecord.ExpiresAt) {
			s.log.Warn("OTP has expired",
				logger.String("user_id", userID),
				logger.String("otp_hash", otpRecord.OTPHash),
			)
			return &error.AuthError{
				Message: "OTP has expired",
				Code:    authErrors.CodeOTPExpired,
				Error: pkgErrors.New("OTP has expired", authErrors.CodeOTPExpired).
					WithService(authErrors.ServiceName).
					WithDetail("user_id", userID),
			}
		}

		ok, _, errHash := s.hashingService.SimpleVerify(ctx, otp, otpRecord.OTPHash)
		if errHash != nil {
			s.log.Error("Failed to verify OTP hash",
				logger.String("user_id", userID),
				logger.String("otp_hash", otpRecord.OTPHash),
				logger.Error(errHash),
			)
			return &error.AuthError{
				Message: "Invalid OTP",
				Code:    authErrors.CodeInvalidOTP,
				Error: pkgErrors.FromError(errHash, authErrors.CodeInvalidOTP, "failed to compare OTP hash").
					WithService(authErrors.ServiceName).
					WithDetail("user_id", userID),
			}
		}
		if !ok {
			s.log.Warn("Invalid OTP provided for verification",
				logger.String("user_id", userID),
				logger.String("otp_hash", otpRecord.OTPHash),
			)
			return &error.AuthError{
				Message: "Invalid OTP",
				Code:    authErrors.CodeInvalidOTP,
				Error: pkgErrors.New("invalid OTP", authErrors.CodeInvalidOTP).
					WithService(authErrors.ServiceName).
					WithDetail("user_id", userID),
			}
		}
	}

	if len(otpRecords) > 1 {
		// loop through OTP records to find a match (handle case where multiple OTPs were generated due to retries)
		matchFound := false
		for _, otpRecord := range otpRecords {
			if time.Now().After(otpRecord.ExpiresAt) {
				continue // skip expired OTPs
			}
			ok, _, errHash := s.hashingService.SimpleVerify(ctx, otp, otpRecord.OTPHash)
			if errHash != nil {
				continue // skip OTPs that fail to verify (shouldn't happen but handle just in case)
			}
			if ok {
				matchFound = true
				break
			}
		}
		if !matchFound {
			s.log.Warn("No matching OTP found among multiple records",
				logger.String("user_id", userID),
				logger.Int("otp_record_count", len(otpRecords)),
			)
			return &error.AuthError{
				Message: "Invalid OTP",
				Code:    authErrors.CodeInvalidOTP,
				Error: pkgErrors.New("invalid OTP", authErrors.CodeInvalidOTP).
					WithService(authErrors.ServiceName).
					WithDetail("user_id", userID),
			}
		}
	}

	// Mark phone as verified in database
	err = s.repo.MarkPhoneVerified(ctx, userID)
	if err != nil {
		return &error.AuthError{
			Message: "Failed to mark phone as verified",
			Code:    authErrors.CodeDatabaseError,
			Error: pkgErrors.FromError(err, authErrors.CodeDatabaseError, "failed to mark phone as verified").
				WithService(authErrors.ServiceName).
				WithDetail("user_id", userID),
		}
	}

	s.log.Info("Phone number verified successfully", logger.String("user_id", userID))
	return nil
}
