package counciladmin

import (
	"database/sql"
	"librequenda/audit"
	"librequenda/middlewares"
	"net/http"
	"sync"

	"github.com/angelbarreiros/Penguin/router"
)

type counciladminRoutes struct {
	db     *sql.DB
	router *router.Router
}

var routesOnce *sync.Once = new(sync.Once)
var counciladminRoutesInstance *counciladminRoutes

func CounciladminRoutes(db *sql.DB, router *router.Router) *counciladminRoutes {
	routesOnce.Do(func() {
		counciladminRoutesInstance = &counciladminRoutes{
			db:     db,
			router: router,
		}
	})
	return counciladminRoutesInstance
}

// Council administration: every handler checks "admin of this council, or superadmin".
func (r *counciladminRoutes) RegisterRoutes() {
	var h = CounciladminHandler(r.db)
	var routes = []router.Route{
		{Path: "/tenants/{slug}/admin/facilities", Method: http.MethodGet, Handler: h.ListFacilities},
		{Path: "/tenants/{slug}/admin/facilities", Method: http.MethodPost, Handler: h.CreateFacility},
		{Path: "/tenants/{slug}/admin/facilities/{id}", Method: http.MethodGet, Handler: h.GetFacility},
		{Path: "/tenants/{slug}/admin/facilities/{id}", Method: http.MethodPut, Handler: h.UpdateFacility},
		{Path: "/tenants/{slug}/admin/facilities/{id}", Method: http.MethodDelete, Handler: h.DeleteFacility},
		{Path: "/tenants/{slug}/admin/facilities/{id}/duplicate", Method: http.MethodPost, Handler: h.DuplicateFacility},
		{Path: "/tenants/{slug}/admin/facilities/{id}/schedule", Method: http.MethodPut, Handler: h.ReplaceSchedule},
		{Path: "/tenants/{slug}/admin/facilities/{id}/devices", Method: http.MethodGet, Handler: h.ListDevices},
		{Path: "/tenants/{slug}/admin/facilities/{id}/devices", Method: http.MethodPost, Handler: h.CreateDevice},
		{Path: "/tenants/{slug}/admin/facilities/{id}/devices/{deviceId}", Method: http.MethodDelete, Handler: h.DeleteDevice},
		{Path: "/tenants/{slug}/admin/facilities/{id}/checkin", Method: http.MethodGet, Handler: h.GetCheckinSettings},
		{Path: "/tenants/{slug}/admin/facilities/{id}/checkin", Method: http.MethodPut, Handler: h.SaveCheckinSettings},
		{Path: "/tenants/{slug}/admin/facilities/{id}/checkin/qr", Method: http.MethodPost, Handler: h.RotateCheckinQr},
		{Path: "/tenants/{slug}/admin/facilities/{id}/checkin/nfc", Method: http.MethodPost, Handler: h.RotateCheckinNfc},
		{Path: "/tenants/{slug}/admin/facilities/{id}/checkin/code", Method: http.MethodPost, Handler: h.RotateCheckinCode},
		{Path: "/tenants/{slug}/admin/facilities/{id}/active", Method: http.MethodPut, Handler: h.SetActive},
		{Path: "/tenants/{slug}/admin/facilities/{id}/closures/preview", Method: http.MethodPost, Handler: h.PreviewClosure},
		{Path: "/tenants/{slug}/admin/facilities/{id}/closures", Method: http.MethodPost, Handler: h.CreateClosure},
		{Path: "/tenants/{slug}/admin/facilities/{id}/closures/{closureId}", Method: http.MethodDelete, Handler: h.DeleteClosure},
		{Path: "/tenants/{slug}/admin/facilities/{id}/bookings", Method: http.MethodGet, Handler: h.GetStaffDay},
		{Path: "/tenants/{slug}/admin/bookings", Method: http.MethodPost, Handler: h.CreateStaffBooking},
		{Path: "/tenants/{slug}/admin/bookings/{bookingId}/cancel", Method: http.MethodPost, Handler: h.CancelStaffBooking},
		{Path: "/tenants/{slug}/admin/bookings/{bookingId}/participants", Method: http.MethodPost, Handler: h.AddEventParticipant},
		// the analytics assistant: a question about this council's data
		{Path: "/tenants/{slug}/admin/analytics/ask", Method: http.MethodPost, Handler: h.AskAnalytics},
		{Path: "/tenants/{slug}/admin/bookings/{bookingId}/checkin", Method: http.MethodPost, Handler: h.StaffCheckIn},
		{Path: "/tenants/{slug}/admin/bookings/{bookingId}/checkin", Method: http.MethodDelete, Handler: h.UndoStaffCheckIn},
		{Path: "/tenants/{slug}/admin/events", Method: http.MethodGet, Handler: h.ListEvents},
		{Path: "/tenants/{slug}/admin/events", Method: http.MethodPost, Handler: h.CreateEvent},
		{Path: "/tenants/{slug}/admin/events/preview", Method: http.MethodPost, Handler: h.PreviewEvent},
		{Path: "/tenants/{slug}/admin/events/{bookingId}", Method: http.MethodPut, Handler: h.UpdateEvent},
		{Path: "/tenants/{slug}/admin/events/{bookingId}/people", Method: http.MethodGet, Handler: h.EventPeople},
		{Path: "/tenants/{slug}/admin/events/{bookingId}/people/{participantId}", Method: http.MethodDelete, Handler: h.RemoveEventParticipant},
		{Path: "/tenants/{slug}/admin/indicators", Method: http.MethodGet, Handler: h.GetIndicators},
		{Path: "/tenants/{slug}/admin/indicators/export", Method: http.MethodGet, Handler: h.ExportIndicators},
		{Path: "/tenants/{slug}/admin/export", Method: http.MethodGet, Handler: middlewares.WithStrictRateLimit(h.ExportCouncil)},
		{Path: "/tenants/{slug}/admin/facilities/{id}/booking-rules", Method: http.MethodPut, Handler: h.SaveFacilityRules},
		{Path: "/tenants/{slug}/admin/issues", Method: http.MethodGet, Handler: h.ListIssues},
		{Path: "/tenants/{slug}/admin/issues/count", Method: http.MethodGet, Handler: h.CountNewIssues},
		{Path: "/tenants/{slug}/admin/issues/{issueId}", Method: http.MethodPut, Handler: h.ChangeIssue},
		{Path: "/tenants/{slug}/admin/notices", Method: http.MethodGet, Handler: h.ListNotices},
		{Path: "/tenants/{slug}/admin/notices", Method: http.MethodPost, Handler: h.CreateNotice},
		{Path: "/tenants/{slug}/admin/notices/{noticeId}", Method: http.MethodDelete, Handler: h.DeleteNotice},
		{Path: "/tenants/{slug}/admin/polls", Method: http.MethodGet, Handler: h.ListPolls},
		{Path: "/tenants/{slug}/admin/polls", Method: http.MethodPost, Handler: h.CreatePoll},
		{Path: "/tenants/{slug}/admin/polls/{pollId}/close", Method: http.MethodPost, Handler: h.ClosePoll},
		{Path: "/tenants/{slug}/admin/polls/{pollId}", Method: http.MethodDelete, Handler: h.DeletePoll},
		{Path: "/tenants/{slug}/admin/booking-rules", Method: http.MethodGet, Handler: h.GetBookingRules},
		{Path: "/tenants/{slug}/admin/booking-rules", Method: http.MethodPut, Handler: h.SaveBookingRules},
		{Path: "/tenants/{slug}/admin/audit", Method: http.MethodGet, Handler: h.GetAudit},
		{Path: "/tenants/{slug}/admin/residents", Method: http.MethodGet, Handler: h.ListResidents},
		{Path: "/tenants/{slug}/admin/residents", Method: http.MethodPost, Handler: h.AddResident},
		{Path: "/tenants/{slug}/admin/residents/{userId}", Method: http.MethodDelete, Handler: h.RemoveResident},
		{Path: "/tenants/{slug}/admin/resident-requests", Method: http.MethodGet, Handler: h.ListResidentRequests},
		{Path: "/tenants/{slug}/admin/resident-requests/count", Method: http.MethodGet, Handler: h.CountResidentRequests},
		{Path: "/tenants/{slug}/admin/resident-requests/{requestId}/verify", Method: http.MethodPost, Handler: h.VerifyResidentRequest},
		{Path: "/tenants/{slug}/admin/resident-requests/{requestId}", Method: http.MethodDelete, Handler: h.DiscardResidentRequest},
		{Path: "/tenants/{slug}/admin/smtp", Method: http.MethodGet, Handler: h.GetSmtp},
		{Path: "/tenants/{slug}/admin/smtp", Method: http.MethodPut, Handler: h.SaveSmtp},
		{Path: "/tenants/{slug}/admin/smtp", Method: http.MethodDelete, Handler: h.DeleteSmtp},
		{Path: "/tenants/{slug}/admin/smtp/test", Method: http.MethodPost, Handler: middlewares.WithStrictRateLimit(h.TestSmtp)},
		{Path: "/geocode", Method: http.MethodGet, Handler: h.Geocode},
	}
	for _, route := range routes {
		// every change is logged once it succeeded (audit), inside the authentication
		route.Handler = middlewares.AuthMiddleware(audit.Route(r.db, string(route.Method), route.Path, route.Handler))
		route.AdditionalMethods = []router.HTTPMethod{router.OPTIONS}
		r.router.NewRoute(route)
	}
}
