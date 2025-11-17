# Development Guidelines

Coding standards, best practices, and conventions for the Echo Backend project.

## Table of Contents

- [Code Style](#code-style)
- [Project Structure](#project-structure)
- [Design Patterns](#design-patterns)
- [Error Handling](#error-handling)
- [Testing](#testing)
- [Configuration](#configuration)
- [Database](#database)
- [Security](#security)
- [Performance](#performance)
- [Documentation](#documentation)
- [Git Workflow](#git-workflow)

## Code Style

### Go Standards

Follow the official [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments) and [Effective Go](https://golang.org/doc/effective_go).

**Formatting**:
```bash
# Format all code
gofmt -w .

# Or use goimports (preferred)
goimports -w .

# Run from project root
make fmt
```

**Linting**:
```bash
# Install golangci-lint
brew install golangci-lint

# Run linter
golangci-lint run

# Or use make
make lint
```

### Naming Conventions

**Variables**:
```go
// Good - descriptive, camelCase
var userCount int
var isAuthenticated bool
var maxRetryAttempts = 3

// Bad - unclear, inconsistent
var uc int
var auth bool
var MAX_RETRY = 3
```

**Functions**:
```go
// Good - verb-based, clear intent
func GetUserByID(id string) (*User, error)
func ValidatePassword(password string) error
func SendNotification(userID string, message string) error

// Bad - unclear, noun-based
func User(id string) (*User, error)
func Password(pwd string) error
func Notify(uid string, msg string) error
```

**Types**:
```go
// Good - descriptive nouns
type UserRepository interface { }
type AuthService struct { }
type MessageHandler struct { }

// Bad - unclear abbreviations
type UsrRepo interface { }
type AS struct { }
type MsgHndlr struct { }
```

**Constants**:
```go
// Good - descriptive, PascalCase
const (
    MaxLoginAttempts     = 5
    SessionExpiryMinutes = 15
    DefaultPageSize      = 20
)

// Bad - unclear, SCREAMING_CASE
const (
    MAX_LOGIN = 5
    SESS_EXP  = 15
    DEF_SIZE  = 20
)
```

### File Organization

**Package Structure**:
```go
// 1. Package declaration
package handler

// 2. Imports (grouped: stdlib, external, internal)
import (
    "context"
    "fmt"
    "net/http"

    "github.com/gorilla/mux"
    "go.uber.org/zap"

    "echo-backend/shared/pkg/logger"
    "echo-backend/shared/server/response"
)

// 3. Constants
const (
    DefaultTimeout = 30 * time.Second
)

// 4. Types
type UserHandler struct {
    service UserService
    logger  logger.Logger
}

// 5. Constructors
func NewUserHandler(service UserService, logger logger.Logger) *UserHandler {
    return &UserHandler{
        service: service,
        logger:  logger,
    }
}

// 6. Methods
func (h *UserHandler) GetUser(w http.ResponseWriter, r *http.Request) {
    // Implementation
}
```

### Comments

**Package Comments**:
```go
// Package handler provides HTTP handlers for user-related operations.
// It handles user registration, authentication, profile management,
// and other user-centric API endpoints.
package handler
```

**Function Comments**:
```go
// GetUserByID retrieves a user from the database by their unique identifier.
// It returns an error if the user is not found or if a database error occurs.
//
// Parameters:
//   - ctx: Request context for cancellation and deadlines
//   - id: User's unique identifier (UUID)
//
// Returns:
//   - *User: The user object if found
//   - error: Error if user not found or database error
func GetUserByID(ctx context.Context, id string) (*User, error) {
    // Implementation
}
```

**Inline Comments**:
```go
// Good - explains WHY
// Hash password using Argon2id for better security against GPU attacks
passwordHash, err := hashingService.Hash(password)

// Bad - explains WHAT (code already shows this)
// Call Hash function
passwordHash, err := hashingService.Hash(password)
```

## Project Structure

### Standard Service Structure

Every service MUST follow this structure:

```
services/<service-name>/
├── cmd/
│   └── server/
│       └── main.go              # Entry point
├── internal/
│   ├── config/
│   │   ├── config.go           # Config structs
│   │   ├── loader.go           # Config loading
│   │   └── validator.go        # Config validation
│   ├── handler/
│   │   ├── handler.go          # Handler setup
│   │   ├── user.go             # User handlers
│   │   └── auth.go             # Auth handlers
│   ├── service/
│   │   ├── builder.go          # Service builder
│   │   ├── user.go             # User service
│   │   └── auth.go             # Auth service
│   ├── repo/
│   │   ├── user.go             # User repository
│   │   └── auth.go             # Auth repository
│   ├── model/
│   │   ├── user.go             # User model
│   │   └── session.go          # Session model
│   └── health/
│       └── checkers.go         # Health checkers
├── configs/
│   ├── config.yaml             # Base configuration
│   ├── config.dev.yaml         # Dev overrides
│   └── config.prod.yaml        # Prod overrides
├── api/
│   └── v1/                     # API version 1
├── Dockerfile                  # Production build
├── Dockerfile.dev              # Dev build (hot reload)
├── go.mod                      # Module definition
└── go.sum                      # Dependency checksums
```

### Layer Responsibilities

**Handler Layer** (`internal/handler/`):
- HTTP request/response handling
- Input validation
- Call service layer
- Return standardized responses

**Service Layer** (`internal/service/`):
- Business logic
- Orchestrate multiple repositories
- Transaction management
- Caching logic

**Repository Layer** (`internal/repo/`):
- Database access
- Simple CRUD operations
- No business logic

**Model Layer** (`internal/model/`):
- Data structures
- Implement `Model` interface (TableName, PrimaryKey)
- Validation tags

## Design Patterns

### Builder Pattern (Required)

**ALL services MUST use the Builder pattern**:

```go
// service/builder.go
type AuthServiceBuilder struct {
    repo           AuthRepository       // Required
    tokenService   TokenService         // Required
    hashingService HashingService       // Required
    cache          cache.Cache          // Optional
    config         *AuthConfig          // Required
    logger         logger.Logger        // Required
}

func NewAuthServiceBuilder() *AuthServiceBuilder {
    return &AuthServiceBuilder{}
}

func (b *AuthServiceBuilder) WithRepo(repo AuthRepository) *AuthServiceBuilder {
    b.repo = repo
    return b
}

func (b *AuthServiceBuilder) WithTokenService(ts TokenService) *AuthServiceBuilder {
    b.tokenService = ts
    return b
}

// ... other With methods ...

func (b *AuthServiceBuilder) Build() *AuthService {
    // Validate required dependencies
    if b.repo == nil {
        panic("AuthRepository is required")
    }
    if b.tokenService == nil {
        panic("TokenService is required")
    }
    if b.hashingService == nil {
        panic("HashingService is required")
    }
    if b.config == nil {
        panic("AuthConfig is required")
    }
    if b.logger == nil {
        panic("Logger is required")
    }

    return &AuthService{
        repo:           b.repo,
        tokenService:   b.tokenService,
        hashingService: b.hashingService,
        cache:          b.cache,
        config:         b.config,
        logger:         b.logger,
    }
}
```

**Usage**:
```go
authService := service.NewAuthServiceBuilder().
    WithRepo(authRepo).
    WithTokenService(tokenService).
    WithHashingService(hashingService).
    WithCache(cacheClient).
    WithConfig(&cfg.Auth).
    WithLogger(log).
    Build()
```

### Interface-Based Design (Required)

**Define interfaces for all dependencies**:

```go
// Repository interface
type AuthRepository interface {
    CreateUser(ctx context.Context, user *model.User) error
    FindUserByPhone(ctx context.Context, phone string) (*model.User, error)
    UpdateUser(ctx context.Context, user *model.User) error
    DeleteUser(ctx context.Context, userID string) error
}

// Service interface
type AuthService interface {
    Register(ctx context.Context, req RegisterRequest) (*RegisterResponse, error)
    Login(ctx context.Context, req LoginRequest) (*LoginResponse, error)
    VerifyOTP(ctx context.Context, req VerifyOTPRequest) error
    Logout(ctx context.Context, sessionID string) error
}
```

**Benefits**:
- Testability via mocking
- Dependency inversion
- Flexibility to swap implementations

### Repository Pattern (Required)

**Thin data access layer**:

```go
type userRepository struct {
    db database.Database
}

func NewUserRepository(db database.Database) UserRepository {
    return &userRepository{db: db}
}

func (r *userRepository) CreateUser(ctx context.Context, user *model.User) error {
    return r.db.Create(ctx, user)
}

func (r *userRepository) FindUserByID(ctx context.Context, id string) (*model.User, error) {
    user := &model.User{}
    err := r.db.FindByID(ctx, user, id)
    if err != nil {
        return nil, err
    }
    return user, nil
}

// Complex queries
func (r *userRepository) FindUsersByFilter(ctx context.Context, filter UserFilter) ([]*model.User, error) {
    query := "SELECT * FROM users.profiles WHERE 1=1"
    args := []interface{}{}

    if filter.Name != "" {
        query += " AND display_name ILIKE $1"
        args = append(args, "%"+filter.Name+"%")
    }

    if filter.Status != "" {
        query += " AND status = $2"
        args = append(args, filter.Status)
    }

    query += " ORDER BY created_at DESC LIMIT $3 OFFSET $4"
    args = append(args, filter.Limit, filter.Offset)

    users := []*model.User{}
    err := r.db.FindMany(ctx, &users, query, args...)
    return users, err
}
```

## Error Handling

### Error Types

**Use custom error types**:

```go
// shared/pkg/errors/errors.go
type AppError interface {
    error
    Code() string
    Type() ErrorType
    Context() map[string]interface{}
}

type ErrorType string

const (
    ErrorTypeValidation     ErrorType = "validation"
    ErrorTypeNotFound       ErrorType = "not_found"
    ErrorTypeUnauthorized   ErrorType = "unauthorized"
    ErrorTypeInternal       ErrorType = "internal"
    ErrorTypeConflict       ErrorType = "conflict"
)

func NewValidationError(message string, fields map[string]string) AppError {
    return &appError{
        code:    "VALIDATION_ERROR",
        errType: ErrorTypeValidation,
        message: message,
        context: map[string]interface{}{"fields": fields},
    }
}
```

### Error Wrapping

**Always wrap errors with context**:

```go
// Good
user, err := r.repo.FindUserByPhone(ctx, phone)
if err != nil {
    return nil, fmt.Errorf("failed to find user by phone %s: %w", phone, err)
}

// Bad
user, err := r.repo.FindUserByPhone(ctx, phone)
if err != nil {
    return nil, err
}
```

### Error Handling in Handlers

```go
func (h *UserHandler) GetUser(w http.ResponseWriter, r *http.Request) {
    userID := router.Param(r, "id")

    // Input validation
    if userID == "" {
        response.BadRequest(w, "USER_ID_REQUIRED", "User ID is required")
        return
    }

    // Service call
    user, err := h.service.GetUser(r.Context(), userID)
    if err != nil {
        // Check error type
        if errors.Is(err, ErrUserNotFound) {
            response.NotFound(w, "USER_NOT_FOUND", "User not found")
            return
        }

        h.logger.Error("Failed to get user",
            logger.String("user_id", userID),
            logger.Error(err),
        )
        response.InternalServerError(w, "INTERNAL_ERROR", "An error occurred")
        return
    }

    response.OK(w, user)
}
```

## Testing

### Test Structure

```go
// service/auth_test.go
package service

import (
    "context"
    "testing"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/mock"
)

func TestAuthService_Register(t *testing.T) {
    // Arrange
    ctx := context.Background()
    mockRepo := new(MockAuthRepository)
    mockTokenService := new(MockTokenService)

    service := NewAuthServiceBuilder().
        WithRepo(mockRepo).
        WithTokenService(mockTokenService).
        // ... other dependencies
        Build()

    req := RegisterRequest{
        Phone:    "+1234567890",
        Password: "SecurePass123!",
        Name:     "John Doe",
    }

    mockRepo.On("FindUserByPhone", ctx, req.Phone).Return(nil, ErrUserNotFound)
    mockRepo.On("CreateUser", ctx, mock.AnythingOfType("*model.User")).Return(nil)

    // Act
    resp, err := service.Register(ctx, req)

    // Assert
    assert.NoError(t, err)
    assert.NotNil(t, resp)
    assert.Equal(t, req.Phone, resp.Phone)
    mockRepo.AssertExpectations(t)
}

func TestAuthService_Register_DuplicatePhone(t *testing.T) {
    // Arrange
    ctx := context.Background()
    mockRepo := new(MockAuthRepository)

    service := NewAuthServiceBuilder().
        WithRepo(mockRepo).
        // ... other dependencies
        Build()

    req := RegisterRequest{Phone: "+1234567890"}

    existingUser := &model.User{Phone: req.Phone}
    mockRepo.On("FindUserByPhone", ctx, req.Phone).Return(existingUser, nil)

    // Act
    resp, err := service.Register(ctx, req)

    // Assert
    assert.Error(t, err)
    assert.Nil(t, resp)
    assert.Contains(t, err.Error(), "phone already exists")
}
```

### Table-Driven Tests

```go
func TestValidatePhone(t *testing.T) {
    tests := []struct {
        name    string
        phone   string
        wantErr bool
    }{
        {"valid US phone", "+12345678901", false},
        {"valid international", "+442071234567", false},
        {"missing plus", "12345678901", true},
        {"too short", "+1234", true},
        {"invalid characters", "+123-456-7890", true},
        {"empty", "", true},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := ValidatePhone(tt.phone)
            if tt.wantErr {
                assert.Error(t, err)
            } else {
                assert.NoError(t, err)
            }
        })
    }
}
```

### Mocking

```go
// Use testify/mock for interface mocking
type MockAuthRepository struct {
    mock.Mock
}

func (m *MockAuthRepository) CreateUser(ctx context.Context, user *model.User) error {
    args := m.Called(ctx, user)
    return args.Error(0)
}

func (m *MockAuthRepository) FindUserByPhone(ctx context.Context, phone string) (*model.User, error) {
    args := m.Called(ctx, phone)
    if args.Get(0) == nil {
        return nil, args.Error(1)
    }
    return args.Get(0).(*model.User), args.Error(1)
}
```

## Configuration

### Configuration Structure

```go
// internal/config/config.go
type Config struct {
    Service       ServiceConfig       `mapstructure:"service"`
    Server        ServerConfig        `mapstructure:"server"`
    Database      DatabaseConfig      `mapstructure:"database"`
    Cache         CacheConfig         `mapstructure:"cache"`
    Auth          AuthConfig          `mapstructure:"auth"`
    Security      SecurityConfig      `mapstructure:"security"`
    Logging       LoggingConfig       `mapstructure:"logging"`
    Observability ObservabilityConfig `mapstructure:"observability"`
    Shutdown      ShutdownConfig      `mapstructure:"shutdown"`
}

type ServiceConfig struct {
    Name        string `mapstructure:"name"`
    Version     string `mapstructure:"version"`
    Environment string `mapstructure:"environment"`
}

type ServerConfig struct {
    Port         int           `mapstructure:"port"`
    Host         string        `mapstructure:"host"`
    ReadTimeout  time.Duration `mapstructure:"read_timeout"`
    WriteTimeout time.Duration `mapstructure:"write_timeout"`
    IdleTimeout  time.Duration `mapstructure:"idle_timeout"`
}
```

### Environment Variable Interpolation

```yaml
# configs/config.yaml
database:
  host: ${DB_HOST:localhost}
  port: ${DB_PORT:5432}
  name: ${DB_NAME:echo}
  user: ${DB_USER:echo}
  password: ${DB_PASSWORD:echo}
  ssl_mode: ${DB_SSL_MODE:disable}
```

### Validation

```go
// internal/config/validator.go
func (c *Config) ValidateAndSetDefaults() error {
    // Service validation
    if c.Service.Name == "" {
        return fmt.Errorf("service.name is required")
    }

    // Server validation
    if c.Server.Port <= 0 || c.Server.Port > 65535 {
        return fmt.Errorf("server.port must be between 1 and 65535")
    }

    // Set defaults
    if c.Server.ReadTimeout == 0 {
        c.Server.ReadTimeout = 30 * time.Second
    }

    if c.Logging.Level == "" {
        c.Logging.Level = "info"
    }

    return nil
}
```

## Database

### Model Definition

```go
// internal/model/user.go
type User struct {
    ID           uuid.UUID  `db:"id" json:"id"`
    Phone        string     `db:"phone" json:"phone"`
    PasswordHash string     `db:"password_hash" json:"-"`
    Verified     bool       `db:"verified" json:"verified"`
    Locked       bool       `db:"locked" json:"locked"`
    CreatedAt    time.Time  `db:"created_at" json:"created_at"`
    UpdatedAt    time.Time  `db:"updated_at" json:"updated_at"`
    DeletedAt    *time.Time `db:"deleted_at" json:"deleted_at,omitempty"`
}

// Implement Model interface
func (u *User) TableName() string {
    return "auth.users"
}

func (u *User) PrimaryKey() string {
    return "id"
}
```

### Transactions

```go
func (s *AuthService) Register(ctx context.Context, req RegisterRequest) error {
    return s.db.WithTransaction(ctx, func(tx database.Transaction) error {
        // Create user
        user := &model.User{
            ID:           uuid.New(),
            Phone:        req.Phone,
            PasswordHash: hashedPassword,
        }
        if err := tx.Create(ctx, user); err != nil {
            return fmt.Errorf("failed to create user: %w", err)
        }

        // Create profile
        profile := &model.Profile{
            UserID:      user.ID,
            DisplayName: req.Name,
        }
        if err := tx.Create(ctx, profile); err != nil {
            return fmt.Errorf("failed to create profile: %w", err)
        }

        return nil
    })
}
```

### Migrations

```sql
-- database/schemas/auth/20240115_add_user_roles.up.sql
-- Add role column
ALTER TABLE auth.users ADD COLUMN role VARCHAR(50) DEFAULT 'user' NOT NULL;

-- Create index
CREATE INDEX idx_users_role ON auth.users(role) WHERE deleted_at IS NULL;

-- Add check constraint
ALTER TABLE auth.users ADD CONSTRAINT chk_user_role
    CHECK (role IN ('user', 'admin', 'moderator'));
```

```sql
-- database/schemas/auth/20240115_add_user_roles.down.sql
-- Remove check constraint
ALTER TABLE auth.users DROP CONSTRAINT IF EXISTS chk_user_role;

-- Remove index
DROP INDEX IF EXISTS auth.idx_users_role;

-- Remove column
ALTER TABLE auth.users DROP COLUMN IF EXISTS role;
```

## Security

### Password Hashing

```go
// ALWAYS use Argon2id (or bcrypt as fallback)
passwordHash, err := hashingService.Hash(password)

// NEVER store plain text passwords
// NEVER use weak hashing (MD5, SHA1)
```

### JWT Validation

```go
// Validate token in middleware
func AuthMiddleware(tokenService TokenService) Middleware {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            // Extract token
            authHeader := r.Header.Get("Authorization")
            if authHeader == "" {
                response.Unauthorized(w, "MISSING_TOKEN", "Authorization header required")
                return
            }

            // Validate token
            claims, err := tokenService.Validate(authHeader)
            if err != nil {
                response.Unauthorized(w, "INVALID_TOKEN", "Invalid or expired token")
                return
            }

            // Store claims in context
            ctx := context.WithValue(r.Context(), "user_id", claims.UserID)
            next.ServeHTTP(w, r.WithContext(ctx))
        })
    }
}
```

### Input Validation

```go
// Validate all user input
type RegisterRequest struct {
    Phone    string `json:"phone" validate:"required,phone"`
    Password string `json:"password" validate:"required,min=8,max=128"`
    Name     string `json:"name" validate:"required,min=2,max=100"`
}

func (r *RegisterRequest) Validate() error {
    validate := validator.New()

    // Register custom validators
    validate.RegisterValidation("phone", validatePhone)

    if err := validate.Struct(r); err != nil {
        return fmt.Errorf("validation failed: %w", err)
    }

    return nil
}
```

### SQL Injection Prevention

```go
// ALWAYS use parameterized queries
// Good
query := "SELECT * FROM users WHERE phone = $1 AND deleted_at IS NULL"
err := db.FindOne(ctx, &user, query, phone)

// Bad - SQL injection vulnerable
query := fmt.Sprintf("SELECT * FROM users WHERE phone = '%s'", phone)
err := db.FindOne(ctx, &user, query)
```

## Performance

### Database Optimization

```go
// Use connection pooling
database:
  max_open_connections: 25
  max_idle_connections: 10
  connection_lifetime: 5m
  connection_idle_timeout: 10m

// Use indexes for frequent queries
CREATE INDEX idx_users_phone ON auth.users(phone) WHERE deleted_at IS NULL;
CREATE INDEX idx_messages_conversation ON messages.messages(conversation_id, created_at DESC);

// Use pagination
func (r *messageRepository) FindMessages(ctx context.Context, filter MessageFilter) ([]*model.Message, error) {
    query := `
        SELECT * FROM messages.messages
        WHERE conversation_id = $1
        AND deleted_at IS NULL
        ORDER BY created_at DESC
        LIMIT $2 OFFSET $3
    `
    messages := []*model.Message{}
    err := r.db.FindMany(ctx, &messages, query, filter.ConversationID, filter.Limit, filter.Offset)
    return messages, err
}
```

### Caching Strategy

```go
// Cache frequently accessed data
func (s *UserService) GetUser(ctx context.Context, userID string) (*model.User, error) {
    // Try cache first
    cacheKey := fmt.Sprintf("user:%s", userID)
    cached, err := s.cache.Get(ctx, cacheKey)
    if err == nil && cached != nil {
        user := &model.User{}
        if err := json.Unmarshal(cached, user); err == nil {
            return user, nil
        }
    }

    // Fallback to database
    user, err := s.repo.FindByID(ctx, userID)
    if err != nil {
        return nil, err
    }

    // Cache result
    data, _ := json.Marshal(user)
    _ = s.cache.Set(ctx, cacheKey, data, 1*time.Hour)

    return user, nil
}
```

### Context Timeouts

```go
// Set timeouts for operations
func (h *UserHandler) GetUser(w http.ResponseWriter, r *http.Request) {
    ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
    defer cancel()

    user, err := h.service.GetUser(ctx, userID)
    // ...
}
```

## Documentation

### API Documentation

```go
// @Summary      Register new user
// @Description  Register a new user with phone number and password
// @Tags         authentication
// @Accept       json
// @Produce      json
// @Param        request body RegisterRequest true "Registration details"
// @Success      201 {object} RegisterResponse
// @Failure      400 {object} response.ErrorResponse
// @Failure      409 {object} response.ErrorResponse
// @Router       /api/v1/auth/register [post]
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
    // Implementation
}
```

### README Files

Each service should have a README:
```markdown
# Auth Service

Authentication and authorization service.

## Endpoints

- `POST /register` - Register new user
- `POST /login` - User login
- `POST /verify-otp` - Verify OTP
- `POST /logout` - User logout

## Configuration

See `configs/config.yaml`

## Development

```bash
make auth-up
make auth-logs
```
```

## Git Workflow

### Branch Naming

```bash
# Feature branches
git checkout -b feature/user-authentication
git checkout -b feature/websocket-messaging

# Bug fixes
git checkout -b fix/session-expiry
git checkout -b fix/database-connection

# Refactoring
git checkout -b refactor/auth-service
git checkout -b refactor/middleware-chain

# Documentation
git checkout -b docs/api-documentation
git checkout -b docs/architecture-guide
```

### Commit Messages

Follow [Conventional Commits](https://www.conventionalcommits.org/):

```bash
# Feature
git commit -m "feat(auth): add phone-first authentication"

# Bug fix
git commit -m "fix(message): resolve WebSocket connection leak"

# Refactor
git commit -m "refactor(user): extract profile service to separate file"

# Documentation
git commit -m "docs(readme): add quick start guide"

# Performance
git commit -m "perf(database): add index on users.phone column"

# Test
git commit -m "test(auth): add unit tests for registration flow"

# Build
git commit -m "build(docker): update Go version to 1.21"
```

### Pull Request Template

```markdown
## Description
Brief description of changes

## Type of Change
- [ ] Bug fix
- [ ] New feature
- [ ] Breaking change
- [ ] Documentation update
- [ ] Refactoring

## Changes Made
- Change 1
- Change 2
- Change 3

## Testing
- [ ] Unit tests pass
- [ ] Integration tests pass
- [ ] Manual testing completed

## Checklist
- [ ] Code follows style guidelines
- [ ] Self-review completed
- [ ] Comments added for complex logic
- [ ] Documentation updated
- [ ] No new warnings generated
```

---

**Last Updated**: January 2025
**Version**: 1.0.0
