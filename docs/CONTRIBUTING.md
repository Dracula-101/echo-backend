# Contributing to Echo Backend

Thank you for your interest in contributing to Echo Backend! This document provides guidelines and instructions for contributing to the project.

## Table of Contents

- [Code of Conduct](#code-of-conduct)
- [Getting Started](#getting-started)
- [How to Contribute](#how-to-contribute)
- [Development Workflow](#development-workflow)
- [Pull Request Process](#pull-request-process)
- [Coding Standards](#coding-standards)
- [Testing Guidelines](#testing-guidelines)
- [Documentation](#documentation)
- [Community](#community)

## Code of Conduct

### Our Pledge

We are committed to providing a welcoming and inclusive environment for all contributors, regardless of experience level, gender identity, sexual orientation, disability, personal appearance, race, ethnicity, age, religion, or nationality.

### Our Standards

**Examples of positive behavior:**
- Using welcoming and inclusive language
- Being respectful of differing viewpoints and experiences
- Gracefully accepting constructive criticism
- Focusing on what is best for the community
- Showing empathy towards other community members

**Unacceptable behavior:**
- Harassment, trolling, or discriminatory comments
- Publishing others' private information without permission
- Personal or political attacks
- Any conduct which could reasonably be considered inappropriate in a professional setting

### Enforcement

Instances of unacceptable behavior may be reported to the project team. All complaints will be reviewed and investigated promptly and fairly.

## Getting Started

### Prerequisites

Before you begin, ensure you have the following installed:
- **Go** 1.21 or higher
- **Docker** and **Docker Compose**
- **Make**
- **Git**
- **PostgreSQL** (via Docker)
- **Redis** (via Docker)
- **Kafka** (via Docker)

### Fork and Clone

1. **Fork the repository** on GitHub
2. **Clone your fork**:
   ```bash
   git clone https://github.com/Dracula-101/echo-backend.git
   cd echo-backend
   ```
3. **Add upstream remote**:
   ```bash
   git remote add upstream https://github.com/original/echo-backend.git
   ```
4. **Verify remotes**:
   ```bash
   git remote -v
   # origin    https://github.com/Dracula-101/echo-backend.git (fetch)
   # origin    https://github.com/Dracula-101/echo-backend.git (push)
   # upstream  https://github.com/original/echo-backend.git (fetch)
   # upstream  https://github.com/original/echo-backend.git (push)
   ```

### Set Up Development Environment

```bash
# Start all services
make up

# Initialize database
make db-init

# Run migrations
make db-migrate

# Verify services are running
make health

# Run tests
make test
```

## How to Contribute

### Types of Contributions

We welcome the following types of contributions:

1. **Bug Reports** - Report bugs via GitHub Issues
2. **Feature Requests** - Suggest new features or improvements
3. **Bug Fixes** - Submit fixes for known bugs
4. **New Features** - Implement new functionality
5. **Documentation** - Improve or add documentation
6. **Code Refactoring** - Improve code quality without changing functionality
7. **Tests** - Add or improve test coverage
8. **Performance Improvements** - Optimize existing code

### Reporting Bugs

**Before submitting a bug report:**
- Check if the bug has already been reported in GitHub Issues
- Ensure you're using the latest version
- Collect relevant information (logs, environment, steps to reproduce)

**When submitting a bug report, include:**
- **Clear title** - Descriptive summary of the issue
- **Description** - Detailed explanation of the problem
- **Steps to reproduce** - Exact steps to recreate the bug
- **Expected behavior** - What should happen
- **Actual behavior** - What actually happens
- **Environment** - OS, Go version, Docker version
- **Logs** - Relevant log output
- **Screenshots** - If applicable

**Example Bug Report:**
```markdown
**Title:** WebSocket connection drops after 60 seconds

**Description:**
WebSocket connections in the message service are being dropped after exactly 60 seconds of inactivity, even when heartbeat messages are being sent.

**Steps to Reproduce:**
1. Connect to WebSocket: `ws://localhost:8083/ws`
2. Send heartbeat every 30 seconds
3. Wait for 60 seconds
4. Connection is dropped

**Expected Behavior:**
Connection should remain open as long as heartbeat messages are being sent.

**Actual Behavior:**
Connection drops after 60 seconds with error: "connection timeout"

**Environment:**
- OS: macOS 14.0
- Go: 1.21.5
- Docker: 24.0.6

**Logs:**
```
[ERROR] WebSocket connection timeout: user_id=abc123
```
```

### Requesting Features

**Before submitting a feature request:**
- Check if the feature has already been requested
- Consider if it aligns with project goals
- Think about implementation complexity

**When submitting a feature request, include:**
- **Clear title** - Concise description of the feature
- **Problem statement** - What problem does this solve?
- **Proposed solution** - How should it work?
- **Alternatives** - Other approaches you've considered
- **Use cases** - Real-world scenarios where this is useful
- **Mockups** - Visual representations (if applicable)

**Example Feature Request:**
```markdown
**Title:** Add message reactions (emoji reactions)

**Problem:**
Users cannot react to messages with emojis, which is a common feature in modern messaging apps.

**Proposed Solution:**
- Add reactions table: `message_id`, `user_id`, `emoji`, `created_at`
- Add REST endpoint: `POST /api/v1/messages/:id/reactions`
- Broadcast reaction events via WebSocket
- Limit to 5 different reactions per user per message

**Alternatives:**
- Simple like/dislike system (too limited)
- Custom stickers (too complex for MVP)

**Use Cases:**
- Quick acknowledgment without sending a message
- Expressing emotion in group chats
- Voting/polling in conversations
```

## Development Workflow

### 1. Create a Branch

```bash
# Update your local main branch
git checkout main
git pull upstream main

# Create a new branch
git checkout -b feature/message-reactions
# or
git checkout -b fix/websocket-timeout
# or
git checkout -b docs/api-documentation
```

**Branch Naming Convention:**
- `feature/` - New features
- `fix/` - Bug fixes
- `refactor/` - Code refactoring
- `docs/` - Documentation
- `test/` - Tests
- `perf/` - Performance improvements

### 2. Make Changes

**Follow these guidelines:**
- Write clean, readable code
- Follow [GUIDELINES.md](./GUIDELINES.md)
- Add comments for complex logic
- Keep commits focused and atomic
- Write meaningful commit messages

**Commit Message Format:**
```
<type>(<scope>): <subject>

<body>

<footer>
```

**Types:**
- `feat` - New feature
- `fix` - Bug fix
- `docs` - Documentation
- `refactor` - Code refactoring
- `test` - Tests
- `perf` - Performance
- `build` - Build system
- `ci` - CI/CD

**Examples:**
```bash
git commit -m "feat(auth): add email-based authentication with optional phone"
git commit -m "fix(message): resolve WebSocket connection leak"
git commit -m "docs(readme): add quick start guide"
git commit -m "refactor(user): extract profile service to separate file"
git commit -m "test(auth): add unit tests for registration flow"
git commit -m "perf(database): add index on messages.conversation_id"
```

### 3. Test Your Changes

```bash
# Run all tests
make test

# Run tests for specific service
cd services/auth-service
go test -v ./...

# Run tests with coverage
go test -v -cover ./...

# Run integration tests
make test-integration

# Format code
make fmt

# Run linter
make lint
```

### 4. Push to Your Fork

```bash
# Push your branch
git push origin feature/message-reactions

# If you need to update your branch
git fetch upstream
git rebase upstream/main
git push origin feature/message-reactions --force
```

## Pull Request Process

### Before Submitting

**Checklist:**
- [ ] Code follows [GUIDELINES.md](./GUIDELINES.md)
- [ ] All tests pass (`make test`)
- [ ] Code is formatted (`make fmt`)
- [ ] Linter passes (`make lint`)
- [ ] Documentation updated (if needed)
- [ ] Commits are clean and well-described
- [ ] Branch is up to date with main

### Submitting a Pull Request

1. **Go to your fork** on GitHub
2. **Click "New Pull Request"**
3. **Select branches**:
   - Base: `main` (upstream repository)
   - Compare: `feature/message-reactions` (your branch)
4. **Fill out the PR template**:

```markdown
## Description
Add emoji reactions to messages. Users can now react to messages with emojis, and reactions are displayed in real-time via WebSocket.

## Type of Change
- [x] New feature
- [ ] Bug fix
- [ ] Breaking change
- [ ] Documentation update
- [ ] Refactoring

## Changes Made
- Created `message_reactions` table in database
- Added `POST /api/v1/messages/:id/reactions` endpoint
- Implemented WebSocket broadcasting for reactions
- Added reaction limit (5 per user per message)
- Updated message model to include reactions count

## Testing
- [x] Unit tests pass (97% coverage)
- [x] Integration tests pass
- [x] Manual testing completed
- [x] Tested with 1000+ concurrent WebSocket connections

## Screenshots
![Reactions UI](screenshots/reactions.png)

## Checklist
- [x] Code follows style guidelines
- [x] Self-review completed
- [x] Comments added for complex logic
- [x] Documentation updated (API docs, README)
- [x] No new warnings generated
- [x] Database migration files included

## Related Issues
Closes #42
```

5. **Click "Create Pull Request"**

### Review Process

**What to expect:**
1. **Automated checks** - CI/CD pipeline runs tests
2. **Code review** - Maintainers review your code
3. **Feedback** - You may receive comments/suggestions
4. **Revisions** - Make requested changes
5. **Approval** - Once approved, your PR will be merged

**Responding to feedback:**
```bash
# Make requested changes
git add .
git commit -m "refactor(message): optimize reaction query as requested"

# Push updates
git push origin feature/message-reactions
```

**After merge:**
```bash
# Update your local main
git checkout main
git pull upstream main

# Delete your feature branch
git branch -d feature/message-reactions
git push origin --delete feature/message-reactions
```

## Coding Standards

### Go Style Guide

Follow the official [Go Style Guide](https://go.dev/doc/effective_go) and [Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments).

**Key points:**
- Use `gofmt` for formatting
- Use `golangci-lint` for linting
- Follow naming conventions (camelCase for private, PascalCase for public)
- Write clear, concise comments
- Keep functions small and focused
- Use interfaces for dependencies

### Project-Specific Standards

See [GUIDELINES.md](./GUIDELINES.md) for:
- Builder pattern usage
- Interface-based design
- Error handling
- Configuration management
- Database patterns
- Security practices

### Code Review Checklist

**For reviewers:**
- [ ] Code is clear and readable
- [ ] Logic is correct
- [ ] Error handling is appropriate
- [ ] Tests are comprehensive
- [ ] No security vulnerabilities
- [ ] Performance is acceptable
- [ ] Documentation is updated
- [ ] Follows project conventions

**For contributors:**
- [ ] Self-reviewed the code
- [ ] Removed debug code
- [ ] No commented-out code
- [ ] No unnecessary dependencies
- [ ] Variable names are descriptive
- [ ] Complex logic is commented

## Testing Guidelines

### Unit Tests

```go
func TestAuthService_Register(t *testing.T) {
    // Arrange
    ctx := context.Background()
    mockRepo := new(MockAuthRepository)

    service := NewAuthServiceBuilder().
        WithRepo(mockRepo).
        Build()

    // Act
    result, err := service.Register(ctx, req)

    // Assert
    assert.NoError(t, err)
    assert.NotNil(t, result)
}
```

### Integration Tests

```go
func TestAuthIntegration(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping integration test")
    }

    // Test with real database
    db := setupTestDB(t)
    defer teardownTestDB(t, db)

    // Run tests
}
```

### Test Coverage

**Minimum requirements:**
- Unit tests: 70% coverage
- Critical paths: 90% coverage
- New features: 80% coverage

**Check coverage:**
```bash
go test -v -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

## Documentation

### Code Documentation

**Package comments:**
```go
// Package handler provides HTTP handlers for authentication.
// It handles user registration and login with email-based authentication.
// Session management and token generation are handled via JWT tokens.
package handler
```

**Function comments:**
```go
// Register handles user registration with email and password (phone is optional).
// It validates input, checks for duplicate emails, hashes the password using Argon2id,
// and creates the user record in the database.
//
// Parameters:
//   - ctx: Request context
//   - req: Registration request containing email, password, optional phone, and terms acceptance
//
// Returns:
//   - *RegisterResponse: Contains user_id, email, and email verification status
//   - error: Validation errors, duplicate email, or internal errors
func (h *AuthHandler) Register(ctx context.Context, req RegisterRequest) (*RegisterResponse, error)
```

### API Documentation

Update API documentation when adding/changing endpoints:
- Add Swagger/OpenAPI annotations
- Update Postman collection
- Update README examples

### User Documentation

Update user-facing documentation:
- README.md - Quick start, features
- USAGE.md - Developer guide
- ARCHITECTURE.md - System design
- GUIDELINES.md - Coding standards

## Community

### Communication Channels

- **GitHub Issues** - Bug reports, feature requests
- **GitHub Discussions** - Questions, ideas, general discussion
- **Pull Requests** - Code contributions, reviews

### Getting Help

**Before asking for help:**
1. Check existing documentation
2. Search GitHub Issues
3. Review closed issues and PRs

**When asking for help:**
- Provide context and background
- Share relevant code/logs
- Explain what you've tried
- Be specific about the problem

### Recognition

Contributors will be:
- Listed in CONTRIBUTORS.md
- Credited in release notes
- Recognized in project README

### Mentorship

**For first-time contributors:**
- Look for issues labeled `good first issue`
- Ask questions in GitHub Discussions
- Request guidance in pull request comments
- Pair with experienced contributors

## Common Contribution Scenarios

### Adding a New Service

1. Create service directory: `services/new-service/`
2. Follow standard structure (cmd, internal, configs)
3. Implement Builder pattern
4. Add Dockerfile and Dockerfile.dev
5. Update docker-compose files
6. Add Makefile targets
7. Create database schema
8. Update API Gateway routes
9. Write tests
10. Document in README

See [GUIDELINES.md](./GUIDELINES.md#adding-a-new-service) for details.

### Adding a New Endpoint

1. Define route in handler
2. Implement handler method
3. Implement service logic
4. Update repository (if needed)
5. Add tests (unit + integration)
6. Update API documentation
7. Test manually
8. Submit PR

### Fixing a Bug

1. Create issue (if not exists)
2. Write failing test that reproduces bug
3. Fix the bug
4. Ensure test passes
5. Add regression tests
6. Submit PR referencing issue

### Improving Documentation

1. Identify outdated/missing docs
2. Create issue or start directly
3. Make improvements
4. Ensure examples are tested
5. Check for broken links
6. Submit PR

## License

By contributing to Echo Backend, you agree that your contributions will be licensed under the MIT License.

---

**Thank you for contributing to Echo Backend!**

If you have questions, please open a GitHub Discussion or reach out to the maintainers.

**Last Updated**: January 2025
**Version**: 1.0.0
