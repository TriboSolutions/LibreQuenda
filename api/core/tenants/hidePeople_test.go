package tenants

import "testing"

// Signed out, a public booking says who answers for it and how many are in it, never who joined.
func TestHidePeople(t *testing.T) {
	var day = FacilityDayDto{PublicBookings: []PublicTurnDto{
		{Kind: "public", People: []string{"Antón", "María"}, PeopleCount: 2, Owner: &PublicPersonDto{Name: "Antón"}, Joined: []PublicPersonDto{{Name: "María"}}},
		{Kind: "event", People: []string{"Uxía"}, PeopleCount: 1, Joined: []PublicPersonDto{{Name: "Uxía"}}},
	}}
	day.HidePeople()
	var public, event = day.PublicBookings[0], day.PublicBookings[1]
	if len(public.People) != 1 || public.People[0] != "Antón" || len(public.Joined) != 0 || public.PeopleCount != 2 || public.Owner == nil {
		t.Errorf("public booking = %+v, want only the owner, count kept", public)
	}
	if len(event.People) != 0 || len(event.Joined) != 0 || event.PeopleCount != 1 {
		t.Errorf("event = %+v, want nobody named, count kept", event)
	}
}
