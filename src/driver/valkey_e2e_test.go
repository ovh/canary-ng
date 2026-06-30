//go:build e2e

package driver

import "testing"

func TestValkeyE2E(t *testing.T) {
	d, err := NewValkey(ValkeyOpts{
		Hosts:    []string{e2eHost("VALKEY", "127.0.0.1")},
		Port:     e2ePort("VALKEY", 6379),
		Username: e2eEnv("VALKEY", "USERNAME", ""),
		Password: e2eEnv("VALKEY", "PASSWORD", ""),
		Key:      "canary_ng",
		Create:   true,
	})
	if err != nil {
		t.Fatalf("new valkey: %v", err)
	}

	runDriverE2E(t, d)
}
