package counciladmin

import (
	"strings"
	"testing"
	"time"
)

func TestPollInput(t *testing.T) {
	var question, options, err = CleanPollInput(PollInput{Question: "  Que horario?  ", Options: []string{" Mañá ", "", "Tarde"}})
	if err != nil || question != "Que horario?" || len(options) != 2 || options[0] != "Mañá" {
		t.Fatalf("%q %q %+v", question, options, err)
	}
	for name, input := range map[string]PollInput{
		"no question":   {Question: " ", Options: []string{"a", "b"}},
		"long question": {Question: strings.Repeat("ñ", 201), Options: []string{"a", "b"}},
		"one option":    {Question: "?", Options: []string{"a", " "}},
		"seven options": {Question: "?", Options: []string{"1", "2", "3", "4", "5", "6", "7"}},
		"repeated":      {Question: "?", Options: []string{"Pádel", "pádel"}},
		"long option":   {Question: "?", Options: []string{"a", strings.Repeat("b", 121)}},
	} {
		if _, _, err = CleanPollInput(input); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
	// 200 runes, not bytes
	if _, _, err = CleanPollInput(PollInput{Question: strings.Repeat("ñ", 200), Options: []string{"a", "b"}}); err != nil {
		t.Errorf("200 characters refused: %+v", err)
	}
}

func TestEndOfDay(t *testing.T) {
	var madrid, _ = time.LoadLocation("Europe/Madrid")
	// 23:30 in Madrid on 7 November is already the 7th there, 22:30 UTC
	var now = time.Date(2026, 11, 7, 22, 30, 0, 0, time.UTC)
	var end, ok = endOfDay("2026-11-07", madrid, now, 90)
	if !ok || !end.Equal(time.Date(2026, 11, 8, 0, 0, 0, 0, madrid)) {
		t.Fatalf("today: %v %v", end, ok)
	}
	for _, day := range []string{"2026-11-06", "2027-02-06", "7/11/2026", ""} {
		if _, ok = endOfDay(day, madrid, now, 90); ok {
			t.Errorf("%q accepted", day)
		}
	}
	if _, ok = endOfDay("2027-02-05", madrid, now, 90); !ok {
		t.Error("90 days ahead refused")
	}
}

func TestHistoryState(t *testing.T) {
	for state, want := range map[string][2]bool{"": {false, true}, "active": {false, true}, "past": {true, true}, "open": {false, false}, "PAST": {false, false}} {
		var past, valid = historyState(state, "active")
		if past != want[0] || valid != want[1] {
			t.Errorf("%q: past %v valid %v, want %v", state, past, valid, want)
		}
	}
}
