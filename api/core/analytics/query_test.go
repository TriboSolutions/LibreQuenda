package analytics

import "testing"

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
