package counciladmin

import (
	"librequenda/shared"
	"os"
	"testing"
)

// Storage links are signed locally: these make storage.URL work without Garage.
func TestMain(m *testing.M) {
	shared.PUBLIC_URL = "http://localhost"
	shared.STORAGE_ENDPOINT = "storage:3900"
	shared.STORAGE_REGION = "garage"
	shared.STORAGE_BUCKET = "media"
	shared.STORAGE_ACCESS_KEY = "GKtest"
	shared.STORAGE_SECRET_KEY = "secret"
	os.Exit(m.Run())
}
