package internal

import (
	"github.com/prometheus/client_golang/prometheus"
)

type Metrics struct {
	duration *prometheus.HistogramVec
	failures *prometheus.CounterVec
	jobs     *prometheus.CounterVec
	queries  *prometheus.CounterVec
}

func NewMetrics(reg prometheus.Registerer, config *Config) *Metrics {
	labels := []string{config.JobLabelName}
	m := &Metrics{
		duration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    config.DurationMetric,
			Help:    "Execution time of the job",
			Buckets: config.Buckets,
		}, append(labels, config.QueryLabels.Name)),
		failures: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: config.FailuresMetric,
			Help: "Number of execution that has failed",
		}, labels),
		jobs: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: config.JobsMetric,
			Help: "Total number of job executions including failures",
		}, labels),
		queries: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: config.QueriesMetric,
			Help: "Total number of queries executions including failures",
		}, labels),
	}
	reg.MustRegister(m.duration, m.failures, m.jobs, m.queries)
	return m
}
