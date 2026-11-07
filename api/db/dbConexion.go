package db

import (
	"database/sql"
	"librequenda/shared"
	"strings"
	"time"

	_ "github.com/lib/pq"
)

func GetConnectionString() string {

	if shared.DB_USER == "" {
		panic("DB_USER environment variable is not set")
	}

	if shared.DB_PASSWORD == "" {
		panic("DB_PASSWORD environment variable is not set")
	}

	if shared.POSTGRES_DB == "" {
		panic("POSTGRES_DB environment variable is not set")
	}
	var sb strings.Builder
	sb.WriteString("host=db")
	sb.WriteString(" port=5432")
	sb.WriteString(" user=")
	sb.WriteString(shared.DB_USER)
	sb.WriteString(" password=")
	sb.WriteString(shared.DB_PASSWORD)
	sb.WriteString(" dbname=")
	sb.WriteString(shared.POSTGRES_DB)
	sb.WriteString(" sslmode=disable")
	return sb.String()

}
func EstablishConnection() *sql.DB {
	var logger = shared.Logger
	connStr := GetConnectionString()
	var db *sql.DB
	var err error
	maxAttempts := 10
	for i := range maxAttempts {
		db, err = sql.Open("postgres", connStr)
		if err == nil {
			err = db.Ping()
			if err == nil {
				logger.Info("database connection established successfully")
				db.SetMaxOpenConns(40)
				db.SetMaxIdleConns(5)
				db.SetConnMaxLifetime(5 * time.Minute)
				return db
			}
			db.Close()
		}
		wait := 2*i + 1
		logger.Error("could not establish database connection, retrying in %d seconds: %s", wait, err.Error())
		time.Sleep(time.Duration(wait) * time.Second)
	}
	panic("could not establish database connection: " + err.Error())
}
