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
	query := `
		SELECT
			id, user_id, contact_user_id, status, relationship_type,
			nickname, notes, is_favorite, is_pinned, is_archived,
			is_muted, muted_until, contact_source, contact_groups,
			accepted_at, blocked_at, created_at, updated_at,
			COUNT(*) OVER() AS total
		FROM users.contacts
		WHERE user_id = $1
		AND status = $2
		ORDER BY is_pinned DESC, is_favorite DESC, last_interaction_at DESC NULLS LAST
		LIMIT $3 OFFSET $4`

	var rows []struct {
		dbModels.Contact
		Total int `db:"total"`
	}
	if dbErr := r.db.FindMany(ctx, &rows, query, userID, params.Status, params.Limit, params.Offset); dbErr != nil {
		r.log.Error("failed to get contacts",
			logger.String("user_id", userID),
			logger.Error(dbErr),
		)
		return nil, 0, pkgErrors.FromError(dbErr, userErrors.ErrCodeDatabaseError, "failed to get contacts")
	}

	total := 0
	list := make([]*domain.Contact, len(rows))
	for i, row := range rows {
		list[i] = domain.NewContact(row.Contact)
		total = row.Total
	}
	return list, total, nil
}

func (r *contactRepository) GetContact(ctx context.Context, userID, contactID string) (*domain.Contact, pkgErrors.AppError) {
	c := &dbModels.Contact{}
	if dbErr := r.db.FindOne(ctx, c, `SELECT * FROM users.contacts WHERE id = $1 AND user_id = $2`, contactID, userID); dbErr != nil {
		r.log.Error("failed to get contact",
			logger.String("contact_id", contactID),
			logger.Error(dbErr),
		)
		return nil, pkgErrors.FromError(dbErr, userErrors.ErrCodeDatabaseError, "failed to get contact")
	}
	return domain.NewContact(*c), nil
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
		INSERT INTO users.contacts (user_id, contact_user_id, status, contact_source, accepted_at, created_at)
		SELECT contact_user_id, user_id, 'active', 'contact_accept', NOW(), NOW()
		FROM users.contacts
		WHERE id = $1 AND user_id = $2 AND status = 'pending'
		ON CONFLICT (user_id, contact_user_id) DO UPDATE
		SET status = 'active', accepted_at = NOW(), updated_at = NOW()
		RETURNING *`

	c := &dbModels.Contact{}
	if dbErr := r.db.FindOneAndUpdate(ctx, c, query, contactID, userID); dbErr != nil {
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
		SET status     = 'deleted',
		    updated_at = NOW()
		WHERE id = $1
		AND contact_user_id = $2
		AND status = 'pending'`

	if _, dbErr := r.db.Exec(ctx, query, contactID, userID); dbErr != nil {
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
