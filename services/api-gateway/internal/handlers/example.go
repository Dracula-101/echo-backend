package handlers

import (
	"fmt"
	"net/http"
	"time"

	"shared/server/response"
)

// ExampleUserResponse demonstrates the response structure
type ExampleUserResponse struct {
	ID        string    `json:"id"`
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ExampleSuccessHandler demonstrates a successful response with all features
func ExampleSuccessHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Simulate data
		user := ExampleUserResponse{
			ID:        "usr_123456",
			Username:  "johndoe",
			Email:     "john@example.com",
			CreatedAt: time.Now().Add(-24 * time.Hour),
			UpdatedAt: time.Now(),
		}

		// Build response with full features
		builder := response.Success().
			WithContext(r.Context()).
			WithRequest(r).
			WithMessage("User retrieved successfully").
			WithData(user).
			EnableDebug() // Auto-disabled in production

		// Add HATEOAS links
		userID := "123456"
		builder.WithLinks(
			response.SelfLink(fmt.Sprintf("/api/v1/users/%s", userID)),
			response.UpdateLink(fmt.Sprintf("/api/v1/users/%s", userID)),
			response.DeleteLink(fmt.Sprintf("/api/v1/users/%s", userID)),
		)

		// Add debug info (only collected in development)
		if debug := builder.GetDebugCollector(); debug.IsEnabled() {
			// Simulate database query
			debug.AddQuery(response.QueryInfo{
				SQL:          "SELECT * FROM users WHERE id = $1",
				Duration:     2.5,
				RowsAffected: 1,
				Timestamp:    time.Now(),
				Slow:         false,
			})

			// Simulate cache operation
			debug.AddCacheOperation(response.CacheOperation{
				Operation: "GET",
				Key:       fmt.Sprintf("user:%s", userID),
				Hit:       true,
				Duration:  0.5,
				Timestamp: time.Now(),
			})

			// Set feature flags
			debug.SetFeatureFlag("user_profiles_v2", true)
			debug.SetFeatureFlag("email_verification", true)

			// Set timing for custom operations
			debug.SetTiming("business_logic", 10.2)
			debug.SetTiming("authorization", 5.1)
		}

		builder.OK(w)
	}
}

// ExamplePaginatedHandler demonstrates paginated list response
func ExamplePaginatedHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Simulate paginated data
		users := []ExampleUserResponse{
			{ID: "usr_1", Username: "user1", Email: "user1@example.com", CreatedAt: time.Now(), UpdatedAt: time.Now()},
			{ID: "usr_2", Username: "user2", Email: "user2@example.com", CreatedAt: time.Now(), UpdatedAt: time.Now()},
			{ID: "usr_3", Username: "user3", Email: "user3@example.com", CreatedAt: time.Now(), UpdatedAt: time.Now()},
		}

		// Create pagination info
		totalItems := int64(150)
		currentPage := 1
		pageSize := 20
		pagination := response.NewOffsetPagination(totalItems, currentPage, pageSize, len(users), true, false)

		// Build response
		response.Success().
			WithContext(r.Context()).
			WithRequest(r).
			WithMessage("Users retrieved successfully").
			WithData(users).
			WithPagination(pagination).
			WithLinks(
				response.SelfLink("/api/v1/users?page=1"),
				response.NextLink("/api/v1/users?page=2"),
			).
			OK(w)
	}
}

// ExampleValidationErrorHandler demonstrates validation error response
func ExampleValidationErrorHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Create field validation errors
		fieldErrors := []response.FieldError{
			response.RequiredFieldError("email"),
			response.MinLengthFieldError("username", 3),
			{
				Field:       "age",
				Message:     "age must be at least 18",
				Code:        "min_value",
				Value:       15,
				Constraints: "min=18",
			},
		}

		response.ValidationError(r.Context(), r, w, fieldErrors)
	}
}

// ExampleErrorWithDebugHandler demonstrates error response with debug info
func ExampleErrorWithDebugHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		builder := response.Error().
			WithContext(r.Context()).
			WithRequest(r).
			WithMessage("Failed to process request").
			EnableDebug()

		// Add debug info for error investigation
		if debug := builder.GetDebugCollector(); debug.IsEnabled() {
			// Log failed database query
			debug.AddQuery(response.QueryInfo{
				SQL:      "SELECT * FROM users WHERE email = $1",
				Duration: 150.5, // Slow query
				Error:    "connection timeout",
				Timestamp: time.Now(),
				Slow:     true,
			})

			// Log failed external call
			debug.AddExternalCall(response.ExternalCallInfo{
				Service:    "email-service",
				Method:     "POST",
				URL:        "https://api.email.com/send",
				StatusCode: 503,
				Duration:   5000, // 5 seconds
				Retries:    3,
				Error:      "service unavailable",
				Timestamp:  time.Now(),
			})

			// Record circuit state
			debug.AddExternalCall(response.ExternalCallInfo{
				Service:      "payment-service",
				Method:       "POST",
				URL:          "https://api.payment.com/charge",
				Duration:     100,
				CircuitState: "open",
				Error:        "circuit breaker is open",
				Timestamp:    time.Now(),
			})
		}

		builder.Send(w, http.StatusInternalServerError)
	}
}

// ExampleRateLimitHandler demonstrates rate limit error
func ExampleRateLimitHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		retryAfter := 60 // seconds
		response.RateLimitError(r.Context(), r, w, retryAfter)
	}
}

// ExampleCircuitOpenHandler demonstrates circuit breaker error
func ExampleCircuitOpenHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		response.CircuitOpenError(r.Context(), r, w, "payment-service")
	}
}

// ExampleHealthCheckHandler demonstrates a health check endpoint
func ExampleHealthCheckHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		healthData := map[string]interface{}{
			"status": "healthy",
			"checks": map[string]string{
				"database": "up",
				"cache":    "up",
				"queue":    "up",
			},
			"version": "1.0.0",
			"uptime":  "24h",
		}

		response.Success().
			WithContext(r.Context()).
			WithRequest(r).
			WithData(healthData).
			OK(w)
	}
}
