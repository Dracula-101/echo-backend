package search

import "fmt"

type ErrCode string

const (
	ErrCodeNotFound         ErrCode = "NOT_FOUND"
	ErrCodeInvalidRequest   ErrCode = "INVALID_REQUEST"
	ErrCodeIndexNotFound    ErrCode = "INDEX_NOT_FOUND"
	ErrCodeIndexExists      ErrCode = "INDEX_ALREADY_EXISTS"
	ErrCodeDocumentNotFound ErrCode = "DOCUMENT_NOT_FOUND"
	ErrCodeInternal         ErrCode = "INTERNAL"
	ErrCodeTimeout          ErrCode = "TIMEOUT"
	ErrCodeUnauthorized     ErrCode = "UNAUTHORIZED"
)

type SearchError struct {
	Code    ErrCode
	Message string
	Cause   error
}

func (e *SearchError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("search [%s]: %s: %v", e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("search [%s]: %s", e.Code, e.Message)
}

func (e *SearchError) Unwrap() error {
	return e.Cause
}

func newError(code ErrCode, message string, cause error) *SearchError {
	return &SearchError{Code: code, Message: message, Cause: cause}
}

func NewError(code ErrCode, msg string, cause error) *SearchError {
	return newError(code, msg, cause)
}

func ErrNotFound(msg string, cause error) *SearchError {
	return newError(ErrCodeNotFound, msg, cause)
}

func ErrInvalidRequest(msg string, cause error) *SearchError {
	return newError(ErrCodeInvalidRequest, msg, cause)
}

func ErrInternal(msg string, cause error) *SearchError {
	return newError(ErrCodeInternal, msg, cause)
}
