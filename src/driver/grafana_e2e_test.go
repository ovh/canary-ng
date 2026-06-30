//go:build e2e

package driver

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"
)

const grafanaDashboardUID = "fe9k86kifrcowe"

// TestGrafanaE2E verifies the Canary NG dashboard is provisioned into Grafana
// and reachable through its API with the expected title.
func TestGrafanaE2E(t *testing.T) {
	base := fmt.Sprintf("http://%s:%d", e2eHost("GRAFANA", "127.0.0.1"), e2ePort("GRAFANA", 3000))
	user := e2eEnv("GRAFANA", "USER", "canary")
	password := e2eEnv("GRAFANA", "PASSWORD", "canary")

	deadline := time.Now().Add(90 * time.Second)
	for {
		title, err := grafanaDashboardTitle(base, user, password, grafanaDashboardUID)
		if err == nil {
			if title != "Canary NG" {
				t.Fatalf("unexpected dashboard title %q", title)
			}
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("dashboard %q not imported: %v", grafanaDashboardUID, err)
		}
		time.Sleep(2 * time.Second)
	}
}

func grafanaDashboardTitle(base, user, password, uid string) (string, error) {
	u := fmt.Sprintf("%s/api/dashboards/uid/%s", base, uid)
	req, err := http.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return "", err
	}
	req.SetBasicAuth(user, password)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("status %d: %s", resp.StatusCode, body)
	}

	var out struct {
		Dashboard struct {
			Title string `json:"title"`
			UID   string `json:"uid"`
		} `json:"dashboard"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return "", err
	}
	return out.Dashboard.Title, nil
}
