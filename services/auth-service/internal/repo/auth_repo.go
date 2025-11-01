package repository

import (
	"context"
	"shared/pkg/database"
	"shared/pkg/logger"
)

type AuthRepository struct {
	db  database.Database
	log logger.Logger
}

func NewAuthRepository(db database.Database, log logger.Logger) *AuthRepository {
	return &AuthRepository{
		db:  db,
		log: log,
	}
}

func (r *AuthRepository) CreateUser(ctx context.Context, email, passwordHash string) error {
	query := `INSERT INTO users (email, password_hash) VALUES ($1, $2)`
	_, err := r.db.Exec(ctx, query, email, passwordHash)
	return err
}

func (r *AuthRepository) GetUserByEmail(ctx context.Context, email string) (map[string]interface{}, error) {
	query := `SELECT id, email, password_hash, verified FROM users WHERE email = $1`
	rows, err := r.db.Query(ctx, query, email)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	if rows.Next() {
		var id, email, passwordHash string
		var verified bool
		if err := rows.Scan(&id, &email, &passwordHash, &verified); err != nil {
			return nil, err
		}
		return map[string]interface{}{
			"id":            id,
			"email":         email,
			"password_hash": passwordHash,
			"verified":      verified,
		}, nil
	}
	return nil, nil
}
