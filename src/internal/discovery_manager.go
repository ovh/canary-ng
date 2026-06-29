package internal

import (
	"log/slog"
	"time"
)

// DiscoveryManager keeps the running jobs of a discovery-based job configuration
// in sync with the hosts currently returned by discovery. It re-discovers on an
// interval so hosts appearing or disappearing are picked up without restarting
// the application.
type DiscoveryManager struct {
	config       JobConfig
	metrics      *Metrics
	queryLabels  QueryLabelsConfig
	jobLabelName string
	interval     time.Duration
	logger       *slog.Logger
	running      map[string]chan struct{}
	discover     func() ([]string, error)
}

func NewDiscoveryManager(config JobConfig, metrics *Metrics, queryLabels QueryLabelsConfig, jobLabelName string) *DiscoveryManager {
	interval := config.HostsDiscovery.Interval
	if interval == 0 {
		interval = DISCOVERY_INTERVAL
	}

	return &DiscoveryManager{
		config:       config,
		metrics:      metrics,
		queryLabels:  queryLabels,
		jobLabelName: jobLabelName,
		interval:     time.Duration(interval) * time.Second,
		logger:       slog.With("job", config.Name),
		running:      map[string]chan struct{}{},
		discover:     func() ([]string, error) { return DiscoverHosts(config.HostsDiscovery) },
	}
}

func (s *DiscoveryManager) Run() {
	s.logger.Info("discovery manager started", slog.Duration("interval", s.interval))
	s.reconcile()

	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for range ticker.C {
		s.reconcile()
	}
}

// reconcile discovers the current hosts and aligns the running jobs with them.
// On a discovery failure the existing jobs are left running so a transient
// outage does not interrupt monitoring.
func (s *DiscoveryManager) reconcile() {
	hosts, err := s.discover()
	if err != nil {
		s.logger.Warn("could not discover hosts", slog.Any("error", err))
		return
	}
	if len(hosts) == 0 {
		s.logger.Warn("0 host found by discovery")
		return
	}

	desired, err := BuildJobs(s.config, hosts, s.metrics, s.queryLabels, s.jobLabelName)
	if err != nil {
		s.logger.Warn("could not build jobs", slog.Any("error", err))
		return
	}

	for key, stop := range s.running {
		if _, ok := desired[key]; !ok {
			s.logger.Info("stopping job for vanished host", slog.String("host", key))
			close(stop)
			delete(s.running, key)
		}
	}

	for key, job := range desired {
		if _, ok := s.running[key]; ok {
			continue
		}
		s.logger.Info("starting job for discovered host", slog.String("host", key))
		stop := make(chan struct{})
		s.running[key] = stop
		go job.Run(stop)
	}
}
