package driver

import (
	"strings"
	"testing"
)

func TestEtcdEndpoints(t *testing.T) {
	tests := []struct {
		name     string
		input    EtcdOpts
		expected string
	}{
		{"with single host", EtcdOpts{Hosts: []string{"127.0.0.1"}, Key: "canary_ng"}, "127.0.0.1:2379"},
		{"with single host and port", EtcdOpts{Hosts: []string{"127.0.0.1:2380"}, Key: "canary_ng"}, "127.0.0.1:2380"},
		{"with single host and port in different configurations", EtcdOpts{Hosts: []string{"127.0.0.1"}, Port: 2380, Key: "canary_ng"}, "127.0.0.1:2380"},
		{"with multiple hosts", EtcdOpts{Hosts: []string{"192.168.0.1", "192.168.0.2"}, Key: "canary_ng"}, "192.168.0.1:2379,192.168.0.2:2379"},
		{"with multiple hosts and port", EtcdOpts{Hosts: []string{"192.168.0.1", "192.168.0.2"}, Port: 2380, Key: "canary_ng"}, "192.168.0.1:2380,192.168.0.2:2380"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			e, err := NewEtcd(tc.input)
			if err != nil {
				t.Fatalf("could not create etcd: %v", err)
			}

			got := strings.Join(e.co.Endpoints, ",")
			if got != tc.expected {
				t.Errorf("got %s, expect %s", got, tc.expected)
			}

			t.Logf("got %s", got)
		})
	}
}

func TestEtcdTimeout(t *testing.T) {
	tests := []struct {
		name     string
		input    EtcdOpts
		expected int
	}{
		{"without timeout", EtcdOpts{Hosts: []string{"127.0.0.1"}, Key: "canary_ng"}, TIMEOUT},
		{"with timeout", EtcdOpts{Hosts: []string{"127.0.0.1"}, Key: "canary_ng", Timeout: 5}, 5},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			e, err := NewEtcd(tc.input)
			if err != nil {
				t.Fatalf("could not create etcd: %v", err)
			}
			if e.opts.Timeout != tc.expected {
				t.Errorf("got %d, expect %d", e.opts.Timeout, tc.expected)
			}
		})
	}
}

func TestEtcdKeyRequired(t *testing.T) {
	if _, err := NewEtcd(EtcdOpts{Hosts: []string{"127.0.0.1"}}); err == nil {
		t.Error("expected error when key is missing")
	}
}
