package internal

import (
	"fmt"
	"os"

	"github.com/ovh/canary-ng/utils"
	"gopkg.in/yaml.v3"
)

type Config struct {
	ListenAddr     string            `yaml:"listen_addr"`
	Route          string            `yaml:"route"`
	Jobs           []JobConfig       `yaml:"jobs"`
	JobLabelName   string            `yaml:"job_label_name"`
	Buckets        []float64         `yaml:"buckets"`
	DurationMetric string            `yaml:"duration_metric"`
	FailuresMetric string            `yaml:"failures_metric"`
	JobsMetric     string            `yaml:"jobs_metric"`
	QueriesMetric  string            `yaml:"queries_metric"`
	QueryLabels    QueryLabelsConfig `yaml:"query_labels"`
	LogLevel       string            `yaml:"log_level"`
	LogFormat      string            `yaml:"log_format"`
}

type QueryLabelsConfig struct {
	Name            string `yaml:"name"`
	ConnectValue    string `yaml:"connect_value"`
	ReadValue       string `yaml:"read_value"`
	WriteValue      string `yaml:"write_value"`
	DisconnectValue string `yaml:"disconnect_value"`
}

type JobConfig struct {
	Name                 string            `yaml:"name"`
	Labels               map[string]string `yaml:"labels"`
	Type                 string            `yaml:"type"`
	Interval             int               `yaml:"interval"`
	DSN                  string            `yaml:"dsn"`
	Scheme               string            `yaml:"scheme"`
	Username             string            `yaml:"username"`
	Password             string            `yaml:"password"`
	Host                 string            `yaml:"host"`
	Hosts                []string          `yaml:"hosts"`
	CacheHostnames       bool              `yaml:"cache_hostnames"`
	HostsDiscovery       DiscoveryConfig   `yaml:"hosts_discovery"`
	JobPerHost           bool              `yaml:"job_per_host"`
	PrefixNameWithHost   bool              `yaml:"prefix_name_with_host"`
	NameSeparator        string            `yaml:"name_separator"` // used when prefix_name_with_host is defined
	Port                 int               `yaml:"port"`
	QueryType            string            `yaml:"query_type"`
	Timeout              int               `yaml:"timeout"`
	Database             string            `yaml:"database"`
	AuthSource           string            `yaml:"auth_source"`
	AuthMechanism        string            `yaml:"auth_mechanism"`
	Collection           string            `yaml:"collection"`
	Table                string            `yaml:"table"`
	Replicated           bool              `yaml:"replicated"`
	Key                  string            `yaml:"key"`
	Create               bool              `yaml:"create"`
	Secure               bool              `yaml:"secure"`
	SkipVerify           bool              `yaml:"skip_verify"`
	SSLMode              string            `yaml:"sslmode"`
	TLS                  bool              `yaml:"tls"`
	TLSInsecure          bool              `yaml:"tls_insecure"`
	TLSConfig            string            `yaml:"tls_config"`
	AllowNativePasswords bool              `yaml:"allow_native_passwords"`
	MasterSet            string            `yaml:"master_set"`
}

type DiscoveryConfig struct {
	Type        string            `yaml:"type"`
	Addresses   []string          `yaml:"addresses"`
	Datacenter  string            `yaml:"datacenter"`
	Scheme      string            `yaml:"scheme"`
	SkipVerify  bool              `yaml:"skip_verify"`
	Token       string            `yaml:"token"`
	NodeMeta    map[string]string `yaml:"node_meta"`
	ReturnMeta  string            `yaml:"return_meta"`
	ReturnMetas []string          `yaml:"return_metas"`
}

func NewConfig(file string) (config *Config, err error) {

	// Default configuration
	config = &Config{
		ListenAddr:     ":8080",
		Route:          "/metrics",
		LogLevel:       "warn",
		LogFormat:      "text",
		JobLabelName:   "job_name",
		DurationMetric: "canary_ng_duration",
		FailuresMetric: "canary_ng_failures",
		JobsMetric:     "canary_ng_jobs",
		QueriesMetric:  "canary_ng_queries",
		QueryLabels: QueryLabelsConfig{
			Name:            "query",
			ConnectValue:    QUERY_TYPE_CONNECT,
			ReadValue:       QUERY_TYPE_READ,
			WriteValue:      QUERY_TYPE_WRITE,
			DisconnectValue: QUERY_TYPE_DISCONNECT,
		},
	}

	buf, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}
	err = yaml.Unmarshal(buf, &config)
	if err != nil {
		return nil, err
	}

	// Ensure query types are valid
	for _, job := range config.Jobs {
		if !utils.In([]string{QUERY_TYPE_READ, QUERY_TYPE_WRITE, QUERY_TYPE_READ_WRITE}, job.QueryType) {
			return nil, fmt.Errorf("invalid query type %s for job %s", job.QueryType, job.Name)
		}
	}

	// Ensure jobs have a name
	for _, job := range config.Jobs {
		if job.Name == "" {
			return nil, fmt.Errorf("job without name")
		}
	}

	return config, nil
}
