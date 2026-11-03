package me

import (
	"context"
	"librequenda/auth"
	"librequenda/model"
	"librequenda/shared"
	"librequenda/storage"
	"librequenda/zitadel"
	"net/http"
	"time"

	"github.com/google/uuid"
)

// Export gathers everything LibreQuenda keeps about the user (RXPD / GDPR
// access and portability). Name and email come from the copy kept here, which
// SyncCurrentUser refreshes from Zitadel on every sign-in.
func (s *meService) Export(ctx context.Context, session auth.Session) (*ExportResponse, *shared.ErrorResponse) {
	var userId, userErr = CurrentUserId(ctx, s.db, session)
	if userErr != nil {
		return nil, userErr
	}
	var failed = &shared.ErrorResponse{Code: http.StatusInternalServerError, Message: "errors.export_failed"}
	var queries = model.New(s.db)
	var owner = uuid.NullUUID{UUID: userId, Valid: true}

	var user, err = queries.ExportUser(ctx, userId)
	if err != nil {
		return nil, failed
	}
	var roles []model.GetUserTenantRolesRow
	if roles, err = queries.GetUserTenantRoles(ctx, userId); err != nil {
		return nil, failed
	}
	var isSuperadmin bool
	if isSuperadmin, err = queries.IsPlatformAdmin(ctx, userId); err != nil {
		return nil, failed
	}
	var bookings []model.ExportBookingsByUserRow
	if bookings, err = queries.ExportBookingsByUser(ctx, owner); err != nil {
		return nil, failed
	}
	var participations []model.ExportParticipationsByUserRow
	if participations, err = queries.ExportParticipationsByUser(ctx, owner); err != nil {
		return nil, failed
	}
	var notifications []model.ExportNotificationsByUserRow
	if notifications, err = queries.ExportNotificationsByUser(ctx, userId); err != nil {
		return nil, failed
	}
	var issues []model.ListIssuesByReporterRow
	if issues, err = queries.ListIssuesByReporter(ctx, owner); err != nil {
		return nil, failed
	}
	var facilityRatings []model.ExportFacilityRatingsByUserRow
	if facilityRatings, err = queries.ExportFacilityRatingsByUser(ctx, owner); err != nil {
		return nil, failed
	}
	var peopleRatings []model.ExportPeopleRatingsByUserRow
	if peopleRatings, err = queries.ExportPeopleRatingsByUser(ctx, userId); err != nil {
		return nil, failed
	}
	var received model.GetUserRatingSummaryRow
	if received, err = queries.GetUserRatingSummary(ctx, userId); err != nil {
		return nil, failed
	}

	return &ExportResponse{
		GeneratedAt:    time.Now().UTC(),
		Account:        mapExportAccount(user),
		IsSuperadmin:   isSuperadmin,
		Roles:          mapRoles(roles),
		Bookings:       mapExportBookings(bookings),
		Participations: mapExportParticipations(participations),
		Notifications:  mapExportNotifications(notifications),
		Issues:         mapExportIssues(issues),
		Ratings:        mapExportRatings(facilityRatings, peopleRatings, received),
	}, nil
}

// DeleteAccount erases the user (RXPD / GDPR erasure): upcoming bookings are
// cancelled so the slots free up, past ones stay for the council's indicators
// with nobody attached, and the identity is deleted in Zitadel. Zitadel goes
// inside the transaction: if it refuses, nothing here is erased either.
func (s *meService) DeleteAccount(ctx context.Context, session auth.Session) *shared.ErrorResponse {
	var userId, userErr = CurrentUserId(ctx, s.db, session)
	if userErr != nil {
		return userErr
	}
	var failed = &shared.ErrorResponse{Code: http.StatusInternalServerError, Message: "errors.account_delete_failed"}
	var tx, err = s.db.BeginTx(ctx, nil)
	if err != nil {
		return failed
	}
	defer tx.Rollback()
	var queries = model.New(tx)
	var owner = uuid.NullUUID{UUID: userId, Valid: true}

	if _, err = queries.CancelUpcomingBookingsOfUser(ctx, owner); err != nil {
		shared.Logger.Error("erasure of %s: cancel bookings: %v", userId, err)
		return failed
	}
	if err = queries.RevokeInviteLinksOfUser(ctx, owner); err != nil {
		shared.Logger.Error("erasure of %s: revoke invite links: %v", userId, err)
		return failed
	}
	var avatar, deleteErr = queries.DeleteUser(ctx, userId)
	if deleteErr != nil {
		shared.Logger.Error("erasure of %s: delete user: %v", userId, deleteErr)
		return failed
	}
	if err = zitadel.DeleteUser(ctx, session.Subject); err != nil {
		shared.Logger.Error("erasure of %s: zitadel: %v", userId, err)
		return &shared.ErrorResponse{Code: http.StatusBadGateway, Message: "errors.account_delete_failed"}
	}
	if err = tx.Commit(); err != nil {
		// the identity is gone but the data is not: it has to be erased by hand
		shared.Logger.Error("erasure of %s (subject %s): deleted in zitadel but the commit failed: %v", userId, session.Subject, err)
		return failed
	}

	if avatar.Valid {
		if err = storage.Delete(ctx, avatar.String); err != nil {
			shared.Logger.Error("erasure of %s: avatar %s left in storage: %v", userId, avatar.String, err)
		}
	}
	return nil
}
