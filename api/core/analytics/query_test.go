package analytics

import (
	"strings"
	"testing"
)

func TestCheck(t *testing.T) {
	for _, ok := range []string{
		"SELECT 1", "select count(*) from booking;", "  WITH x AS (SELECT 1) SELECT * FROM x",
		"-- bookings per day\nSELECT 1", "/* comment */ SELECT 1",
	} {
		if _, err := Check(ok); err != nil {
			t.Errorf("Check(%q): %v", ok, err)
		}
	}
	for _, bad := range []string{
		"", ";", "DELETE FROM booking", "SELECT 1; DROP TABLE booking", "SET analytics.superadmin = 'on'",
		"-- SELECT\nUPDATE booking SET status = 'cancelled'", "COPY booking TO STDOUT", "BEGIN", "RESET ROLE",
	} {
		if _, err := Check(bad); err == nil {
			t.Errorf("Check(%q) passed", bad)
		}
	}
}

func TestOffTopic(t *testing.T) {
	var long = strings.Repeat("Aquí tes unha lista de tarefas en React. ", 20)
	var ran = []Step{{SQL: "SELECT 1", Result: &Result{}}}
	var failed = []Step{{SQL: "SELECT nope", Error: "column does not exist"}}
	for _, c := range []struct {
		name   string
		answer string
		steps  []Step
		want   bool
	}{
		{"a figure from a query", "Houbo **12** cancelacións en outubro.", ran, false},
		{"a long answer from a query", long, ran, false},
		{"a short clarification", "De que período? Do último mes ou do ano?", nil, false},
		{"code, even after a query", "Aquí tes:\n```jsx\nexport default function Todo() {}\n```", ran, true},
		{"code with tildes", "~~~\nprint(1)\n~~~", nil, true},
		{"a long answer with no query", long, nil, true},
		{"a long answer whose queries all failed", long, failed, true},
	} {
		if got := offTopic(c.answer, c.steps); got != c.want {
			t.Errorf("%s: offTopic = %v, want %v", c.name, got, c.want)
		}
	}
}
