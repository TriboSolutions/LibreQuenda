package db

import (
	"context"
	"database/sql"
	"librequenda/model"

	"github.com/google/uuid"
)

// WithTenant runs fn in a transaction that names one council. The row-level
// security policies (db/init/05_row_security.sql) refuse writes to a council's
// facilities, hours, closures and readers anywhere else.
func WithTenant(ctx context.Context, conn *sql.DB, tenantId uuid.UUID, fn func(queries *model.Queries) error) error {
	var tx, err = conn.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var queries = model.New(tx)
	if err = queries.SetTenant(ctx, tenantId.String()); err != nil {
		return err
	}
	if err = fn(queries); err != nil {
		return err
	}
	return tx.Commit()
}
