package bookings

import (
	"context"
	"database/sql"
	"librequenda/model"
	"testing"

	"github.com/google/uuid"
)

type fakeCouncil struct {
	oneCouncil bool
	cap        int32
	other      string // "" = no booking in another council
	upcoming   int64
}

func (f fakeCouncil) GetCouncilBookingPolicy(context.Context, uuid.UUID) (model.GetCouncilBookingPolicyRow, error) {
	return model.GetCouncilBookingPolicyRow{OneCouncilPerAccount: f.oneCouncil, CouncilMaxActive: f.cap}, nil
}

func (f fakeCouncil) GetUserBookingInOtherCouncil(context.Context, model.GetUserBookingInOtherCouncilParams) (string, error) {
	if f.other == "" {
		return "", sql.ErrNoRows
	}
	return f.other, nil
}

func (f fakeCouncil) CountUserUpcomingInCouncil(context.Context, model.CountUserUpcomingInCouncilParams) (int64, error) {
	return f.upcoming, nil
}

func TestCouncilLimits(t *testing.T) {
	var user, council = uuid.New(), uuid.New()
	var cases = []struct {
		name string
		q    fakeCouncil
		own  bool
		want string
	}{
		{"nothing elsewhere, no cap", fakeCouncil{oneCouncil: true}, true, ""},
		{"bookings in another council", fakeCouncil{oneCouncil: true, other: "Concello de Lugo"}, true, "errors.other_council_booking"},
		{"joining someone's game is not held to one council", fakeCouncil{oneCouncil: true, other: "Concello de Lugo"}, false, ""},
		{"the platform allows several councils", fakeCouncil{oneCouncil: false, other: "Concello de Lugo"}, true, ""},
		{"under the council's cap", fakeCouncil{cap: 3, upcoming: 2}, true, ""},
		{"at the council's cap", fakeCouncil{cap: 3, upcoming: 3}, true, "errors.council_limit_reached"},
		{"joining does not count towards the cap", fakeCouncil{cap: 3, upcoming: 3}, false, ""},
		{"no cap", fakeCouncil{cap: 0, upcoming: 50}, true, ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var got = ""
			if refused := CouncilLimits(context.Background(), c.q, user, council, c.own); refused != nil {
				got = refused.Message
			}
			if got != c.want {
				t.Errorf("got %q, want %q", got, c.want)
			}
		})
	}
	var refused = CouncilLimits(context.Background(), fakeCouncil{oneCouncil: true, other: "Concello de Lugo"}, user, council, true)
	if len(refused.Args) != 1 || refused.Args[0] != "Concello de Lugo" {
		t.Errorf("the message names the other council: %v", refused.Args)
	}
}
