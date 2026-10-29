package counciladmin

import (
	"librequenda/core/reader"
	"net/http"
	"testing"
)

func TestStaffCheckinRefusal(t *testing.T) {
	if got := staffCheckinRefusal(reader.ResultCheckedIn); got != nil {
		t.Fatalf("checked in, but refused with %+v", got)
	}
	var cases = map[string]string{
		reader.ResultAlreadyCheckedIn: "errors.already_checked_in",
		reader.ResultTooEarly:         "errors.staff_checkin_too_early",
		reader.ResultEnded:            "errors.staff_checkin_ended",
		reader.ResultCancelled:        "errors.booking_already_cancelled",
	}
	for result, key := range cases {
		var got = staffCheckinRefusal(result)
		if got == nil || got.Code != http.StatusConflict || got.Message != key {
			t.Errorf("%s: refusal = %+v, want 409 %s", result, got, key)
		}
	}
	if got := staffCheckinRefusal(reader.ResultOtherFacility); got == nil || got.Code != http.StatusInternalServerError {
		t.Errorf("unexpected result: refusal = %+v, want 500", got)
	}
}
