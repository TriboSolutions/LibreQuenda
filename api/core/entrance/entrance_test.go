package entrance

import (
	"database/sql"
	"librequenda/checkin"
	"librequenda/model"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestProve(t *testing.T) {
	var secret = []byte("a facility secret of 32 bytes...")
	var token = uuid.New()
	var turn = time.Date(2026, 11, 3, 18, 0, 0, 0, time.UTC)
	var facility = model.GetFacilityEntranceRow{
		Latitude:   sql.NullFloat64{Float64: 42.66453, Valid: true},
		Longitude:  sql.NullFloat64{Float64: -8.10815, Valid: true},
		PrintedQr:  true,
		QrToken:    uuid.NullUUID{UUID: token, Valid: true},
		TurnCodes:  true,
		CodeSecret: secret,
		RadiusM:    100,
	}
	var lat, lon, accuracy, far = 42.66453, -8.10815, 15.0, 42.67
	var here = CheckinInput{Latitude: &lat, Longitude: &lon, Accuracy: &accuracy}
	var with = func(change func(*CheckinInput)) CheckinInput {
		var input = here
		change(&input)
		return input
	}
	var qrOff = facility
	qrOff.PrintedQr = false
	var codesOff = facility
	codesOff.TurnCodes = false
	var noPin = facility
	noPin.Latitude, noPin.Longitude = sql.NullFloat64{}, sql.NullFloat64{}
	var open = []time.Time{turn}

	var cases = []struct {
		name     string
		facility model.GetFacilityEntranceRow
		input    CheckinInput
		want     string
	}{
		{"printed QR at the door", facility, with(func(i *CheckinInput) { i.QrToken = token.String() }), ""},
		{"turn code at the door", facility, with(func(i *CheckinInput) { i.Code = checkin.TurnCode(secret, turn) }), ""},
		{"no proof", facility, here, ResultMethodOff},
		{"printed QR turned off", qrOff, with(func(i *CheckinInput) { i.QrToken = token.String() }), ResultMethodOff},
		{"old print", facility, with(func(i *CheckinInput) { i.QrToken = uuid.NewString() }), ResultInvalidQr},
		{"not a token", facility, with(func(i *CheckinInput) { i.QrToken = "door" }), ResultInvalidQr},
		{"codes turned off", codesOff, with(func(i *CheckinInput) { i.Code = checkin.TurnCode(secret, turn) }), ResultMethodOff},
		{"code of another turn", facility, with(func(i *CheckinInput) { i.Code = checkin.TurnCode(secret, turn.Add(time.Hour)) }), ResultWrongCode},
		{"facility without a pin", noPin, with(func(i *CheckinInput) { i.QrToken = token.String() }), ResultNoFacilityLocation},
		{"no location", facility, CheckinInput{QrToken: token.String()}, checkin.LocationMissing},
		{"far away", facility, with(func(i *CheckinInput) { i.QrToken = token.String(); i.Latitude = &far }), checkin.LocationTooFar},
	}
	for _, c := range cases {
		if got := prove(c.facility, c.input, open); got != c.want {
			t.Errorf("%s: got %q, want %q", c.name, got, c.want)
		}
	}
}
