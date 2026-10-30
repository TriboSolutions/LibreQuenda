package platform

import (
	"context"
	"librequenda/model"
	"librequenda/shared"
	"net/http"
)

// BookingPolicyDto: the booking rules that span every council, set by the
// superadministrators. OneCouncilPerAccount: a person holds upcoming bookings
// (own or joined) in one council at a time (api/core/bookings.CouncilLimits).
type BookingPolicyDto struct {
	OneCouncilPerAccount bool `json:"one_council_per_account"`
}

func (s *platformService) GetBookingPolicy(ctx context.Context) (*BookingPolicyDto, *shared.ErrorResponse) {
	var one, err = model.New(s.db).GetBookingPolicy(ctx)
	if err != nil {
		return nil, &shared.ErrorResponse{Code: http.StatusInternalServerError, Message: "errors.booking_policy_failed"}
	}
	return &BookingPolicyDto{OneCouncilPerAccount: one}, nil
}

func (s *platformService) SaveBookingPolicy(ctx context.Context, input BookingPolicyDto) (*BookingPolicyDto, *shared.ErrorResponse) {
	if err := model.New(s.db).SaveBookingPolicy(ctx, input.OneCouncilPerAccount); err != nil {
		return nil, &shared.ErrorResponse{Code: http.StatusInternalServerError, Message: "errors.booking_policy_failed"}
	}
	return &input, nil
}
