package me

import (
	"context"
	"database/sql"
	"errors"
	"librequenda/auth"
	"librequenda/checkin"
	"librequenda/model"
	"librequenda/shared"
	"net/http"
	"time"

	"github.com/google/uuid"
)

// PersonalQr issues a new personal QR, for a reader to check the person in or
// book the turn about to start (see reader.personalCheckIn).
func (s *meService) PersonalQr(ctx context.Context, session auth.Session) (*PersonalQrDto, *shared.ErrorResponse) {
	var failed = &shared.ErrorResponse{Code: http.StatusInternalServerError, Message: "errors.qr_failed"}
	var userId, userErr = CurrentUserId(ctx, s.db, session)
	if userErr != nil {
		return nil, userErr
	}
	var queries = model.New(s.db)
	if err := queries.DeleteExpiredUserQrTokens(ctx, userId); err != nil {
		return nil, failed
	}
	var issued, err = queries.CreateUserQrToken(ctx, model.CreateUserQrTokenParams{
		UserID:    userId,
		ExpiresAt: time.Now().Add(checkin.PersonalValidFor),
	})
	if err != nil {
		return nil, failed
	}
	return &PersonalQrDto{
		Code:           checkin.PersonalPayload(issued.Token),
		ExpiresAt:      issued.ExpiresAt,
		RefreshSeconds: int(checkin.PersonalRefresh / time.Second),
	}, nil
}

// PersonalQrStatus: what became of one of the person's personal QRs; someone else's reads as unused.
func (s *meService) PersonalQrStatus(ctx context.Context, session auth.Session, token uuid.UUID) (*PersonalQrStatusDto, *shared.ErrorResponse) {
	var userId, userErr = CurrentUserId(ctx, s.db, session)
	if userErr != nil {
		return nil, userErr
	}
	var row, err = model.New(s.db).GetUserQrTokenUse(ctx, model.GetUserQrTokenUseParams{Token: token, UserID: userId})
	if errors.Is(err, sql.ErrNoRows) {
		return &PersonalQrStatusDto{}, nil
	}
	if err != nil {
		return nil, &shared.ErrorResponse{Code: http.StatusInternalServerError, Message: "errors.qr_failed"}
	}
	var response = &PersonalQrStatusDto{Used: row.UsedAt.Valid}
	if row.BookingID.Valid {
		response.BookingId = row.BookingID.UUID.String()
	}
	return response, nil
}
