package dbconn

import (
	"database/sql"
	"fmt"
	"net"
	"net/url"
	"os"
	"time"

	_ "github.com/lib/pq"
)

const (
	maxOpenConnections    = 25
	maxIdleConnections    = 5
	connectionMaxLifetime = 5 * time.Minute
	connectionMaxIdleTime = 5 * time.Minute
)

// Connect opens the PostgreSQL connection, configures its pool, and verifies
// that the database is available before returning it.
func Connect() (*sql.DB, error) {
	conn, err := sql.Open("postgres", dataSourceName())
	if err != nil {
		return nil, fmt.Errorf("open database connection: %w", err)
	}

	if err := conn.Ping(); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("database is unavailable: %w", err)
	}

	conn.SetMaxOpenConns(maxOpenConnections)
	conn.SetMaxIdleConns(maxIdleConnections)
	conn.SetConnMaxLifetime(connectionMaxLifetime)
	conn.SetConnMaxIdleTime(connectionMaxIdleTime)

	return conn, nil
}

func dataSourceName() string {
	sslMode := os.Getenv("DB_SSLMODE")
	if sslMode == "" {
		sslMode = "disable"
	}

	dsnURL := url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(os.Getenv("DB_USER"), os.Getenv("DB_PASSWORD")),
		Host:   net.JoinHostPort(os.Getenv("DB_HOST"), os.Getenv("DB_PORT")),
		Path:   os.Getenv("DB_NAME"),
	}
	query := dsnURL.Query()
	query.Set("sslmode", sslMode)
	dsnURL.RawQuery = query.Encode()

	return dsnURL.String()
}
