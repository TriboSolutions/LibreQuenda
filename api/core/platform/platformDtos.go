package platform

import (
	"librequenda/storage"
	"librequenda/validate"
	"strings"
	"time"
	"unicode/utf8"
)

// Paths the web app already uses at the root: a council slug may not take them.
var reservedSlugs = map[string]bool{
	"admin": true, "api": true, "callback": true, "me": true, "config": true,
	"assets": true, "static": true, "login": true, "logout": true, "new": true, "perfil": true, "lector": true,
	"reservas": true, "invitacion": true, "accesibilidade": true, "condicions": true, "privacidade": true, "datos": true,
}

type TenantInput struct {
	Slug         string `json:"slug"`
	Name         string `json:"name"`
	Timezone     string `json:"timezone"`
	ContactEmail string `json:"contact_email"`
	ContactPhone string `json:"contact_phone"`
	// INE code of the municipality (5 digits), for MeteoGalicia's forecast; "" = none
	MunicipalityCode string `json:"municipality_code"`
	LogoRef          string `json:"logo_ref"`    // a storage reference from POST /media, never a URL
	IsActive         *bool  `json:"is_active"`   // update only
	AdminEmail       string `json:"admin_email"` // create only: the first council administrator
}

func (in *TenantInput) normalise() {
	in.Slug = strings.ToLower(strings.TrimSpace(in.Slug))
	in.Name = strings.TrimSpace(in.Name)
	in.Timezone = strings.TrimSpace(in.Timezone)
	in.ContactEmail = strings.ToLower(strings.TrimSpace(in.ContactEmail))
	in.ContactPhone = strings.TrimSpace(in.ContactPhone)
	in.MunicipalityCode = strings.TrimSpace(in.MunicipalityCode)
	in.LogoRef = strings.TrimSpace(in.LogoRef)
	in.AdminEmail = strings.ToLower(strings.TrimSpace(in.AdminEmail))
}

// validate returns the i18n key of the first invalid field, or "".
func (in *TenantInput) validate(creating bool) string {
	switch {
	case in.Name == "" || utf8.RuneCountInString(in.Name) > 255:
		return "errors.invalid_name"
	case !validate.Slug(in.Slug):
		return "errors.invalid_slug"
	case reservedSlugs[in.Slug]:
		return "errors.slug_reserved"
	case !validTimezone(in.Timezone):
		return "errors.invalid_timezone"
	case in.ContactEmail != "" && !validEmail(in.ContactEmail):
		return "errors.invalid_email"
	case in.ContactPhone != "" && !validate.Phone(in.ContactPhone, 50):
		return "errors.invalid_phone"
	case in.MunicipalityCode != "" && !validMunicipalityCode(in.MunicipalityCode):
		return "errors.invalid_municipality_code"
	// a logo is uploaded for an existing council: a new one gets it in its settings
	case in.LogoRef != "" && (creating || !storage.ValidRef(in.LogoRef)):
		return "errors.invalid_image"
	case creating && !validEmail(in.AdminEmail):
		return "errors.invalid_admin_email"
	}
	return ""
}

func validTimezone(name string) bool {
	if name == "" {
		return false
	}
	_, err := time.LoadLocation(name)
	return err == nil
}

func validEmail(email string) bool {
	return validate.Email(email)
}

type EmailInput struct {
	Email string `json:"email"`
}

type TenantSummaryDto struct {
	Slug                   string    `json:"slug"`
	Name                   string    `json:"name"`
	Timezone               string    `json:"timezone"`
	ContactEmail           string    `json:"contact_email"`
	ContactPhone           string    `json:"contact_phone"`
	LogoUrl                string    `json:"logo_url"`
	IsActive               bool      `json:"is_active"`
	CreatedAt              time.Time `json:"created_at"`
	FacilityCount          int32     `json:"facility_count"`
	AdminCount             int32     `json:"admin_count"`
	PendingInvitationCount int32     `json:"pending_invitation_count"`
	BookingCount           int32     `json:"booking_count"`
}

type TenantsResponse struct {
	Tenants []TenantSummaryDto `json:"tenants"`
}

type MemberDto struct {
	UserId   string    `json:"user_id"`
	Email    string    `json:"email"`
	FullName string    `json:"full_name"`
	Role     string    `json:"role"`
	Since    time.Time `json:"since"`
}

type InvitationDto struct {
	Id        string    `json:"id"`
	Email     string    `json:"email"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"created_at"`
}

type TenantDetailDto struct {
	Slug             string          `json:"slug"`
	Name             string          `json:"name"`
	Timezone         string          `json:"timezone"`
	ContactEmail     string          `json:"contact_email"`
	ContactPhone     string          `json:"contact_phone"`
	MunicipalityCode string          `json:"municipality_code"`
	LogoRef          string          `json:"logo_ref"` // sent back unchanged when the form is saved
	LogoUrl          string          `json:"logo_url"`
	IsActive         bool            `json:"is_active"`
	BookingCount     int64           `json:"booking_count"`
	Admins           []MemberDto     `json:"admins"`
	Invitations      []InvitationDto `json:"invitations"`
}

type SuperadminDto struct {
	UserId    string    `json:"user_id"`
	Email     string    `json:"email"`
	FullName  string    `json:"full_name"`
	FromEnv   bool      `json:"from_env"` // SUPERADMIN_EMAIL: cannot be removed here
	CreatedAt time.Time `json:"created_at"`
}

type SuperadminsResponse struct {
	Superadmins []SuperadminDto `json:"superadmins"`
}

// validMunicipalityCode: an INE municipality code, five digits.
func validMunicipalityCode(code string) bool {
	if len(code) != 5 {
		return false
	}
	for _, r := range code {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}
