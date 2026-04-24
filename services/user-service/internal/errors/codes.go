package errors

const (
	ServiceName = "user-service"
)

// Error codes
const (
	// User errors
	ErrCodeUserNotFound        = "USER_NOT_FOUND"
	ErrCodeUserBlocked         = "USER_BLOCKED"
	ErrCodeUserAlreadyExists   = "USER_ALREADY_EXISTS"
	ErrCodeInvalidUserID       = "INVALID_USER_ID"
	ErrCodeInvalidUserData     = "INVALID_USER_DATA"
	ErrCodeUsernameUnavailable = "USERNAME_UNAVAILABLE"

	// Profile errors
	ErrCodeProfileNotFound     = "PROFILE_NOT_FOUND"
	ErrCodeInvalidProfileData  = "INVALID_PROFILE_DATA"
	ErrCodeProfileUpdateFailed = "PROFILE_UPDATE_FAILED"

	// Search errors
	ErrCodeSearchFailed       = "SEARCH_FAILED"
	ErrCodeInvalidSearchQuery = "INVALID_SEARCH_QUERY"

	// Database errors
	ErrCodeDatabaseError      = "DATABASE_ERROR"
	ErrCodeDatabaseConnection = "DATABASE_CONNECTION_ERROR"

	// Cache errors
	ErrCodeCacheError = "CACHE_ERROR"

	// General errors
	ErrCodeInternalError   = "INTERNAL_ERROR"
	ErrCodeInvalidRequest  = "INVALID_REQUEST"
	ErrCodeUnauthorized    = "UNAUTHORIZED"
	ErrCodeForbidden       = "FORBIDDEN"
	ErrCodeValidationError = "VALIDATION_ERROR"

	// Consumer errors
	ErrCodeConsumerStartFailed = "CONSUMER_START_FAILED"
	ErrCodeConsumerCloseFailed = "CONSUMER_CLOSE_FAILED"

	ErrCodeUsernameGenerationFailed  = "USERNAME_GENERATION_FAILED"
	ErrCodeProfileBlocked            = "PROFILE_BLOCKED"
	ErrCodeUserAlreadyBlocked        = "USER_ALREADY_BLOCKED"
	ErrCodeCannotBlockSelf           = "CANNOT_BLOCK_SELF"
	ErrCodeCannotRemoveCurrentDevice = "CANNOT_REMOVE_CURRENT_DEVICE"
)
