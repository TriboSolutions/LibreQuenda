package me

import (
	"context"
	"database/sql"
	"librequenda/auth"
	"librequenda/model"
	"librequenda/shared"
	"librequenda/storage"
	"librequenda/zitadel"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

type meService struct {
	db *sql.DB
}

var sMeInstance *meService
var sMeOnce *sync.Once = new(sync.Once)

func GetMeService(db *sql.DB) *meService {
	sMeOnce.Do(func() {
		sMeInstance = &meService{db: db}
	})
	return sMeInstance
}

func (s *meService) SyncCurrentUser(ctx context.Context, session auth.Session) (*MeResponse, *shared.ErrorResponse) {
	var info, err = auth.FetchUserInfo(ctx, session.AccessToken)
	if err != nil || info.Sub != session.Subject {
		shared.Logger.Error("failed to fetch userinfo for %s: %v", session.Subject, err)
		return nil, &shared.ErrorResponse{
			Code:    http.StatusBadGateway,
			Message: "errors.user_info_failed",
		}
	}

	var queries = model.New(s.db)
	if info.EmailVerified {
		// local testing: the seed's tester, waiting with a booking, becomes this account
		if err = queries.AdoptSeedTester(ctx, model.AdoptSeedTesterParams{IdpSubject: session.Subject, Email: info.Email}); err != nil {
			shared.Logger.Error("failed to adopt the seed tester for %s: %v", session.Subject, err)
		}
	}
	var user model.UpsertUserBySubjectRow
	user, err = queries.UpsertUserBySubject(ctx, model.UpsertUserBySubjectParams{
		IdpSubject:    session.Subject,
		Email:         info.Email,
		FullName:      sql.NullString{String: info.Name, Valid: info.Name != ""},
		EmailVerified: info.EmailVerified,
	})
	if err != nil {
		shared.Logger.Error("failed to upsert user %s: %v", session.Subject, err)
		return nil, &shared.ErrorResponse{
			Code:    http.StatusInternalServerError,
			Message: "errors.user_sync_failed",
		}
	}

	// Roles only follow a verified email: anyone could register with someone else's address
	if info.EmailVerified {
		_, err = queries.AcceptPendingInvitations(ctx, model.AcceptPendingInvitationsParams{UserID: user.ID, Email: info.Email})
		if err == nil && shared.SUPERADMIN_EMAIL != "" && strings.EqualFold(info.Email, shared.SUPERADMIN_EMAIL) {
			err = queries.AddPlatformAdmin(ctx, user.ID)
		}
		if err != nil {
			shared.Logger.Error("failed to grant roles to %s: %v", session.Subject, err)
			return nil, &shared.ErrorResponse{
				Code:    http.StatusInternalServerError,
				Message: "errors.user_sync_failed",
			}
		}
	}

	var roles []model.GetUserTenantRolesRow
	roles, err = queries.GetUserTenantRoles(ctx, user.ID)
	if err != nil {
		return nil, &shared.ErrorResponse{
			Code:    http.StatusInternalServerError,
			Message: "errors.user_sync_failed",
		}
	}
	var isSuperadmin bool
	isSuperadmin, err = queries.IsPlatformAdmin(ctx, user.ID)
	if err != nil {
		return nil, &shared.ErrorResponse{
			Code:    http.StatusInternalServerError,
			Message: "errors.user_sync_failed",
		}
	}

	var profile model.GetUserProfileRow
	profile, err = queries.GetUserProfile(ctx, user.ID)
	if err != nil {
		return nil, &shared.ErrorResponse{
			Code:    http.StatusInternalServerError,
			Message: "errors.user_sync_failed",
		}
	}

	var pending []model.ListPendingInvitationsForEmailRow
	pending, err = queries.ListPendingInvitationsForEmail(ctx, info.Email)
	if err != nil {
		return nil, &shared.ErrorResponse{
			Code:    http.StatusInternalServerError,
			Message: "errors.user_sync_failed",
		}
	}

	var response = withProfile(MapMeResponse(user, info.EmailVerified, isSuperadmin, roles), profile)
	// no photo of their own here: the one on their sign-in account (Zitadel, or Google through it)
	if response.AvatarUrl == "" {
		response.AvatarUrl = info.Picture
	}
	response.PendingInvitations = make([]PendingInvitationDto, 0, len(pending))
	for _, p := range pending {
		response.PendingInvitations = append(response.PendingInvitations, PendingInvitationDto{TenantName: p.TenantName, TenantSlug: p.TenantSlug})
	}
	return response, nil
}

var phonePattern = regexp.MustCompile(`^\+?[0-9 ()-]{6,30}$`)
var languages = map[string]bool{"": true, "gl": true, "es": true, "en": true, "ca": true}

// validateProfile returns the parsed date of birth, or the i18n key of the first invalid field.
func validateProfile(in *ProfileInput, today time.Time) (time.Time, string) {
	in.Phone = strings.TrimSpace(in.Phone)
	in.AvatarRef = strings.TrimSpace(in.AvatarRef)
	var birth, err = time.Parse("2006-01-02", strings.TrimSpace(in.BirthDate))
	switch {
	case err != nil || !birth.Before(today) || birth.Before(today.AddDate(-120, 0, 0)):
		return birth, "errors.invalid_birth_date"
	case in.Phone != "" && !phonePattern.MatchString(in.Phone):
		return birth, "errors.invalid_phone_format"
	case in.AvatarRef != "" && !storage.Owned(in.AvatarRef, uuid.Nil, "users"):
		return birth, "errors.invalid_image"
	}
	return birth, ""
}

func (s *meService) UpdateProfile(ctx context.Context, session auth.Session, input ProfileInput) (*MeResponse, *shared.ErrorResponse) {
	var birth, key = validateProfile(&input, time.Now())
	if key != "" {
		return nil, &shared.ErrorResponse{Code: http.StatusUnprocessableEntity, Message: key}
	}
	var userId, userErr = CurrentUserId(ctx, s.db, session)
	if userErr != nil {
		return nil, userErr
	}
	var before, _ = model.New(s.db).GetUserProfile(ctx, userId)
	var err = model.New(s.db).UpdateUserProfile(ctx, model.UpdateUserProfileParams{
		ID:        userId,
		BirthDate: sql.NullTime{Time: birth, Valid: true},
		Phone:     sql.NullString{String: input.Phone, Valid: input.Phone != ""},
		AvatarRef: sql.NullString{String: input.AvatarRef, Valid: input.AvatarRef != ""},
	})
	if err != nil {
		return nil, &shared.ErrorResponse{Code: http.StatusInternalServerError, Message: "errors.profile_save_failed"}
	}
	if input.AvatarRef != before.AvatarRef.String {
		syncAvatar(ctx, session, input.AvatarRef)
	}
	return s.SyncCurrentUser(ctx, session)
}

// syncAvatar copies the new profile photo to the person's Zitadel account, or
// takes it off, so the sign-in pages show the same one. Best effort: the photo
// is already saved here, and a token without Zitadel's audience (signed in
// before it was asked for) just leaves Zitadel as it was until the next save.
func syncAvatar(ctx context.Context, session auth.Session, ref string) {
	var err error
	if ref == "" {
		err = zitadel.RemoveOwnAvatar(ctx, session.AccessToken)
	} else {
		var photo *storage.Object
		if photo, err = storage.Get(ctx, ref); err == nil {
			defer photo.Reader.Close()
			err = zitadel.SetOwnAvatar(ctx, session.AccessToken, photo.Reader, photo.ContentType)
		}
	}
	if err != nil {
		shared.Logger.Error("profile photo of %s not copied to Zitadel: %v", session.Subject, err)
	}
}

// SetShowRecommendations turns "For you" on the home page on or off, also from
// the section itself (no need to fill in the whole profile).
func (s *meService) SetShowRecommendations(ctx context.Context, session auth.Session, show bool) (*MeResponse, *shared.ErrorResponse) {
	var userId, userErr = CurrentUserId(ctx, s.db, session)
	if userErr != nil {
		return nil, userErr
	}
	if err := model.New(s.db).SetShowRecommendations(ctx, model.SetShowRecommendationsParams{ID: userId, Show: show}); err != nil {
		return nil, &shared.ErrorResponse{Code: http.StatusInternalServerError, Message: "errors.profile_save_failed"}
	}
	return s.SyncCurrentUser(ctx, session)
}

// UpdatePreferences saves the profile's preferences on their own.
func (s *meService) UpdatePreferences(ctx context.Context, session auth.Session, input PreferencesInput) (*MeResponse, *shared.ErrorResponse) {
	if !languages[input.PreferredLanguage] {
		return nil, &shared.ErrorResponse{Code: http.StatusUnprocessableEntity, Message: "errors.invalid_language"}
	}
	var userId, userErr = CurrentUserId(ctx, s.db, session)
	if userErr != nil {
		return nil, userErr
	}
	var err = model.New(s.db).UpdateUserPreferences(ctx, model.UpdateUserPreferencesParams{
		ID:                  userId,
		PreferredLanguage:   sql.NullString{String: input.PreferredLanguage, Valid: input.PreferredLanguage != ""},
		EmailReminders:      input.EmailReminders,
		EmailNotifications:  input.EmailNotifications,
		ShowRecommendations: input.ShowRecommendations,
	})
	if err != nil {
		return nil, &shared.ErrorResponse{Code: http.StatusInternalServerError, Message: "errors.profile_save_failed"}
	}
	return s.SyncCurrentUser(ctx, session)
}
