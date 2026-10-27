package me

import (
	"librequenda/model"
	"librequenda/storage"
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
	BirthDate         string `json:"birth_date"` // YYYY-MM-DD, "" until filled in
	Phone             string `json:"phone"`
	AvatarRef         string `json:"avatar_ref"` // sent back unchanged when the form is saved
	AvatarUrl         string `json:"avatar_url"`
	PreferredLanguage string `json:"preferred_language"`
	EmailReminders    bool   `json:"email_reminders"`
	ProfileComplete   bool   `json:"profile_complete"` // the date of birth is required before booking

	// roles waiting for this person: they are granted once the email is verified
	PendingInvitations []PendingInvitationDto `json:"pending_invitations"`
}

type PendingInvitationDto struct {
	TenantName string `json:"tenant_name"`
	TenantSlug string `json:"tenant_slug"`
}

type ProfileInput struct {
	BirthDate         string `json:"birth_date"`
	Phone             string `json:"phone"`
	AvatarRef         string `json:"avatar_ref"` // a storage reference from POST /media, never a URL
	PreferredLanguage string `json:"preferred_language"`
	EmailReminders    bool   `json:"email_reminders"`
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
	response.ProfileComplete = profile.BirthDate.Valid
	return response
}

func MapMeResponse(user model.UpsertUserBySubjectRow, emailVerified bool, isSuperadmin bool, roles []model.GetUserTenantRolesRow) *MeResponse {
	var response = &MeResponse{
		Id:            user.ID.String(),
		Email:         user.Email,
		EmailVerified: emailVerified,
		IsSuperadmin:  isSuperadmin,
		FullName:      user.FullName.String,
		Roles:         make([]TenantRoleDto, 0, len(roles)),
	}
	for _, role := range roles {
		response.Roles = append(response.Roles, TenantRoleDto{
			TenantId:   role.TenantID.String(),
			TenantSlug: role.TenantSlug,
			TenantName: role.TenantName,
			Role:       string(role.Role),
		})
	}
	return response
}
