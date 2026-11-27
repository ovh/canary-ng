package internal

import (
	"path/filepath"
	"testing"
)

func TestNewJobEtcdPassesTLSFiles(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "missing")

	tests := []struct {
		name   string
		config JobConfig
	}{
		{"ca_file", JobConfig{CAFile: missing}},
		{"cert_file and key_file", JobConfig{CertFile: missing, KeyFile: missing}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tc.config.Name = "etcd"
			tc.config.Type = JOB_TYPE_ETCD
			tc.config.Hosts = []string{"127.0.0.1"}
			tc.config.Key = "canary_ng"

			if _, err := NewJob(tc.config, testMetrics(), QueryLabelsConfig{}, "job_name"); err == nil {
				t.Error("expected an error from the unreadable file")
			}
		})
	}
}
