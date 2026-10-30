package bookings

import (
	"librequenda/model"
	"librequenda/schedule"
	"testing"
	"time"
)

func TestTurnsInARow(t *testing.T) {
	var at = func(h int) time.Time { return time.Date(2026, 11, 2, h, 0, 0, 0, time.UTC) }
	var theirs = []model.ListUserBookingsNearRow{{StartsAt: at(9), EndsAt: at(10)}, {StartsAt: at(10), EndsAt: at(11)}, {StartsAt: at(13), EndsAt: at(14)}}
	if got := turnsInARow(theirs, schedule.Slot{Start: at(11), End: at(12)}); got != 3 {
		t.Errorf("after 9-10 and 10-11: want 3 in a row, got %d", got)
	}
	if got := turnsInARow(theirs, schedule.Slot{Start: at(12), End: at(13)}); got != 2 {
		t.Errorf("right before 13-14: want 2 in a row, got %d", got)
	}
	if got := turnsInARow(nil, schedule.Slot{Start: at(12), End: at(13)}); got != 1 {
		t.Errorf("alone: want 1, got %d", got)
	}
}

func TestNoShowBlock(t *testing.T) {
	var now = time.Date(2026, 11, 2, 12, 0, 0, 0, time.UTC)
	var days = func(n int) time.Time { return now.AddDate(0, 0, -n) }
	if _, blocked := noShowBlock([]time.Time{days(1), days(5)}, 3, 7, now); blocked {
		t.Error("two no-shows under a limit of three: not blocked")
	}
	var until, blocked = noShowBlock([]time.Time{days(2), days(5), days(9)}, 3, 7, now)
	if !blocked || !until.Equal(days(2).AddDate(0, 0, 7)) {
		t.Errorf("three no-shows: blocked for 7 days after the latest, got %v %v", until, blocked)
	}
	if _, blocked := noShowBlock([]time.Time{days(8), days(10), days(12)}, 3, 7, now); blocked {
		t.Error("the block is over 7 days after the latest no-show")
	}
}
