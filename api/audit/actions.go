package audit

// action: the stable code of what a route does, and the path parameter that
// names its target. An empty code means "not worth logging" (a preview).
type action struct {
	code   string
	target string // facility, device, closure, booking, tenant, user, tag, tag_category, invitation
	param  string // the path parameter holding the target's id
}

// actions lists the administrative routes by "METHOD path", as registered.
// A change through a route missing here is still logged, as "other".
var actions = map[string]action{
	// council administration
	"POST /tenants/{slug}/admin/facilities":                             {code: "facility.created"},
	"PUT /tenants/{slug}/admin/facilities/{id}":                         {code: "facility.updated", target: "facility", param: "id"},
	"DELETE /tenants/{slug}/admin/facilities/{id}":                      {code: "facility.deleted", target: "facility", param: "id"},
	"PUT /tenants/{slug}/admin/facilities/{id}/schedule":                {code: "facility.schedule_changed", target: "facility", param: "id"},
	"PUT /tenants/{slug}/admin/facilities/{id}/active":                  {code: "facility.activation_changed", target: "facility", param: "id"},
	"POST /tenants/{slug}/admin/facilities/{id}/devices":                {code: "reader.added", target: "facility", param: "id"},
	"DELETE /tenants/{slug}/admin/facilities/{id}/devices/{deviceId}":   {code: "reader.removed", target: "facility", param: "id"},
	"PUT /tenants/{slug}/admin/facilities/{id}/checkin":                 {code: "checkin.settings_changed", target: "facility", param: "id"},
	"POST /tenants/{slug}/admin/facilities/{id}/checkin/qr":             {code: "checkin.qr_renewed", target: "facility", param: "id"},
	"POST /tenants/{slug}/admin/facilities/{id}/checkin/code":           {code: "checkin.code_renewed", target: "facility", param: "id"},
	"POST /tenants/{slug}/admin/facilities/{id}/closures/preview":       {code: ""},
	"POST /tenants/{slug}/admin/facilities/{id}/closures":               {code: "closure.created", target: "facility", param: "id"},
	"DELETE /tenants/{slug}/admin/facilities/{id}/closures/{closureId}": {code: "closure.deleted", target: "facility", param: "id"},
	"PUT /tenants/{slug}/admin/facilities/{id}/booking-rules":           {code: "booking_window.facility_changed", target: "facility", param: "id"},
	"PUT /tenants/{slug}/admin/booking-rules":                           {code: "booking_window.council_changed"},
	"POST /tenants/{slug}/admin/bookings":                               {code: "staff.booking_created"},
	"POST /tenants/{slug}/admin/bookings/{bookingId}/cancel":            {code: "staff.booking_cancelled", target: "booking", param: "bookingId"},
	"POST /tenants/{slug}/admin/bookings/{bookingId}/checkin":           {code: "staff.checked_in", target: "booking", param: "bookingId"},
	"DELETE /tenants/{slug}/admin/bookings/{bookingId}/checkin":         {code: "staff.check_in_undone", target: "booking", param: "bookingId"},
	"PUT /tenants/{slug}/admin/smtp":                                    {code: "smtp.changed"},
	"DELETE /tenants/{slug}/admin/smtp":                                 {code: "smtp.removed"},
	"POST /tenants/{slug}/admin/smtp/test":                              {code: "smtp.tested"},
	"GET /tenants/{slug}/admin/export":                                  {code: "council.exported"},
	"GET /tenants/{slug}/admin/indicators/export":                       {code: "indicators.exported"},
	"GET /tenants/{slug}/admin/audit":                                   {code: ""},

	// superadministrators
	"POST /admin/tenants":                           {code: "council.created"},
	"PUT /admin/tenants/{slug}":                     {code: "council.updated"},
	"DELETE /admin/tenants/{slug}":                  {code: "council.deleted"},
	"POST /admin/tenants/{slug}/invitations":        {code: "admin.invited"},
	"DELETE /admin/tenants/{slug}/invitations/{id}": {code: "admin.invitation_withdrawn", target: "invitation", param: "id"},
	"DELETE /admin/tenants/{slug}/admins/{userId}":  {code: "admin.removed", target: "user", param: "userId"},
	"POST /admin/superadmins":                       {code: "superadmin.added"},
	"DELETE /admin/superadmins/{userId}":            {code: "superadmin.removed", target: "user", param: "userId"},
	"POST /admin/tag-categories":                    {code: "tag_category.created"},
	"PUT /admin/tag-categories/{id}":                {code: "tag_category.updated", target: "tag_category", param: "id"},
	"DELETE /admin/tag-categories/{id}":             {code: "tag_category.deleted", target: "tag_category", param: "id"},
	"POST /admin/tags":                              {code: "tag.created"},
	"PUT /admin/tags/{id}":                          {code: "tag.updated", target: "tag", param: "id"},
	"DELETE /admin/tags/{id}":                       {code: "tag.deleted", target: "tag", param: "id"},
	"PUT /admin/branding":                           {code: "branding.changed"},
	"GET /admin/audit":                              {code: ""},

	// residents: only what concerns their account as a whole, without content
	"GET /me/export": {code: "account.exported"},
	"DELETE /me":     {code: "account.deleted"},
}
