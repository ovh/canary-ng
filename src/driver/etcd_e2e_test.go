//go:build e2e

package driver

import "testing"

func TestEtcdE2E(t *testing.T) {
	d, err := NewEtcd(EtcdOpts{
		Hosts:    []string{e2eHost("ETCD", "127.0.0.1")},
		Port:     e2ePort("ETCD", 2379),
		Username: e2eEnv("ETCD", "USERNAME", ""),
		Password: e2eEnv("ETCD", "PASSWORD", ""),
		Key:      "canary_ng",
		Create:   true,
	})
	if err != nil {
		t.Fatalf("new etcd: %v", err)
	}

	runDriverE2E(t, d)
}
