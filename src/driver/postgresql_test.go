package driver

import (
	"fmt"
	"testing"
)

func TestPostgresqlDSN(t *testing.T) {
	tests := []struct {
		name     string
		input    PostgresqlOpts
		expected string
	}{
		{"with dsn", PostgresqlOpts{DSN: "postgres://canary:password@127.0.0.1:5432/canary_db", Database: "canary_db", Table: "canary_table"}, "postgres://canary:password@127.0.0.1:5432/canary_db"},
		{"with single host", PostgresqlOpts{Hosts: []string{"127.0.0.1"}, Database: "canary_db", Table: "canary_table"}, "postgresql://127.0.0.1/canary_db"},
		{"with single host and port", PostgresqlOpts{Hosts: []string{"127.0.0.1:5432"}, Database: "canary_db", Table: "canary_table"}, "postgresql://127.0.0.1:5432/canary_db"},
		{"with single host and port in different configurations", PostgresqlOpts{Hosts: []string{"127.0.0.1"}, Port: 5432, Database: "canary_db", Table: "canary_table"}, "postgresql://127.0.0.1:5432/canary_db"},
		{"with multiple hosts", PostgresqlOpts{Hosts: []string{"192.168.0.1:5432", "192.168.0.2:5432"}, Database: "canary_db", Table: "canary_table"}, "postgresql://192.168.0.1:5432,192.168.0.2:5432/canary_db"},
		{"with multiple hosts and port", PostgresqlOpts{Hosts: []string{"192.168.0.1:5432", "192.168.0.2:5432"}, Database: "canary_db", Table: "canary_table"}, "postgresql://192.168.0.1:5432,192.168.0.2:5432/canary_db"},
		{"with multiple hosts and port in different configurations", PostgresqlOpts{Hosts: []string{"192.168.0.1", "192.168.0.2"}, Port: 5432, Database: "canary_db", Table: "canary_table"}, "postgresql://192.168.0.1:5432,192.168.0.2:5432/canary_db"},
		{"with ssl mode", PostgresqlOpts{Hosts: []string{"127.0.0.1"}, Database: "canary_db", Table: "canary_table", SSLMode: "require"}, "postgresql://127.0.0.1/canary_db?sslmode=require"},
		{"with username", PostgresqlOpts{Hosts: []string{"127.0.0.1"}, Database: "canary_db", Table: "canary_table", Username: "canary_user"}, "postgresql://canary_user@127.0.0.1/canary_db"},
		{"with username and password", PostgresqlOpts{Hosts: []string{"127.0.0.1"}, Database: "canary_db", Table: "canary_table", Username: "canary_user", Password: "canary_pwd"}, "postgresql://canary_user:canary_pwd@127.0.0.1/canary_db"},
	}

	for _, tc := range tests {
		t.Run(fmt.Sprintf(tc.name), func(t *testing.T) {
			m, err := NewPostgresql(tc.input)
			if err != nil {
				t.Errorf("could not create postgresql: %v", err)
			}

			uri, err := m.parseDSN()
			if err != nil {
				t.Errorf("could not generate postgresql dsn: %v", err)
			}

			if uri.String() != tc.expected {
				t.Errorf("got %s, expect %s", uri.String(), tc.expected)
			}

			t.Logf("got %s", uri.String())
		})
	}
}
