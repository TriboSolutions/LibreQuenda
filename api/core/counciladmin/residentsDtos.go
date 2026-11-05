package counciladmin

import "time"

// --- verified residents ----------------------------------------------------

// ResidentDto: someone the council verified as its resident.
type ResidentDto struct {
	UserId     string    `json:"user_id"`
	Email      string    `json:"email"`
	FullName   string    `json:"full_name"`
	VerifiedAt time.Time `json:"verified_at"`
	VerifiedBy string    `json:"verified_by"` // the administrator's name, "" once their account is gone
}

type ResidentsResponse struct {
	Residents []ResidentDto `json:"residents"`
	Total     int64         `json:"total"` // matching the search, across pages
	Page      int           `json:"page"`
	PageSize  int           `json:"page_size"`
}

type ResidentInput struct {
	Email string `json:"email"`
}

const residentsPageSize = 50

// residentSorts: ?sort= on the list; name (A-Z) is the default.
var residentSorts = map[string]bool{"name": true, "recent": true}
