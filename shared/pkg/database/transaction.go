package database

import (
	"context"
)

type TxFunc func(ctx context.Context, tx Transaction) error

func WithTransaction(ctx context.Context, db Database, fn TxFunc) error {
	tx, err := db.Begin(ctx)
	if err != nil {
		return err
	}

	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		}
	}()

	if err := fn(ctx, tx); err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			return rbErr
		}
		return err
	}

	return tx.Commit()
}

func WithTransactionOpts(ctx context.Context, db Database, opts *TxOptions, fn TxFunc) error {
	tx, err := db.BeginTx(ctx, opts)
	if err != nil {
		return err
	}

	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		}
	}()

	if err := fn(ctx, tx); err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			return rbErr
		}
		return err
	}

	return tx.Commit()
}
