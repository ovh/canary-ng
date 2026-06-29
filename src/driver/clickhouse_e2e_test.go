//go:build e2e

package driver

import "testing"

func TestClickhouseE2E(t *testing.T) {
	d, err := NewClickhouse(ClickhousebOpts{
		Hosts:    []string{e2eHost("CLICKHOUSE", "127.0.0.1")},
		Port:     e2ePort("CLICKHOUSE", 9000),
		Username: e2eEnv("CLICKHOUSE", "USERNAME", "canary"),
		Password: e2eEnv("CLICKHOUSE", "PASSWORD", "canary"),
		Database: e2eEnv("CLICKHOUSE", "DATABASE", "canary"),
		Table:    "canary_ng",
		Create:   true,
	})
	if err != nil {
		t.Fatalf("new clickhouse: %v", err)
	}

	runDriverE2E(t, d)
}
