package internal

import (
	"io"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

// slowDriver simulates a backend whose connect/query/disconnect cycle takes
// longer than the loop interval, and records the boundary of each cycle.
type slowDriver struct {
	work time.Duration

	mu     sync.Mutex
	starts []time.Time
	ends   []time.Time
}

func (d *slowDriver) Connect() error {
	d.mu.Lock()
	d.starts = append(d.starts, time.Now())
	d.mu.Unlock()
	time.Sleep(d.work)
	return nil
}

func (d *slowDriver) Read() error  { return nil }
func (d *slowDriver) Write() error { return nil }

func (d *slowDriver) Disconnect() error {
	d.mu.Lock()
	d.ends = append(d.ends, time.Now())
	d.mu.Unlock()
	return nil
}

// gaps returns the idle time between the end of each measurement and the start
// of the next one.
func (d *slowDriver) gaps() []time.Duration {
	d.mu.Lock()
	defer d.mu.Unlock()

	var gaps []time.Duration
	for i := 0; i+1 < len(d.starts) && i < len(d.ends); i++ {
		gaps = append(gaps, d.starts[i+1].Sub(d.ends[i]))
	}
	return gaps
}

func TestJobLoopKeepsIntervalCooldown(t *testing.T) {
	slog.SetDefault(slog.New(slog.NewTextHandler(io.Discard, nil)))

	const interval = 40 * time.Millisecond
	driver := &slowDriver{work: 60 * time.Millisecond}

	j := &Job{
		config:      JobConfig{Name: "test", QueryType: QUERY_TYPE_READ},
		driver:      driver,
		metrics:     testMetrics(),
		labels:      prometheus.Labels{"job_name": "test"},
		queryLabels: QueryLabelsConfig{Name: "query"},
		logger:      slog.With("job", "test"),
	}

	stop := make(chan struct{})
	done := make(chan struct{})
	go func() {
		j.loop(stop, interval)
		close(done)
	}()

	time.Sleep(600 * time.Millisecond)
	close(stop)
	<-done

	gaps := driver.gaps()
	if len(gaps) < 3 {
		t.Fatalf("expected at least 3 measurement gaps, got %d", len(gaps))
	}

	min := interval * 8 / 10
	for i, gap := range gaps {
		if gap < min {
			t.Errorf("gap %d = %v, want >= %v (interval cooldown lost, measurements ran back to back)", i, gap, min)
		}
	}
}
