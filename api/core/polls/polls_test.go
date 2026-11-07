package polls

import (
	"librequenda/model"
	"testing"
)

func TestMayVote(t *testing.T) {
	var open = model.GetPollForVoteRow{IsOpen: true, EmailVerified: true}
	if key := mayVote(open); key != "" {
		t.Errorf("open poll, verified email: %s", key)
	}
	var cases = map[string]model.GetPollForVoteRow{
		"errors.poll_closed":         {IsOpen: false, EmailVerified: true},
		"errors.email_unverified":    {IsOpen: true},
		"errors.poll_residents_only": {IsOpen: true, EmailVerified: true, ResidentsOnly: true},
	}
	for want, poll := range cases {
		if key := mayVote(poll); key != want {
			t.Errorf("%+v: %q, want %q", poll, key, want)
		}
	}
	var resident = model.GetPollForVoteRow{IsOpen: true, EmailVerified: true, ResidentsOnly: true, IsResident: true}
	if key := mayVote(resident); key != "" {
		t.Errorf("verified resident: %s", key)
	}
}
