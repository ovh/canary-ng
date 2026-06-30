//go:build e2e

package driver

import "testing"

func TestPostgresqlE2E(t *testing.T) {
	d, err := NewPostgresql(PostgresqlOpts{
		Hosts:    []string{e2eHost("POSTGRESQL", "127.0.0.1")},
		Port:     e2ePort("POSTGRESQL", 5432),
		Username: e2eEnv("POSTGRESQL", "USERNAME", "canary"),
		Password: e2eEnv("POSTGRESQL", "PASSWORD", "canary"),
		Database: e2eEnv("POSTGRESQL", "DATABASE", "canary"),
		SSLMode:  e2eEnv("POSTGRESQL", "SSLMODE", "disable"),
		Table:    "canary_ng",
		Create:   true,
	})
	if err != nil {
		t.Fatalf("new postgresql: %v", err)
	}

	runDriverE2E(t, d)
}
