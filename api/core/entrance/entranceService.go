package entrance

import (
	"context"
	"database/sql"
	"errors"
	"librequenda/auth"
	"librequenda/checkin"
	"librequenda/core/me"
	"librequenda/core/reader"
	"librequenda/model"
	"librequenda/shared"
	"math"
	"net/http"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"
)

type entranceService struct {
	db *sql.DB
}

var sInstance *entranceService
var sOnce *sync.Once = new(sync.Once)

func GetEntranceService(db *sql.DB) *entranceService {
	sOnce.Do(func() {
		sInstance = &entranceService{db: db}
	})
	return sInstance
}

var failed = &shared.ErrorResponse{Code: http.StatusInternalServerError, Message: "errors.checkin_failed"}

func (s *entranceService) facility(ctx context.Context, tenantSlug, facilitySlug string) (model.GetFacilityEntranceRow, *shared.ErrorResponse) {
	var facility, err = model.New(s.db).GetFacilityEntrance(ctx, model.GetFacilityEntranceParams{TenantSlug: tenantSlug, FacilitySlug: facilitySlug})
	if errors.Is(err, sql.ErrNoRows) {
		return facility, &shared.ErrorResponse{Code: http.StatusNotFound, Message: "errors.facility_not_found"}
	}
	if err != nil {
		return facility, failed
	}
	return facility, nil
}

// Get: the facility's ways in from the phone, and the person's booking that can go in now.
func (s *entranceService) Get(ctx context.Context, session auth.Session, tenantSlug, facilitySlug string) (*EntranceDto, *shared.ErrorResponse) {
	var facility, notFound = s.facility(ctx, tenantSlug, facilitySlug)
	if notFound != nil {
		return nil, notFound
	}
	var userId, userErr = me.CurrentUserId(ctx, s.db, session)
	if userErr != nil {
		return nil, userErr
	}
	var response = &EntranceDto{
		FacilityName: facility.Name,
		FacilitySlug: facility.Slug,
		TenantName:   facility.TenantName,
		TenantSlug:   facility.TenantSlug,
		Timezone:     facility.Timezone,
		PrintedQr:    facility.PrintedQr && facility.Latitude.Valid,
		EntryCode:    facility.EntryCode && facility.Latitude.Valid,
		RadiusM:      int(facility.RadiusM),
		FlashBooking: facility.FlashBooking,
	}
	var token, found, err = s.bookingToken(ctx, userId, facility.ID)
	if err != nil {
		return nil, err
	}
	if found {
		var booking, getErr = model.New(s.db).GetBookingByCheckinToken(ctx, token)
		if getErr != nil {
			return nil, failed
		}
		response.Booking = &EntranceBookingDto{StartsAt: booking.StartsAt, EndsAt: booking.EndsAt}
		if booking.CheckedInAt.Valid {
			response.Booking.CheckedInAt = &booking.CheckedInAt.Time
		}
	}
	return response, nil
}

// CheckIn: the phone proves it is at the door (printed QR or entry code, and its
// location); then the person's booking goes in, or, when asked, a free turn is
// booked on the spot. Both through the reader's own check-in.
func (s *entranceService) CheckIn(ctx context.Context, session auth.Session, tenantSlug, facilitySlug string, input CheckinInput) (*CheckinResultDto, *shared.ErrorResponse) {
	var facility, notFound = s.facility(ctx, tenantSlug, facilitySlug)
	if notFound != nil {
		return nil, notFound
	}
	var userId, userErr = me.CurrentUserId(ctx, s.db, session)
	if userErr != nil {
		return nil, userErr
	}
	if refused := prove(facility, input); refused != "" {
		return &CheckinResultDto{CheckinResultDto: reader.CheckinResultDto{Result: refused}}, nil
	}

	var readers = reader.GetReaderService(s.db)
	var token, found, err = s.bookingToken(ctx, userId, facility.ID)
	if err != nil {
		return nil, err
	}
	var entry = reader.Entry{Method: model.CheckinMethodEntryCode}
	if input.QrToken != "" {
		entry.Method = model.CheckinMethodPrintedQr
	}
	var result *reader.CheckinResultDto
	switch {
	case found:
		result, err = readers.CheckInBooking(ctx, facility.ID, entry, token)
	case !input.Book:
		return &CheckinResultDto{CheckinResultDto: reader.CheckinResultDto{Result: ResultNoBooking}, CanBook: facility.FlashBooking}, nil
	default:
		result, err = readers.FlashCheckIn(ctx, userId, "", facility.ID, entry)
	}
	if err != nil {
		return nil, err
	}
	return &CheckinResultDto{CheckinResultDto: *result}, nil
}

// prove checks the proof of being at the door; "" when it holds, else the result to answer.
func prove(facility model.GetFacilityEntranceRow, input CheckinInput) string {
	switch {
	case input.QrToken != "":
		if !facility.PrintedQr {
			return ResultMethodOff
		}
		var token, err = uuid.Parse(input.QrToken)
		if err != nil || !facility.QrToken.Valid || token != facility.QrToken.UUID {
			return ResultInvalidQr
		}
	case input.Code != "":
		if !facility.EntryCode {
			return ResultMethodOff
		}
		if !checkin.CodeMatches(facility.CodeSecret, input.Code) {
			return ResultWrongCode
		}
	default:
		return ResultMethodOff
	}
	if !facility.Latitude.Valid || !facility.Longitude.Valid {
		return ResultNoFacilityLocation
	}
	var position *checkin.Position
	if input.Latitude != nil && input.Longitude != nil && input.Accuracy != nil {
		position = &checkin.Position{Latitude: *input.Latitude, Longitude: *input.Longitude, Accuracy: *input.Accuracy}
	}
	return checkin.CheckLocation(position, facility.Latitude.Float64, facility.Longitude.Float64, int(facility.RadiusM))
}

// bookingToken: the check-in token of the person's booking here that can go in now.
func (s *entranceService) bookingToken(ctx context.Context, userId, facilityId uuid.UUID) (uuid.UUID, bool, *shared.ErrorResponse) {
	var token, err = model.New(s.db).GetUserCheckinTokenAtFacility(ctx, model.GetUserCheckinTokenAtFacilityParams{
		UserID:     uuid.NullUUID{UUID: userId, Valid: true},
		FacilityID: facilityId,
		WindowEnds: time.Now().Add(checkin.EarlyWindow),
	})
	if errors.Is(err, sql.ErrNoRows) {
		return uuid.Nil, false, nil
	}
	if err != nil {
		return uuid.Nil, false, failed
	}
	return token, true, nil
}

// Nearby: for a flash booking with the entry code, the facilities that take it
// and are within their radius of the phone, nearest first.
func (s *entranceService) Nearby(ctx context.Context, input NearbyInput) (*NearbyDto, *shared.ErrorResponse) {
	var response = &NearbyDto{Facilities: []NearbyFacilityDto{}}
	var position *checkin.Position
	if input.Latitude != nil && input.Longitude != nil && input.Accuracy != nil {
		position = &checkin.Position{Latitude: *input.Latitude, Longitude: *input.Longitude, Accuracy: *input.Accuracy}
	}
	// a radius never gets in the way of telling a missing or blurry location
	if problem := checkin.CheckLocation(position, 0, 0, math.MaxInt32); problem != "" {
		response.Result = problem
		return response, nil
	}
	var rows, err = model.New(s.db).ListFlashCodeFacilities(ctx)
	if err != nil {
		return nil, failed
	}
	response.Facilities = nearby(rows, *position)
	return response, nil
}

// nearby keeps the facilities the position could be at, nearest first.
func nearby(rows []model.ListFlashCodeFacilitiesRow, position checkin.Position) []NearbyFacilityDto {
	type near struct {
		row      model.ListFlashCodeFacilitiesRow
		distance float64
	}
	var found []near
	for _, row := range rows {
		if checkin.CheckLocation(&position, row.Latitude, row.Longitude, int(row.RadiusM)) == "" {
			found = append(found, near{row, checkin.DistanceMetres(position.Latitude, position.Longitude, row.Latitude, row.Longitude)})
		}
	}
	sort.Slice(found, func(i, j int) bool { return found[i].distance < found[j].distance })
	var facilities = make([]NearbyFacilityDto, 0, len(found))
	for _, f := range found {
		facilities = append(facilities, NearbyFacilityDto{FacilityName: f.row.Name, FacilitySlug: f.row.Slug, TenantName: f.row.TenantName, TenantSlug: f.row.TenantSlug})
	}
	return facilities
}
