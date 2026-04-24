package contacts

import (
	"context"
	"fmt"
	"strings"
	"time"

	"user-service/internal/domain"
	userErrors "user-service/internal/errors"

	"shared/pkg/database"
	"shared/pkg/database/postgres"
	dbModels "shared/pkg/database/postgres/models"
	pkgErrors "shared/pkg/errors"
	"shared/pkg/logger"
	"shared/pkg/utils"

	"github.com/lib/pq"
)

type contactRepository struct {
	db  database.Database
	log logger.Logger
}

func NewContactRepository(db database.Database, log logger.Logger) ContactRepository {
	if db == nil {
		panic("database is required for ContactRepository")
	}
	if log == nil {
		panic("logger is required for ContactRepository")
	}
	return &contactRepository{db: db, log: log}
}

func (r *contactRepository) GetContactIDs(ctx context.Context, userID string) ([]string, pkgErrors.AppError) {
	query := `
		SELECT contact_user_id FROM users.contacts
		WHERE user_id = $1
		AND status = 'active'`

	var rows []struct {
		ContactUserID string `db:"contact_user_id"`
	}
	if dbErr := r.db.FindMany(ctx, &rows, query, userID); dbErr != nil {
		r.log.Error("failed to get contact IDs",
			logger.String("user_id", userID),
			logger.Error(dbErr),
		)
		return nil, pkgErrors.FromError(dbErr, userErrors.ErrCodeDatabaseError, "failed to get contact IDs")
	}

	ids := make([]string, len(rows))
	for i, row := range rows {
		ids[i] = row.ContactUserID
	}
	return ids, nil
}

func (r *contactRepository) GetContacts(ctx context.Context, userID string, params GetContactsParams) ([]*domain.Contact, int, pkgErrors.AppError) {
	whereClauses := []string{"c.user_id = $1", "c.status = $2"}
	args := []any{userID, params.Status}
	argIdx := 3

	if params.Relationship != "" {
		switch strings.ToLower(params.Relationship) {
		case "favorite":
			whereClauses = append(whereClauses, "c.is_favorite = true")
		case "pinned":
			whereClauses = append(whereClauses, "c.is_pinned = true")
		case "archived":
			whereClauses = append(whereClauses, "c.is_archived = true")
		case "muted":
			whereClauses = append(whereClauses, "c.is_muted = true")
		default:
			whereClauses = append(whereClauses, fmt.Sprintf("c.relationship_type = $%d", argIdx))
			args = append(args, params.Relationship)
			argIdx++
		}
	}

	if params.GroupID != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("$%d = ANY(c.contact_groups)", argIdx))
		args = append(args, *params.GroupID)
		argIdx++
	}

	if params.Search != nil && strings.TrimSpace(*params.Search) != "" {
		whereClauses = append(whereClauses, fmt.Sprintf(`(
			LOWER(c.nickname) LIKE '%%' || LOWER($%d) || '%%'
			OR EXISTS (
				SELECT 1 FROM users.profiles p
				WHERE p.user_id = c.contact_user_id
				AND (
					LOWER(p.display_name) LIKE '%%' || LOWER($%d) || '%%'
					OR LOWER(p.username) LIKE '%%' || LOWER($%d) || '%%'
				)
			)
		)`, argIdx, argIdx, argIdx))
		args = append(args, strings.TrimSpace(*params.Search))
		argIdx++
	}

	query := fmt.Sprintf(`
		SELECT
			c.id, c.user_id, c.contact_user_id, c.status, c.relationship_type,
			c.nickname, c.notes, c.is_favorite, c.is_pinned, c.is_archived,
			c.is_muted, c.muted_until, c.contact_source, c.contact_groups,
			c.last_interaction_at, c.accepted_at, c.blocked_at, c.created_at, c.updated_at,
			p.display_name, p.username, p.avatar_url,
			us.online_status_visibility
		FROM users.contacts c
		JOIN users.profiles p ON p.user_id = c.contact_user_id
		LEFT JOIN users.settings us ON us.user_id = c.contact_user_id
		WHERE %s
		ORDER BY c.is_pinned DESC, c.is_favorite DESC, c.last_interaction_at DESC NULLS LAST, p.display_name ASC
		LIMIT $%d OFFSET $%d`,
		strings.Join(whereClauses, " AND "),
		argIdx,
		argIdx+1,
	)

	queryArgs := append([]any{}, args...)
	queryArgs = append(queryArgs, params.Limit, params.Offset)

	var rows []struct {
		ID                     string                    `db:"id"`
		UserID                 string                    `db:"user_id"`
		ContactUserID          string                    `db:"contact_user_id"`
		Status                 dbModels.ContactStatus    `db:"status"`
		RelationshipType       dbModels.RelationshipType `db:"relationship_type"`
		Nickname               *string                   `db:"nickname"`
		Notes                  *string                   `db:"notes"`
		IsFavorite             bool                      `db:"is_favorite"`
		IsPinned               bool                      `db:"is_pinned"`
		IsArchived             bool                      `db:"is_archived"`
		IsMuted                bool                      `db:"is_muted"`
		MutedUntil             *time.Time                `db:"muted_until"`
		ContactSource          *string                   `db:"contact_source"`
		ContactGroups          pq.StringArray            `db:"contact_groups"`
		LastInteractionAt      *time.Time                `db:"last_interaction_at"`
		AcceptedAt             *time.Time                `db:"accepted_at"`
		BlockedAt              *time.Time                `db:"blocked_at"`
		CreatedAt              *time.Time                `db:"created_at"`
		UpdatedAt              *time.Time                `db:"updated_at"`
		DisplayName            *string                   `db:"display_name"`
		Username               *string                   `db:"username"`
		AvatarURL              *string                   `db:"avatar_url"`
		OnlineStatusVisibility *string                   `db:"online_status_visibility"`
	}
	if dbErr := r.db.FindMany(ctx, &rows, query, queryArgs...); dbErr != nil {
		r.log.Error("failed to get contacts",
			logger.String("user_id", userID),
			logger.Error(dbErr),
		)
		return nil, 0, pkgErrors.FromError(dbErr, userErrors.ErrCodeDatabaseError, "failed to get contacts")
	}

	countQuery := fmt.Sprintf(`
		SELECT COUNT(*)
		FROM users.contacts c
		WHERE %s`, strings.Join(whereClauses, " AND "))

	var total int
	if dbErr := r.db.QueryRow(ctx, countQuery, args...).Scan(&total); dbErr != nil {
		r.log.Error("failed to count contacts",
			logger.String("user_id", userID),
			logger.Error(dbErr),
		)
		return nil, 0, pkgErrors.FromError(dbErr, userErrors.ErrCodeDatabaseError, "failed to count contacts")
	}

	contacts := make([]*domain.Contact, len(rows))
	for i, row := range rows {
		model := dbModels.Contact{
			ID:                row.ID,
			UserID:            row.UserID,
			ContactUserID:     row.ContactUserID,
			Status:            row.Status,
			RelationshipType:  row.RelationshipType,
			Nickname:          row.Nickname,
			Notes:             row.Notes,
			IsFavorite:        row.IsFavorite,
			IsPinned:          row.IsPinned,
			IsArchived:        row.IsArchived,
			IsMuted:           row.IsMuted,
			MutedUntil:        row.MutedUntil,
			ContactSource:     row.ContactSource,
			ContactGroups:     row.ContactGroups,
			LastInteractionAt: row.LastInteractionAt,
			AcceptedAt:        row.AcceptedAt,
			BlockedAt:         row.BlockedAt,
			CreatedAt:         row.CreatedAt,
			UpdatedAt:         row.UpdatedAt,
		}

		c := domain.NewContact(model)
		c.DisplayName = row.DisplayName
		c.Username = row.Username
		c.AvatarURL = row.AvatarURL
		c.OnlineStatusVisibility = row.OnlineStatusVisibility
		contacts[i] = c
	}

	return contacts, total, nil
}

func (r *contactRepository) GetPendingIncomingRequestsCount(ctx context.Context, userID string) (int, pkgErrors.AppError) {
	query := `
		SELECT COUNT(*)
		FROM users.contacts
		WHERE contact_user_id = $1
		AND status = 'pending'`

	var total int
	if dbErr := r.db.QueryRow(ctx, query, userID).Scan(&total); dbErr != nil {
		r.log.Error("failed to count pending incoming contact requests",
			logger.String("user_id", userID),
			logger.Error(dbErr),
		)
		return 0, pkgErrors.FromError(dbErr, userErrors.ErrCodeDatabaseError, "failed to count pending incoming contact requests")
	}

	return total, nil
}

func (r *contactRepository) GetContact(ctx context.Context, userID, contactID string) (*domain.Contact, pkgErrors.AppError) {
	c := &dbModels.Contact{}
	// For accept/reject flows, the authenticated user is the recipient of a pending request.
	dbErr := r.db.FindOne(ctx, c, `SELECT * FROM users.contacts WHERE contact_user_id = $1 AND id = $2`, userID, contactID)
	if dbErr != nil {
		if postgres.IsNotFoundError(dbErr) {
			return nil, nil
		}
		r.log.Error("failed to get contact",
			logger.String("contact_id", contactID),
			logger.Error(dbErr),
		)
		return nil, pkgErrors.FromError(dbErr, userErrors.ErrCodeDatabaseError, "failed to get contact")
	}
	return domain.NewContact(*c), nil
}

func (r *contactRepository) RemoveContact(ctx context.Context, contactID, userID string) pkgErrors.AppError {
	r.log.Info("Removing contact",
		logger.String("user_id", userID),
		logger.String("contact_id", contactID),
	)
	query := `SELECT id, user_id, contact_user_id FROM users.contacts WHERE id = $1 AND user_id = $2`
	c := &dbModels.Contact{}
	if dbErr := r.db.FindOne(ctx, c, query, contactID, userID); dbErr != nil {
		if postgres.IsNotFoundError(dbErr) {
			r.log.Info("Contact not found for removal",
				logger.String("user_id", userID),
				logger.String("contact_id", contactID),
			)
			return pkgErrors.New(userErrors.ErrCodeInvalidRequest, "Contact not found")
		}
		r.log.Error("Failed to find contact for removal",
			logger.String("user_id", userID),
			logger.String("contact_id", contactID),
			logger.Error(dbErr),
		)
		return pkgErrors.FromError(dbErr, userErrors.ErrCodeDatabaseError, "Failed to find contact for removal")
	}

	deleteQuery := `DELETE FROM users.contacts WHERE id = $1`
	if _, dbErr := r.db.Exec(ctx, deleteQuery, contactID); dbErr != nil {
		r.log.Error("Failed to delete contact",
			logger.String("user_id", userID),
			logger.String("contact_id", contactID),
			logger.Error(dbErr),
		)
		return pkgErrors.FromError(dbErr, userErrors.ErrCodeDatabaseError, "Failed to delete contact")
	}

	updateReciprocalQuery := `
		UPDATE users.contacts
		SET status = 'removed', updated_at = NOW()
		WHERE user_id = $1 AND contact_user_id = $2 AND status = 'active'`

	if _, dbErr := r.db.Exec(ctx, updateReciprocalQuery, c.ContactUserID, c.UserID); dbErr != nil {
		r.log.Error("Failed to update reciprocal contact relationship",
			logger.String("user_id", userID),
			logger.String("contact_id", contactID),
			logger.Error(dbErr),
		)
		return pkgErrors.FromError(dbErr, userErrors.ErrCodeDatabaseError, "Failed to update reciprocal contact relationship")
	}

	r.log.Info("Contact removed successfully",
		logger.String("user_id", userID),
		logger.String("contact_id", contactID),
	)
	return nil
}

func (r *contactRepository) GetPendingContactRequests(ctx context.Context, userID string) ([]*domain.Contact, pkgErrors.AppError) {
	query := `
		SELECT
			id, user_id, contact_user_id, status, relationship_type,
			nickname, notes, is_favorite, is_pinned, is_archived,
			is_muted, muted_until, contact_source, contact_groups,
			accepted_at, blocked_at, created_at, updated_at
		FROM users.contacts
		WHERE contact_user_id = $1
		AND status = 'pending'
		ORDER BY created_at DESC`

	var rows []dbModels.Contact
	if dbErr := r.db.FindMany(ctx, &rows, query, userID); dbErr != nil {
		r.log.Error("failed to get pending contact requests",
			logger.String("user_id", userID),
			logger.Error(dbErr),
		)
		return nil, pkgErrors.FromError(dbErr, userErrors.ErrCodeDatabaseError, "failed to get pending contact requests")
	}

	list := make([]*domain.Contact, len(rows))
	for i, row := range rows {
		list[i] = domain.NewContact(row)
	}
	return list, nil
}

func (r *contactRepository) ContactExists(ctx context.Context, userID, targetID string) (*domain.Contact, pkgErrors.AppError) {
	c := &dbModels.Contact{}
	dbErr := r.db.FindOne(ctx, c, `SELECT * FROM users.contacts WHERE user_id = $1 AND contact_user_id = $2`, userID, targetID)
	if dbErr != nil {
		if postgres.IsNotFoundError(dbErr) {
			return nil, nil
		}
		r.log.Error("failed to check if contact exists",
			logger.String("user_id", userID),
			logger.String("target_id", targetID),
			logger.Error(dbErr),
		)
		return nil, pkgErrors.FromError(dbErr, userErrors.ErrCodeDatabaseError, "failed to check if contact exists")
	}
	return domain.NewContact(*c), nil
}

func (r *contactRepository) CreateContactRequest(ctx context.Context, userID, targetID, message, source string) (*domain.Contact, pkgErrors.AppError) {
	c := &dbModels.Contact{
		UserID:           userID,
		ContactUserID:    targetID,
		RelationshipType: dbModels.RelationshipTypePending,
		Status:           dbModels.ContactStatusPending,
		ContactSource:    utils.Ptr(source),
		Notes:            utils.Ptr(message),
		CreatedAt:        utils.Ptr(time.Now()),
		UpdatedAt:        utils.Ptr(time.Now()),
	}
	id, dbErr := r.db.Insert(ctx, c)
	if dbErr != nil {
		r.log.Error("failed to create contact request",
			logger.String("user_id", userID),
			logger.String("target_id", targetID),
			logger.Error(dbErr),
		)
		return nil, pkgErrors.FromError(dbErr, userErrors.ErrCodeDatabaseError, "failed to create contact request")
	}
	c.ID = *id
	return domain.NewContact(*c), nil
}

func (r *contactRepository) AcceptContactRequest(ctx context.Context, contactID, userID string) (*domain.Contact, pkgErrors.AppError) {
	query := `
		WITH accepted_request AS (
			UPDATE users.contacts
			SET relationship_type = 'friend',
				status = 'active',
				accepted_at = NOW(),
				updated_at = NOW(),
				last_interaction_at = NOW()
			WHERE id = $1 AND contact_user_id = $2 AND status = 'pending'
			RETURNING user_id, contact_user_id
		)
		INSERT INTO users.contacts (user_id, contact_user_id, relationship_type, status, contact_source, accepted_at, created_at)
		SELECT contact_user_id, user_id, 'friend', 'active', 'contact_accept', NOW(), NOW()
		FROM accepted_request
		ON CONFLICT (user_id, contact_user_id) DO UPDATE
		SET relationship_type = 'friend', status = 'active', accepted_at = NOW(), updated_at = NOW()
		RETURNING *`

	c := &dbModels.Contact{}
	if dbErr := r.db.FindOneAndUpdate(ctx, c, query, contactID, userID); dbErr != nil {
		if postgres.IsNotFoundError(dbErr) {
			return nil, pkgErrors.New(userErrors.ErrCodeInvalidRequest, "Contact request not found or no longer pending")
		}
		r.log.Error("failed to accept contact request",
			logger.String("contact_id", contactID),
			logger.Error(dbErr),
		)
		return nil, pkgErrors.FromError(dbErr, userErrors.ErrCodeDatabaseError, "failed to accept contact request")
	}
	return domain.NewContact(*c), nil
}

func (r *contactRepository) DeclineContactRequest(ctx context.Context, contactID, userID string) pkgErrors.AppError {
	query := `
		UPDATE users.contacts
		SET relationship_type = 'pending',
		    status     = 'rejected',
		    updated_at = NOW(),
			last_interaction_at = NOW()
		WHERE id = $1
		AND contact_user_id = $2
		AND status = 'pending'
		RETURNING *`

	c := &dbModels.Contact{}
	if dbErr := r.db.FindOneAndUpdate(ctx, c, query, contactID, userID); dbErr != nil {
		if postgres.IsNotFoundError(dbErr) {
			return pkgErrors.New(userErrors.ErrCodeInvalidRequest, "Contact request not found or no longer pending")
		}
		r.log.Error("failed to decline contact request",
			logger.String("contact_id", contactID),
			logger.Error(dbErr),
		)
		return pkgErrors.FromError(dbErr, userErrors.ErrCodeDatabaseError, "failed to decline contact request")
	}
	return nil
}

func (r *contactRepository) UpdateContact(ctx context.Context, contactID, userID string, params UpdateContactParams) (*domain.Contact, pkgErrors.AppError) {
	setClauses := []string{"updated_at = NOW()"}
	args := []any{}
	argIdx := 1

	addParam := func(col string, val any) {
		setClauses = append(setClauses, fmt.Sprintf("%s = $%d", col, argIdx))
		args = append(args, val)
		argIdx++
	}

	if params.Nickname != nil {
		addParam("nickname", *params.Nickname)
	}
	if params.Notes != nil {
		addParam("notes", *params.Notes)
	}
	if params.IsFavorite != nil {
		addParam("is_favorite", *params.IsFavorite)
	}
	if params.IsPinned != nil {
		addParam("is_pinned", *params.IsPinned)
	}
	if params.IsArchived != nil {
		addParam("is_archived", *params.IsArchived)
	}
	if params.IsMuted != nil {
		addParam("is_muted", *params.IsMuted)
	}
	if params.MutedUntil != nil {
		addParam("muted_until", *params.MutedUntil)
	}
	if params.ContactGroups != nil {
		addParam("contact_groups", *params.ContactGroups)
	}

	args = append(args, contactID, userID)
	query := fmt.Sprintf(`
		UPDATE users.contacts SET %s
		WHERE id = $%d
		AND user_id = $%d
		AND status = 'active'
		RETURNING *`,
		strings.Join(setClauses, ", "), argIdx, argIdx+1,
	)

	c := &dbModels.Contact{}
	if dbErr := r.db.FindOneAndUpdate(ctx, c, query, args...); dbErr != nil {
		if postgres.IsNotFoundError(dbErr) {
			return nil, pkgErrors.New(userErrors.ErrCodeInvalidRequest, "Contact not found or not in active state")
		}
		r.log.Error("failed to update contact",
			logger.String("contact_id", contactID),
			logger.Error(dbErr),
		)
		return nil, pkgErrors.FromError(dbErr, userErrors.ErrCodeDatabaseError, "failed to update contact")
	}
	return domain.NewContact(*c), nil
}

func (r *contactRepository) DeleteContact(ctx context.Context, contactID, userID string) pkgErrors.AppError {
	query := `DELETE FROM users.contacts WHERE id = $1 AND user_id = $2`

	if _, dbErr := r.db.Exec(ctx, query, contactID, userID); dbErr != nil {
		r.log.Error("failed to delete contact",
			logger.String("contact_id", contactID),
			logger.Error(dbErr),
		)
		return pkgErrors.FromError(dbErr, userErrors.ErrCodeDatabaseError, "failed to delete contact")
	}
	return nil
}
