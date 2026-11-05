package me

import (
	"librequenda/model"
	"librequenda/rating"
	"librequenda/storage"
	"time"
)

type TenantRoleDto struct {
	TenantId   string `json:"tenant_id"`
	TenantSlug string `json:"tenant_slug"`
	TenantName string `json:"tenant_name"`
	Role       string `json:"role"`
}

type MeResponse struct {
	Id            string          `json:"id"`
	Email         string          `json:"email"`
	EmailVerified bool            `json:"email_verified"`
	FullName      string          `json:"full_name"`
	IsSuperadmin  bool            `json:"is_superadmin"`
	Roles         []TenantRoleDto `json:"roles"`

	// profile kept by LibreQuenda; name and email belong to Zitadel
	BirthDate           string `json:"birth_date"` // YYYY-MM-DD, "" until filled in
	Phone               string `json:"phone"`
	AvatarRef           string `json:"avatar_ref"` // sent back unchanged when the form is saved
	AvatarUrl           string `json:"avatar_url"`
	PreferredLanguage   string `json:"preferred_language"`
	EmailReminders      bool   `json:"email_reminders"`
	EmailNotifications  bool   `json:"email_notifications"`  // off: booking and problem notices in the app only
	ShowRecommendations bool   `json:"show_recommendations"` // "For you" on the home page
	ProfileComplete     bool   `json:"profile_complete"`     // the date of birth is required before booking
	// what others gave them after public bookings; null until someone did
	Rating *rating.Summary `json:"rating"`

	// roles waiting for this person: they are granted once the email is verified
	PendingInvitations []PendingInvitationDto `json:"pending_invitations"`

	// councils that verified this person as their resident: they book at their residents-only facilities
	ResidentOf []ResidentOfDto `json:"resident_of"`
}

type ResidentOfDto struct {
	TenantName string    `json:"tenant_name"`
	TenantSlug string    `json:"tenant_slug"`
	VerifiedAt time.Time `json:"verified_at"`
}

type PendingInvitationDto struct {
	TenantName string `json:"tenant_name"`
	TenantSlug string `json:"tenant_slug"`
}

// ProfileInput: the personal data; the preferences are saved apart (PreferencesInput).
type ProfileInput struct {
	BirthDate string `json:"birth_date"`
	Phone     string `json:"phone"`
	AvatarRef string `json:"avatar_ref"` // a storage reference from POST /media, never a URL
}

// PreferencesInput: how the app speaks and what it shows or sends them.
type PreferencesInput struct {
	PreferredLanguage   string `json:"preferred_language"` // "" = the website's
	EmailReminders      bool   `json:"email_reminders"`
	EmailNotifications  bool   `json:"email_notifications"`  // off: booking and problem notices in the app only
	ShowRecommendations bool   `json:"show_recommendations"` // "For you" on the home page
}

func withProfile(response *MeResponse, profile model.GetUserProfileRow) *MeResponse {
	if profile.BirthDate.Valid {
		response.BirthDate = profile.BirthDate.Time.Format("2006-01-02")
	}
	response.Phone = profile.Phone.String
	response.AvatarRef = profile.AvatarRef.String
	response.AvatarUrl = storage.URL(profile.AvatarRef.String)
	response.PreferredLanguage = profile.PreferredLanguage.String
	response.EmailReminders = profile.EmailReminders
	response.EmailNotifications = profile.EmailNotifications
	response.ShowRecommendations = profile.ShowRecommendations
	response.ProfileComplete = profile.BirthDate.Valid
	response.Rating = rating.Of(profile.RatingAverage, profile.RatingCount)
	return response
}

func MapMeResponse(user model.UpsertUserBySubjectRow, emailVerified bool, isSuperadmin bool, roles []model.GetUserTenantRolesRow) *MeResponse {
	var response = &MeResponse{
		Id:            user.ID.String(),
		Email:         user.Email,
		EmailVerified: emailVerified,
		IsSuperadmin:  isSuperadmin,
		FullName:      user.FullName.String,
		Roles:         mapRoles(roles),
	}
	return response
}

func mapRoles(roles []model.GetUserTenantRolesRow) []TenantRoleDto {
	var dtos = make([]TenantRoleDto, 0, len(roles))
	for _, role := range roles {
		dtos = append(dtos, TenantRoleDto{
			TenantId:   role.TenantID.String(),
			TenantSlug: role.TenantSlug,
			TenantName: role.TenantName,
			Role:       string(role.Role),
		})
	}
	return dtos
}

func mapResidentOf(rows []model.ListResidentCouncilsByUserRow) []ResidentOfDto {
	var out = make([]ResidentOfDto, 0, len(rows))
	for _, r := range rows {
		out = append(out, ResidentOfDto{TenantName: r.Name, TenantSlug: r.Slug, VerifiedAt: r.VerifiedAt})
	}
	return out
}
