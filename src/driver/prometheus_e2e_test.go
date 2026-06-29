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

type promQueryResponse struct {
	Status string `json:"status"`
	Data   struct {
		ResultType string `json:"resultType"`
		Result     []struct {
			Metric map[string]string `json:"metric"`
			Value  [2]any            `json:"value"`
		} `json:"result"`
	} `json:"data"`
}

func promQuery(t *testing.T, base, query string) promQueryResponse {
	t.Helper()

	u := fmt.Sprintf("%s/api/v1/query?query=%s", base, url.QueryEscape(query))
	resp, err := http.Get(u)
	if err != nil {
		t.Fatalf("query %q: %v", query, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body for %q: %v", query, err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("query %q: status %d: %s", query, resp.StatusCode, body)
	}

	var out promQueryResponse
	if err := json.Unmarshal(body, &out); err != nil {
		t.Fatalf("decode %q: %v", query, err)
	}
	if out.Status != "success" {
		t.Fatalf("query %q: status %s", query, out.Status)
	}
	return out
}

// TestPrometheusE2E verifies the full metrics pipeline: the canary-ng binary
// runs against every database, exposes /metrics, and Prometheus scrapes
// non-empty canary_ng_* series for each driver.
func TestPrometheusE2E(t *testing.T) {
	base := fmt.Sprintf("http://%s:%d", e2eHost("PROMETHEUS", "127.0.0.1"), e2ePort("PROMETHEUS", 9090))

	jobs := []string{"postgresql", "mysql", "mongodb", "clickhouse", "valkey"}
	metrics := []string{"canary_ng_jobs", "canary_ng_queries", "canary_ng_duration_count"}

	deadline := time.Now().Add(90 * time.Second)
	for {
		missing := waitForSeries(t, base, metrics, jobs)
		if len(missing) == 0 {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("canary_ng series still empty after timeout: %v", missing)
		}
		time.Sleep(2 * time.Second)
	}
}

func waitForSeries(t *testing.T, base string, metrics, jobs []string) []string {
	t.Helper()

	var missing []string
	for _, metric := range metrics {
		seen := map[string]bool{}
		for _, r := range promQuery(t, base, metric).Data.Result {
			seen[r.Metric["job_name"]] = true
		}
		for _, job := range jobs {
			if !seen[job] {
				missing = append(missing, fmt.Sprintf("%s{job_name=%q}", metric, job))
			}
		}
	}
	return missing
}
