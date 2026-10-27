package me

import (
	"context"
	"database/sql"
	"errors"
	"librequenda/auth"
	"librequenda/model"
	"librequenda/shared"

	"github.com/google/uuid"
)

// CurrentUserId returns the app_user id of the session, creating the row from
// Zitadel the first time this subject is seen.
func CurrentUserId(ctx context.Context, db *sql.DB, session auth.Session) (uuid.UUID, *shared.ErrorResponse) {
	var id, err = model.New(db).GetUserIdBySubject(ctx, session.Subject)
	if err == nil {
		return id, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return uuid.Nil, &shared.ErrorResponse{Code: 500, Message: "errors.user_sync_failed"}
	}

	var me, syncErr = GetMeService(db).SyncCurrentUser(ctx, session)
	if syncErr != nil {
		return uuid.Nil, syncErr
	}
	return uuid.MustParse(me.Id), nil
}
