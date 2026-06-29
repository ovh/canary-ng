//go:build e2e

package driver

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"testing"
	"time"
)

type consulNode struct {
	Node    string            `json:"Node"`
	Address string            `json:"Address"`
	Meta    map[string]string `json:"Meta"`
}

func consulNodesByDriver(t *testing.T, base, driver string) []consulNode {
	t.Helper()

	u := fmt.Sprintf("%s/v1/catalog/nodes?node-meta=%s", base, url.QueryEscape("driver:"+driver))
	resp, err := http.Get(u)
	if err != nil {
		t.Fatalf("consul nodes for driver %q: %v", driver, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read consul body for driver %q: %v", driver, err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("consul nodes for driver %q: status %d: %s", driver, resp.StatusCode, body)
	}

	var nodes []consulNode
	if err := json.Unmarshal(body, &nodes); err != nil {
		t.Fatalf("decode consul nodes for driver %q: %v", driver, err)
	}
	return nodes
}

// TestConsulDiscoveryE2E verifies host discovery through Consul end to end: each
// backend is registered in the Consul catalog under a "driver" node meta,
// canary-ng discovers its address from that meta, and Prometheus scrapes the
// resulting per-driver series.
func TestConsulDiscoveryE2E(t *testing.T) {
	consulBase := fmt.Sprintf("http://%s:%d", e2eHost("CONSUL", "127.0.0.1"), e2ePort("CONSUL", 8500))
	promBase := fmt.Sprintf("http://%s:%d", e2eHost("PROMETHEUS", "127.0.0.1"), e2ePort("PROMETHEUS", 9090))

	backends := map[string]string{
		"postgresql": "postgresql",
		"mysql":      "mysql",
		"mongodb":    "mongodb",
		"clickhouse": "clickhouse",
		"valkey":     "valkey",
	}

	for driver, address := range backends {
		nodes := consulNodesByDriver(t, consulBase, driver)
		if len(nodes) == 0 {
			t.Fatalf("no consul node registered with node meta driver=%s", driver)
		}
		found := false
		for _, n := range nodes {
			if n.Address == address {
				found = true
			}
		}
		if !found {
			t.Fatalf("consul node for driver %s missing address %q, got %+v", driver, address, nodes)
		}
	}

	metric := "canary_ng_jobs"
	jobs := []string{"postgresql", "mysql", "mongodb", "clickhouse", "valkey"}

	deadline := time.Now().Add(90 * time.Second)
	for {
		missing := waitForSeries(t, promBase, []string{metric}, jobs)
		if len(missing) == 0 {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("discovered series still empty after timeout: %v", missing)
		}
		time.Sleep(2 * time.Second)
	}
}
