package me

import (
	"context"
	"librequenda/auth"
	"librequenda/checkin"
	"librequenda/model"
	"librequenda/shared"
	"net/http"
	"time"
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
