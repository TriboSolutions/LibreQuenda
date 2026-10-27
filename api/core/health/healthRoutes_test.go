package health

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type healthTestConnector struct {
	pingErr error
}

func (c *healthTestConnector) Connect(context.Context) (driver.Conn, error) {
	return &healthTestConnection{pingErr: c.pingErr}, nil
}

func (c *healthTestConnector) Driver() driver.Driver {
	return healthTestDriver{pingErr: c.pingErr}
}

type healthTestDriver struct {
	pingErr error
}

func (d healthTestDriver) Open(string) (driver.Conn, error) {
	return &healthTestConnection{pingErr: d.pingErr}, nil
}

type healthTestConnection struct {
	pingErr error
}

func (c *healthTestConnection) Prepare(string) (driver.Stmt, error) {
	return nil, errors.New("prepare is not supported")
}

func (c *healthTestConnection) Close() error {
	return nil
}

func (c *healthTestConnection) Begin() (driver.Tx, error) {
	return nil, errors.New("transactions are not supported")
}

func (c *healthTestConnection) Ping(context.Context) error {
	return c.pingErr
}

func TestAPIHealthHandler(t *testing.T) {
	var recorder *httptest.ResponseRecorder = httptest.NewRecorder()
	var request *http.Request = httptest.NewRequest(http.MethodGet, "/health/api", nil)

	apiHealthHandler(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}
}

func TestDatabaseHealthHandler(t *testing.T) {
	var tests []struct {
		name           string
		pingErr        error
		expectedStatus int
		expectedBody   string
	} = []struct {
		name           string
		pingErr        error
		expectedStatus int
		expectedBody   string
	}{
		{name: "database available", expectedStatus: http.StatusOK},
		{
			name:           "database unavailable",
			pingErr:        errors.New("database unavailable"),
			expectedStatus: http.StatusServiceUnavailable,
			expectedBody:   `{"status":"db down"}`,
		},
	}

	var test struct {
		name           string
		pingErr        error
		expectedStatus int
		expectedBody   string
	}
	for _, test = range tests {
		t.Run(test.name, func(t *testing.T) {
			var db *sql.DB = sql.OpenDB(&healthTestConnector{pingErr: test.pingErr})
			defer db.Close()

			var recorder *httptest.ResponseRecorder = httptest.NewRecorder()
			var request *http.Request = httptest.NewRequest(http.MethodGet, "/health/db", nil)
			databaseHealthHandler(db)(recorder, request)

			if recorder.Code != test.expectedStatus {
				t.Fatalf("expected status %d, got %d", test.expectedStatus, recorder.Code)
			}
			if test.expectedBody != "" && !strings.Contains(recorder.Body.String(), test.expectedBody) {
				t.Fatalf("expected body to contain %q, got %q", test.expectedBody, recorder.Body.String())
			}
		})
	}
}
