package data

import (
	"context"
	"database/sql"
	"fmt"
)

type txContextKey struct{}

// TxManager starts and completes database transactions.
type TxManager struct {
	db *sql.DB
}

// NewTxManager creates a transaction manager for db.
func NewTxManager(db *sql.DB) *TxManager {
	return &TxManager{db: db}
}

// ContextWithTx returns a child context containing tx.
func ContextWithTx(ctx context.Context, tx *sql.Tx) context.Context {
	return context.WithValue(ctx, txContextKey{}, tx)
}

// TxFromContext returns the active transaction from ctx, when present.
func TxFromContext(ctx context.Context) (*sql.Tx, bool) {
	tx, ok := ctx.Value(txContextKey{}).(*sql.Tx)
	return tx, ok && tx != nil
}

// WithTx runs fn inside a transaction, committing on success and rolling back on error or panic.
func (m *TxManager) WithTx(ctx context.Context, fn func(context.Context, *sql.Tx) error) (err error) {
	if m == nil || m.db == nil {
		return fmt.Errorf("database is nil")
	}
	if fn == nil {
		return fmt.Errorf("transaction function is nil")
	}

	tx, err := m.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	defer func() {
		if recovered := recover(); recovered != nil {
			_ = tx.Rollback()
			panic(recovered)
		}
	}()

	if err := fn(ContextWithTx(ctx, tx), tx); err != nil {
		if rollbackErr := tx.Rollback(); rollbackErr != nil {
			return fmt.Errorf("transaction failed: %w; rollback failed: %v", err, rollbackErr)
		}
		return err
	}

	return tx.Commit()
}
