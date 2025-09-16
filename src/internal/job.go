package internal

import (
	"fmt"
	"log/slog"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/ovh/canary-ng/discover"
	"github.com/ovh/canary-ng/driver"
	"github.com/prometheus/client_golang/prometheus"
)

const (
	JOB_INTERVAL          = 1
	JOB_NAME_SEPARATOR    = "/"
	JOB_TYPE_CLICKHOUSE   = "clickhouse"
	JOB_TYPE_MONGODB      = "mongodb"
	JOB_TYPE_MYSQL        = "mysql"
	JOB_TYPE_POSTGRESQL   = "postgresql"
	JOB_TYPE_VALKEY       = "valkey"
	QUERY_TYPE_CONNECT    = "connect"
	QUERY_TYPE_READ       = "read"
	QUERY_TYPE_WRITE      = "write"
	QUERY_TYPE_READ_WRITE = "read_write"
	QUERY_TYPE_DISCONNECT = "disconnect"
	DISCOVER_TYPE_CONSUL  = "consul"
)

type Job struct {
	config      JobConfig
	metrics     *Metrics
	labels      prometheus.Labels
	queryLabels QueryLabelsConfig
	driver      driver.Driver
	discover    *discover.Discover
	logger      *slog.Logger
	start       time.Time
}

// Create multiple jobs
func NewJobs(config JobConfig, metrics *Metrics, queryLabels QueryLabelsConfig, jobLabelName string) (jobs []*Job, err error) {

	var hosts []string

	if config.Host != "" {
		hosts = []string{config.Host}

	} else if len(config.Hosts) > 0 {
		hosts = config.Hosts

	} else if config.HostsDiscovery.Type != "" {
		hosts, err = DiscoverHosts(config.HostsDiscovery)
		if err != nil {
			return nil, err
		}
		if len(hosts) == 0 {
			return nil, fmt.Errorf("0 host found by discovery")
		}
	} else if config.DSN == "" {
		return nil, fmt.Errorf("host, hosts, hosts_discovery or dsn required for job %s", config.Name)
	}

	if config.JobPerHost && len(hosts) > 0 {
		for _, h := range hosts {
			// Save original name because it could be changed by the prefix
			name := config.Name
			config.Hosts = []string{h}
			config.Name = AddHostPrefix(config)
			j, err := NewJob(config, metrics, queryLabels, jobLabelName)
			if err != nil {
				return nil, err
			}
			jobs = append(jobs, j)
			// Restore original name
			config.Name = name
		}
		return jobs, nil
	}

	config.Hosts = hosts
	config.Name = AddHostPrefix(config)

	j, err := NewJob(config, metrics, queryLabels, jobLabelName)
	if err != nil {
		return nil, err
	}

	return []*Job{j}, nil
}

// Add host prefix to the job name
func AddHostPrefix(config JobConfig) string {
	if !config.PrefixNameWithHost {
		return config.Name
	}
	separator := JOB_NAME_SEPARATOR
	return strings.Join([]string{strings.Join(config.Hosts, separator), config.Name}, separator)
}

// Create a single job
func NewJob(config JobConfig, metrics *Metrics, queryLabels QueryLabelsConfig, jobLabelName string) (j *Job, err error) {
	logger := slog.With("job", config.Name)

	l := prometheus.Labels{}

	for k, v := range config.Labels {
		l[k] = v
	}
	if jobLabelName == "" {
		return nil, fmt.Errorf("missing job label name")
	}
	l[jobLabelName] = config.Name

	if config.CacheHostnames {
		if config.Scheme == "mongodb+srv" {
			slog.Warn("skipping cache_hostnames for mongodb+srv scheme")
		} else {
			hosts := []string{}
			for _, host := range config.Hosts {
				if net.ParseIP(host) == nil {
					slog.Debug("resolving hostname", slog.Any("host", host))
					resolvedHosts, err := net.LookupIP(host)
					if err != nil {
						return nil, err
					}
					for _, resolvedHost := range resolvedHosts {
						slog.Debug("resolved host", slog.Any("host", host), slog.Any("resolved", resolvedHost))
						hosts = append(hosts, resolvedHost.String())
					}
				}
			}
			config.Hosts = hosts
		}
	}

	var d driver.Driver

	switch config.Type {
	case JOB_TYPE_CLICKHOUSE:
		d, err = driver.NewClickhouse(driver.ClickhousebOpts{
			DSN:        config.DSN,
			Hosts:      config.Hosts,
			Port:       config.Port,
			Username:   config.Username,
			Password:   config.Password,
			Timeout:    config.Timeout,
			Database:   config.Database,
			Table:      config.Table,
			Create:     config.Create,
			Cluster:    config.Cluster,
			Secure:     config.Secure,
			SkipVerify: config.SkipVerify,
			Logger:     logger,
		})
		if err != nil {
			return nil, err
		}
	case JOB_TYPE_MONGODB:
		d, err = driver.NewMongodb(driver.MongodbOpts{
			DSN:           config.DSN,
			Scheme:        config.Scheme,
			Hosts:         config.Hosts,
			Username:      config.Username,
			Password:      config.Password,
			AuthSource:    config.AuthSource,
			AuthMechanism: config.AuthMechanism,
			Port:          config.Port,
			TLS:           config.TLS,
			TLSInsecure:   config.TLSInsecure,
			Timeout:       config.Timeout,
			Database:      config.Database,
			Collection:    config.Collection,
			Create:        config.Create,
			Logger:        logger,
		})
		if err != nil {
			return nil, err
		}
	case JOB_TYPE_MYSQL:
		// The MySQL driver can handle only one host
		host := config.Host
		if len(config.Hosts) == 1 {
			host = config.Hosts[0]
		} else if len(config.Hosts) > 0 {
			return nil, fmt.Errorf("multiple hosts detected for mysql: %s", strings.Join(config.Hosts, ", "))
		}

		d, err = driver.NewMysql(driver.MysqlOpts{
			DSN:                  config.DSN,
			Host:                 host,
			Port:                 config.Port,
			Username:             config.Username,
			Password:             config.Password,
			Timeout:              config.Timeout,
			TLSConfig:            config.TLSConfig,
			Database:             config.Database,
			Table:                config.Table,
			Create:               config.Create,
			AllowNativePasswords: config.AllowNativePasswords,
			Logger:               logger,
		})
		if err != nil {
			return nil, err
		}
	case JOB_TYPE_POSTGRESQL:
		d, err = driver.NewPostgresql(driver.PostgresqlOpts{
			DSN:      config.DSN,
			Hosts:    config.Hosts,
			Port:     config.Port,
			Username: config.Username,
			Password: config.Password,
			SSLMode:  config.SSLMode,
			Timeout:  config.Timeout,
			Database: config.Database,
			Table:    config.Table,
			Create:   config.Create,
			Logger:   logger,
		})
		if err != nil {
			return nil, err
		}
	case JOB_TYPE_VALKEY:
		var db int
		if config.Database != "" {
			db, err = strconv.Atoi(config.Database)
			if err != nil {
				return nil, err
			}
		}
		d, err = driver.NewValkey(driver.ValkeyOpts{
			DSN:        config.DSN,
			Hosts:      config.Hosts,
			Port:       config.Port,
			MasterSet:  config.MasterSet,
			Username:   config.Username,
			Password:   config.Password,
			Timeout:    config.Timeout,
			Database:   db,
			Key:        config.Key,
			Create:     config.Create,
			TLS:        config.TLS,
			SkipVerify: config.SkipVerify,
			Logger:     logger,
		})
		if err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("unsupported job type %s", config.Type)
	}

	if config.Interval == 0 {
		config.Interval = JOB_INTERVAL
	}

	return &Job{
		config:      config,
		driver:      d,
		metrics:     metrics,
		labels:      l,
		queryLabels: queryLabels,
		logger:      logger,
	}, nil
}

func DiscoverHosts(config DiscoveryConfig) (hosts []string, err error) {
	var dh discover.Discover
	switch config.Type {
	case DISCOVER_TYPE_CONSUL:
		dh, err = discover.NewConsul(discover.ConsulOpts{
			Addresses:   config.Addresses,
			Datacenter:  config.Datacenter,
			Scheme:      config.Scheme,
			Token:       config.Token,
			SkipVerify:  config.SkipVerify,
			NodeMeta:    config.NodeMeta,
			ReturnMeta:  config.ReturnMeta,
			ReturnMetas: config.ReturnMetas,
		})
		if err != nil {
			return []string{}, err
		}
	default:
		return []string{}, fmt.Errorf("unsupported discovery type %s", config.Type)
	}

	hosts, err = dh.Discover()
	if err != nil {
		return []string{}, err
	}
	return hosts, nil
}

func (j *Job) Measure() {
	j.logger.Debug("starting to measure")

	j.StartMeasurement()
	err := j.driver.Connect()
	if err != nil {
		j.IncrFailures()
		j.logger.Warn("could not connect", slog.Any("error", err))
		return
	}
	j.EndMeasurement(QUERY_TYPE_CONNECT)

	switch j.config.QueryType {
	case QUERY_TYPE_READ:
		j.StartMeasurement()
		if err := j.driver.Read(); err != nil {
			j.IncrFailures()
			j.logger.Warn("could not read", slog.Any("error", err))
			return
		}
		j.EndMeasurement(QUERY_TYPE_READ)

	case QUERY_TYPE_WRITE:
		j.StartMeasurement()
		if err := j.driver.Write(); err != nil {
			j.IncrFailures()
			j.logger.Warn("could not write", slog.Any("error", err))
			return
		}
		j.EndMeasurement(QUERY_TYPE_WRITE)

	case QUERY_TYPE_READ_WRITE:
		j.StartMeasurement()
		if err := j.driver.Read(); err != nil {
			j.IncrFailures()
			j.logger.Warn("could not read", slog.Any("error", err))
			return
		}
		j.EndMeasurement(QUERY_TYPE_READ)

		j.StartMeasurement()
		if err := j.driver.Write(); err != nil {
			j.IncrFailures()
			j.logger.Warn("could not write", slog.Any("error", err))
			return
		}
		j.EndMeasurement(QUERY_TYPE_WRITE)

	default:
		j.IncrFailures()
		j.driver.Disconnect()
		return
	}

	j.StartMeasurement()
	err = j.driver.Disconnect()
	if err != nil {
		j.logger.Warn("could not disconnect", slog.Any("error", err))
		j.IncrFailures()
		return
	}
	j.EndMeasurement(QUERY_TYPE_DISCONNECT)
	j.IncrJobs()
}

func (j *Job) IncrFailures() {
	j.metrics.failures.With(j.labels).Add(1)
	j.IncrQueries()
	j.IncrJobs()
}

func (j *Job) IncrQueries() {
	j.metrics.queries.With(j.labels).Add(1)
}

func (j *Job) IncrJobs() {
	j.metrics.jobs.With(j.labels).Add(1)
}

func (j *Job) ObserveDuration(queryType string, duration float64) {
	labels := make(map[string]string)
	for k, v := range j.labels {
		labels[k] = v
	}

	switch queryType {
	case QUERY_TYPE_CONNECT:
		labels[j.queryLabels.Name] = j.queryLabels.ConnectValue
	case QUERY_TYPE_READ:
		labels[j.queryLabels.Name] = j.queryLabels.ReadValue
	case QUERY_TYPE_WRITE:
		labels[j.queryLabels.Name] = j.queryLabels.WriteValue
	case QUERY_TYPE_DISCONNECT:
		labels[j.queryLabels.Name] = j.queryLabels.DisconnectValue
	default:
		j.logger.Warn("invalid query type in observation, skipping", slog.Any("query_type", queryType))
		return
	}
	j.metrics.duration.With(labels).Observe(duration)
}

func (j *Job) StartMeasurement() {
	j.start = time.Now()
}

func (j *Job) EndMeasurement(queryType string) {
	end := time.Now()
	duration := end.Sub(j.start).Seconds()
	j.ObserveDuration(queryType, duration)
	j.IncrQueries()
}

func (j *Job) Run() {
	j.logger.Info("job started")
	for {
		j.Measure()
		j.logger.Info("measurement performed")

		w := "second"
		if j.config.Interval > 1 {
			w = "seconds"
		}
		j.logger.Debug(fmt.Sprintf("waiting for %d %s before next measurement", j.config.Interval, w))
		time.Sleep(time.Duration(j.config.Interval * int(time.Second)))
	}
}
