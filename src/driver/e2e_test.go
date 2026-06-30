//go:build e2e

package driver

import (
	"os"
	"strconv"
	"testing"
)

func e2eEnv(driver, key, def string) string {
	if v := os.Getenv("CANARY_E2E_" + driver + "_" + key); v != "" {
		return v
	}
	return def
}

func e2eHost(driver, def string) string {
	return e2eEnv(driver, "HOST", def)
}

func e2ePort(driver string, def int) int {
	p, err := strconv.Atoi(e2eEnv(driver, "PORT", strconv.Itoa(def)))
	if err != nil {
		return def
	}
	return p
}

// runDriverE2E exercises the full canary measurement against a live database,
// matching the order internal.Job.Measure uses. Write runs before Read so the
// canary row exists, since several drivers return the original error on a
// missing row even when create is enabled.
func runDriverE2E(t *testing.T, d Driver) {
	t.Helper()

	if err := d.Connect(); err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer func() {
		if err := d.Disconnect(); err != nil {
			t.Errorf("disconnect: %v", err)
		}
	}()

	if err := d.Write(); err != nil {
		t.Fatalf("write: %v", err)
	}

	if err := d.Read(); err != nil {
		t.Fatalf("read: %v", err)
	}
}
