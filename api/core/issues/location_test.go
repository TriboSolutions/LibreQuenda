package issues

import (
	"database/sql"
	"librequenda/issue"
	"librequenda/model"
	"testing"
)

func TestAtFacility(t *testing.T) {
	var pinned = model.GetFacilityForIssueRow{
		Latitude:  sql.NullFloat64{Float64: 42.66453, Valid: true},
		Longitude: sql.NullFloat64{Float64: -8.10815, Valid: true},
		RadiusM:   100,
	}
	var at = func(lat, lon, accuracy float64) issue.Report {
		return issue.Report{Latitude: &lat, Longitude: &lon, Accuracy: &accuracy}
	}
	var cases = []struct {
		name     string
		input    issue.Report
		facility model.GetFacilityForIssueRow
		want     string
	}{
		{"at the door", at(42.66460, -8.10820, 10), pinned, ""},
		{"no location sent", issue.Report{}, pinned, "location_required"},
		{"a blurry fix", at(42.66453, -8.10815, 900), pinned, "location_inaccurate"},
		{"in another town", at(43.3623, -8.4115, 10), pinned, "too_far"},
		{"a facility without a pin is not checked", issue.Report{}, model.GetFacilityForIssueRow{}, ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := atFacility(c.input, c.facility); got != c.want {
				t.Errorf("got %q, want %q", got, c.want)
			}
		})
	}
}
