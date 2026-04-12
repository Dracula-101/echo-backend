package repo

import (
	"context"
	"shared/pkg/cache"
	"shared/pkg/database"
	"shared/pkg/database/postgres"
	dbModel "shared/pkg/database/postgres/models"
	pkgErrors "shared/pkg/errors"
	"shared/pkg/logger"

	"github.com/google/uuid"
)

// PollRepository handles DB access for polls, options, and votes.
type PollRepository interface {
	CreatePoll(ctx context.Context, poll *dbModel.Poll, options []*dbModel.PollOption) pkgErrors.AppError
	GetPollByMessageID(ctx context.Context, messageID uuid.UUID) (*dbModel.Poll, []*dbModel.PollOption, pkgErrors.AppError)
	GetPollByID(ctx context.Context, pollID uuid.UUID) (*dbModel.Poll, pkgErrors.AppError)
	VotePoll(ctx context.Context, vote *dbModel.PollVote) pkgErrors.AppError
	GetPollResults(ctx context.Context, pollID uuid.UUID) ([]*dbModel.PollOption, int, pkgErrors.AppError)
}

type pollRepository struct {
	db     database.Database
	cache  cache.Cache
	logger logger.Logger
}

func NewPollRepository(db database.Database, cache cache.Cache, log logger.Logger) PollRepository {
	return &pollRepository{
		db:     db,
		cache:  cache,
		logger: log,
	}
}

// CreatePoll inserts a poll and its multiple options within a transaction.
func (r *pollRepository) CreatePoll(ctx context.Context, poll *dbModel.Poll, options []*dbModel.PollOption) pkgErrors.AppError {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		r.logger.Error("Failed to begin transaction for CreatePoll", logger.Error(err))
		return pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "Failed to create poll transaction")
	}
	defer func() { _ = tx.Rollback() }()

	pollQuery := `
		INSERT INTO messages.polls (
			id, message_id, question, allow_multiple_answers, 
			is_anonymous, is_quiz, correct_option_id, explanation, 
			closes_at, is_closed, closed_at, total_votes, created_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)
	`
	if _, err := tx.Exec(ctx, pollQuery,
		poll.ID, poll.MessageID, poll.Question, poll.AllowMultiple,
		poll.IsAnonymous, poll.IsQuiz, poll.CorrectOptionID, poll.Explanation,
		poll.ClosesAt, poll.IsClosed, poll.ClosedAt, poll.TotalVotes, poll.CreatedAt,
	); err != nil {
		r.logger.Error("Failed to insert poll", logger.Error(err))
		return pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "Failed to insert poll")
	}

	optionQuery := `
		INSERT INTO messages.poll_options (
			id, poll_id, option_text, option_order, vote_count, vote_percentage, created_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7)
	`
	for _, opt := range options {
		if _, err := tx.Exec(ctx, optionQuery,
			opt.ID, opt.PollID, opt.OptionText, opt.OptionOrder,
			opt.VoteCount, opt.VotePercentage, opt.CreatedAt,
		); err != nil {
			r.logger.Error("Failed to insert poll option", logger.Error(err))
			return pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "Failed to insert poll option")
		}
	}

	if err := tx.Commit(); err != nil {
		return pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "Failed to commit create poll transaction")
	}
	return nil
}

// GetPollByMessageID retrieves a poll and its options by the associated message ID.
func (r *pollRepository) GetPollByMessageID(ctx context.Context, messageID uuid.UUID) (*dbModel.Poll, []*dbModel.PollOption, pkgErrors.AppError) {
	pollQuery := `SELECT * FROM messages.polls WHERE message_id = $1`
	var poll dbModel.Poll

	err := r.db.QueryRow(ctx, pollQuery, messageID).Scan(
		&poll.ID, &poll.MessageID, &poll.Question, &poll.AllowMultiple,
		&poll.IsAnonymous, &poll.IsQuiz, &poll.CorrectOptionID, &poll.Explanation,
		&poll.ClosesAt, &poll.IsClosed, &poll.ClosedAt, &poll.TotalVotes, &poll.CreatedAt,
	)

	if err != nil {
		if postgres.IsNotFoundError(err) {
			return nil, nil, pkgErrors.FromError(err, pkgErrors.CodeNotFound, "Poll not found")
		}
		r.logger.Error("Failed to query poll by message_id", logger.Error(err))
		return nil, nil, pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "Failed to get poll")
	}

	optionsQuery := `SELECT * FROM messages.poll_options WHERE poll_id = $1 ORDER BY option_order ASC`
	rows, err := r.db.Query(ctx, optionsQuery, poll.ID)
	if err != nil {
		return nil, nil, pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "Failed to get poll options")
	}
	defer rows.Close()

	var options []*dbModel.PollOption
	for rows.Next() {
		opt := &dbModel.PollOption{}
		if err := rows.Scan(
			&opt.ID, &opt.PollID, &opt.OptionText, &opt.OptionOrder,
			&opt.VoteCount, &opt.VotePercentage, &opt.CreatedAt,
		); err != nil {
			return nil, nil, pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "Failed to scan poll option")
		}
		options = append(options, opt)
	}

	return &poll, options, nil
}

func (r *pollRepository) GetPollByID(ctx context.Context, pollID uuid.UUID) (*dbModel.Poll, pkgErrors.AppError) {
	pollQuery := `SELECT * FROM messages.polls WHERE id = $1`
	var poll dbModel.Poll
	err := r.db.QueryRow(ctx, pollQuery, pollID).Scan(
		&poll.ID, &poll.MessageID, &poll.Question, &poll.AllowMultiple,
		&poll.IsAnonymous, &poll.IsQuiz, &poll.CorrectOptionID, &poll.Explanation,
		&poll.ClosesAt, &poll.IsClosed, &poll.ClosedAt, &poll.TotalVotes, &poll.CreatedAt,
	)
	if err != nil {
		if postgres.IsNotFoundError(err) {
			return nil, pkgErrors.FromError(err, pkgErrors.CodeNotFound, "Poll not found")
		}
		return nil, pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "Failed to get poll")
	}
	return &poll, nil
}

// VotePoll inserts a vote and updates the relevant denormalized aggregates atomically.
func (r *pollRepository) VotePoll(ctx context.Context, vote *dbModel.PollVote) pkgErrors.AppError {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "Failed to begin vote transaction")
	}
	defer func() { _ = tx.Rollback() }()

	voteQuery := `
		INSERT INTO messages.poll_votes (id, poll_id, poll_option_id, user_id, voted_at)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (poll_id, user_id, poll_option_id) DO NOTHING
	`
	res, err := tx.Exec(ctx, voteQuery, vote.ID, vote.PollID, vote.PollOptionID, vote.UserID, vote.VotedAt)
	if err != nil {
		r.logger.Error("Failed to insert vote", logger.Error(err))
		return pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "Failed to cast vote")
	}

	rowsAffected, affErr := res.RowsAffected()
	if affErr != nil {
		return pkgErrors.FromError(affErr, pkgErrors.CodeDatabaseError, "Failed to get rows affected")
	}
	if rowsAffected == 0 {
		return pkgErrors.New(pkgErrors.CodeAlreadyExists, "Already voted on this option")
	}

	// Update the specific option's count
	if _, err := tx.Exec(ctx, `UPDATE messages.poll_options SET vote_count = vote_count + 1 WHERE id = $1`, vote.PollOptionID); err != nil {
		return pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "Failed to update option metadata")
	}

	// Update the poll's total vote count
	if _, err := tx.Exec(ctx, `UPDATE messages.polls SET total_votes = total_votes + 1 WHERE id = $1`, vote.PollID); err != nil {
		return pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "Failed to update poll metadata")
	}

	// Recalculate percentages securely in DB
	recalcQuery := `
		UPDATE messages.poll_options
		SET vote_percentage = (vote_count::float / NULLIF((SELECT total_votes FROM messages.polls WHERE id = $1), 0)) * 100
		WHERE poll_id = $1
	`
	if _, err := tx.Exec(ctx, recalcQuery, vote.PollID); err != nil {
		return pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "Failed to recalculate percentages")
	}

	if err := tx.Commit(); err != nil {
		return pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "Failed to commit vote transaction")
	}
	return nil
}

// GetPollResults fetches options with real-time aggregates.
func (r *pollRepository) GetPollResults(ctx context.Context, pollID uuid.UUID) ([]*dbModel.PollOption, int, pkgErrors.AppError) {
	// Let's directly get the total votes as well
	var totalVotes int
	if err := r.db.QueryRow(ctx, `SELECT total_votes FROM messages.polls WHERE id = $1`, pollID).Scan(&totalVotes); err != nil {
		if postgres.IsNotFoundError(err) {
			return nil, 0, pkgErrors.FromError(err, pkgErrors.CodeNotFound, "Poll not found")
		}
		return nil, 0, pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "Failed to fetch poll total")
	}

	query := `SELECT * FROM messages.poll_options WHERE poll_id = $1 ORDER BY option_order ASC`
	rows, err := r.db.Query(ctx, query, pollID)
	if err != nil {
		return nil, 0, pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "Failed to get poll options")
	}
	defer rows.Close()

	var options []*dbModel.PollOption
	for rows.Next() {
		opt := &dbModel.PollOption{}
		if err := rows.Scan(
			&opt.ID, &opt.PollID, &opt.OptionText, &opt.OptionOrder,
			&opt.VoteCount, &opt.VotePercentage, &opt.CreatedAt,
		); err != nil {
			return nil, 0, pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "Failed to scan poll option")
		}
		options = append(options, opt)
	}
	return options, totalVotes, nil
}
