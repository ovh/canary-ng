package internal

import (
	"fmt"
	"io"
	"log/slog"
	"sort"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

func testMetrics() *Metrics {
	config := &Config{
		JobLabelName:   "job_name",
		DurationMetric: "canary_ng_duration",
		FailuresMetric: "canary_ng_failures",
		JobsMetric:     "canary_ng_jobs",
		QueriesMetric:  "canary_ng_queries",
		QueryLabels:    QueryLabelsConfig{Name: "query"},
	}
	return NewMetrics(prometheus.NewRegistry(), config)
}

func newTestSupervisor(discover func() ([]string, error)) *Supervisor {
	config := JobConfig{
		Name:           "discovery",
		Type:           JOB_TYPE_POSTGRESQL,
		QueryType:      QUERY_TYPE_READ,
		Database:       "canary_db",
		Table:          "canary_table",
		Interval:       3600,
		Timeout:        1,
		JobPerHost:     true,
		HostsDiscovery: DiscoveryConfig{Type: DISCOVER_TYPE_CONSUL},
	}
	return &Supervisor{
		config:       config,
		metrics:      testMetrics(),
		queryLabels:  QueryLabelsConfig{Name: "query"},
		jobLabelName: "job_name",
		interval:     time.Second,
		logger:       slog.With("job", config.Name),
		running:      map[string]chan struct{}{},
		discover:     discover,
	}
}

func runningKeys(s *Supervisor) []string {
	keys := make([]string, 0, len(s.running))
	for k := range s.running {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func isClosed(ch chan struct{}) bool {
	select {
	case <-ch:
		return true
	default:
		return false
	}
}

func TestSupervisorReconcile(t *testing.T) {
	slog.SetDefault(slog.New(slog.NewTextHandler(io.Discard, nil)))

	hosts := []string{"127.0.0.1", "127.0.0.2"}
	discoverErr := error(nil)
	discover := func() ([]string, error) { return hosts, discoverErr }

	s := newTestSupervisor(discover)

	t.Run("starts a job per discovered host", func(t *testing.T) {
		s.reconcile()
		got := runningKeys(s)
		want := []string{"127.0.0.1", "127.0.0.2"}
		if fmt.Sprint(got) != fmt.Sprint(want) {
			t.Errorf("got %v, expect %v", got, want)
		}
	})

	t.Run("keeps surviving hosts and reconciles the delta", func(t *testing.T) {
		survivor := s.running["127.0.0.2"]
		vanished := s.running["127.0.0.1"]

		hosts = []string{"127.0.0.2", "127.0.0.3"}
		s.reconcile()

		got := runningKeys(s)
		want := []string{"127.0.0.2", "127.0.0.3"}
		if fmt.Sprint(got) != fmt.Sprint(want) {
			t.Errorf("got %v, expect %v", got, want)
		}
		if s.running["127.0.0.2"] != survivor {
			t.Error("surviving host job was restarted instead of kept")
		}
		if !isClosed(vanished) {
			t.Error("vanished host job was not stopped")
		}
	})

	t.Run("keeps running jobs when discovery fails", func(t *testing.T) {
		before := runningKeys(s)

		discoverErr = fmt.Errorf("consul unreachable")
		s.reconcile()
		discoverErr = nil

		if got := runningKeys(s); fmt.Sprint(got) != fmt.Sprint(before) {
			t.Errorf("got %v, expect unchanged %v", got, before)
		}
	})

	t.Run("keeps running jobs when discovery returns no host", func(t *testing.T) {
		before := runningKeys(s)

		hosts = []string{}
		s.reconcile()
		hosts = []string{"127.0.0.2", "127.0.0.3"}

		if got := runningKeys(s); fmt.Sprint(got) != fmt.Sprint(before) {
			t.Errorf("got %v, expect unchanged %v", got, before)
		}
	})

	for _, ch := range s.running {
		close(ch)
	}
}
