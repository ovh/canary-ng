package internal

import (
	"fmt"
	"io"
	"log/slog"
	"slices"
	"testing"
)

func TestFilterResolvableHosts(t *testing.T) {
	slog.SetDefault(slog.New(slog.NewTextHandler(io.Discard, nil)))

	resolvable := map[string]bool{"db1.example.com": true, "db2.example.com": true}
	var lookups []string
	lookup := func(host string) ([]string, error) {
		lookups = append(lookups, host)
		if resolvable[host] {
			return []string{"192.0.2.1"}, nil
		}
		return nil, fmt.Errorf("lookup %s: no such host", host)
	}

	hosts := []string{"db1.example.com", "invalid..host", "", "192.0.2.10", "2001:db8::1", "unknown.example.com", "db2.example.com"}
	got := filterResolvableHosts(hosts, lookup)
	want := []string{"db1.example.com", "192.0.2.10", "2001:db8::1", "db2.example.com"}
	if !slices.Equal(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}

	if slices.Contains(lookups, "192.0.2.10") || slices.Contains(lookups, "2001:db8::1") {
		t.Fatalf("IP addresses must not be resolved, lookups %v", lookups)
	}
	if slices.Contains(lookups, "") {
		t.Fatalf("empty host must not be resolved, lookups %v", lookups)
	}
}

func TestFilterResolvableHostsAllInvalid(t *testing.T) {
	slog.SetDefault(slog.New(slog.NewTextHandler(io.Discard, nil)))

	lookup := func(host string) ([]string, error) { return nil, fmt.Errorf("no such host") }
	if got := filterResolvableHosts([]string{"a.invalid", "b.invalid"}, lookup); len(got) != 0 {
		t.Fatalf("got %v, want no host", got)
	}
}
