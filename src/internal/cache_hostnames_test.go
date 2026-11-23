package internal

import (
	"io"
	"log/slog"
	"slices"
	"testing"
)

func TestNewJobCacheHostnamesKeepsIPs(t *testing.T) {
	slog.SetDefault(slog.New(slog.NewTextHandler(io.Discard, nil)))

	config := JobConfig{
		Name:           "cache",
		Type:           JOB_TYPE_POSTGRESQL,
		QueryType:      QUERY_TYPE_READ,
		Database:       "canary_db",
		Table:          "canary_table",
		Timeout:        1,
		CacheHostnames: true,
		Hosts:          []string{"192.0.2.10", "2001:db8::1", "localhost"},
	}

	j, err := NewJob(config, testMetrics(), QueryLabelsConfig{Name: "query"}, "job_name")
	if err != nil {
		t.Fatalf("NewJob: %v", err)
	}

	for _, ip := range []string{"192.0.2.10", "2001:db8::1"} {
		if !slices.Contains(j.config.Hosts, ip) {
			t.Fatalf("IP %s missing from cached hosts %v", ip, j.config.Hosts)
		}
	}
	if slices.Contains(j.config.Hosts, "localhost") {
		t.Fatalf("hostname not resolved in cached hosts %v", j.config.Hosts)
	}
	if len(j.config.Hosts) < 3 {
		t.Fatalf("resolved addresses of localhost missing from %v", j.config.Hosts)
	}
}
