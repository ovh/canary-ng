//go:build e2e

package driver

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"go.etcd.io/etcd/api/v3/v3rpc/rpctypes"
	clientv3 "go.etcd.io/etcd/client/v3"
)

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

const (
	etcdE2EUser     = "canary_ng"
	etcdE2EPassword = "canary"
	etcdE2EPrefix   = "/canary_ng/"
)

func etcdE2ECertsDir() string {
	return e2eEnv("ETCD_TLS", "CERTS_DIR", "../../docker/tls/etcd")
}

func etcdE2ETLSHost() string {
	return e2eHost("ETCD_TLS", "127.0.0.1")
}

func etcdE2ETLSPort() int {
	return e2ePort("ETCD_TLS", 2381)
}

// setupEtcdE2EAuth mirrors production, root authenticates with its
// client certificate CN and canary_ng only gets the /canary_ng/ prefix
func setupEtcdE2EAuth(t *testing.T) {
	t.Helper()
	dir := etcdE2ECertsDir()

	cert, err := tls.LoadX509KeyPair(filepath.Join(dir, "root.crt"), filepath.Join(dir, "root.key"))
	if err != nil {
		t.Fatalf("load root certificate: %v", err)
	}
	ca, err := os.ReadFile(filepath.Join(dir, "ca.crt"))
	if err != nil {
		t.Fatalf("read ca: %v", err)
	}
	pool := x509.NewCertPool()
	pool.AppendCertsFromPEM(ca)

	client, err := clientv3.New(clientv3.Config{
		Endpoints:   buildEtcdEndpoints([]string{etcdE2ETLSHost()}, etcdE2ETLSPort()),
		DialTimeout: 5 * time.Second,
		TLS:         &tls.Config{Certificates: []tls.Certificate{cert}, RootCAs: pool},
	})
	if err != nil {
		t.Fatalf("connect as root: %v", err)
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	ignoreExists := func(step string, err error) {
		t.Helper()
		if err != nil && !errors.Is(rpctypes.Error(err), rpctypes.ErrRoleAlreadyExist) && !errors.Is(rpctypes.Error(err), rpctypes.ErrUserAlreadyExist) {
			t.Fatalf("%s: %v", step, err)
		}
	}

	_, err = client.RoleAdd(ctx, "root")
	ignoreExists("add root role", err)
	_, err = client.UserAddWithOptions(ctx, "root", "", &clientv3.UserAddOptions{NoPassword: true})
	ignoreExists("add root user", err)
	_, err = client.UserGrantRole(ctx, "root", "root")
	ignoreExists("grant root role", err)

	_, err = client.RoleAdd(ctx, etcdE2EUser)
	ignoreExists("add canary role", err)
	_, err = client.RoleGrantPermission(ctx, etcdE2EUser, etcdE2EPrefix, clientv3.GetPrefixRangeEnd(etcdE2EPrefix), clientv3.PermissionType(clientv3.PermReadWrite))
	ignoreExists("grant canary permission", err)
	_, err = client.UserAdd(ctx, etcdE2EUser, etcdE2EPassword)
	ignoreExists("add canary user", err)
	_, err = client.UserGrantRole(ctx, etcdE2EUser, etcdE2EUser)
	ignoreExists("grant canary role", err)

	if _, err := client.AuthEnable(ctx); err != nil {
		t.Fatalf("enable auth: %v", err)
	}
}

func etcdE2ETLSOpts(password string, withCert bool) EtcdOpts {
	dir := etcdE2ECertsDir()
	opts := EtcdOpts{
		Hosts:    []string{etcdE2ETLSHost()},
		Port:     etcdE2ETLSPort(),
		Username: etcdE2EUser,
		Password: password,
		Key:      etcdE2EPrefix + "canary",
		Create:   true,
		Timeout:  5,
		CAFile:   filepath.Join(dir, "ca.crt"),
	}
	if withCert {
		opts.CertFile = filepath.Join(dir, "client.crt")
		opts.KeyFile = filepath.Join(dir, "client.key")
	}
	return opts
}

func TestEtcdE2EClientCertificate(t *testing.T) {
	setupEtcdE2EAuth(t)

	d, err := NewEtcd(etcdE2ETLSOpts(etcdE2EPassword, true))
	if err != nil {
		t.Fatalf("new etcd: %v", err)
	}

	runDriverE2E(t, d)
}

func TestEtcdE2EClientCertificateRejected(t *testing.T) {
	setupEtcdE2EAuth(t)

	tests := []struct {
		name string
		opts EtcdOpts
	}{
		{"without client certificate", etcdE2ETLSOpts(etcdE2EPassword, false)},
		{"with wrong password", etcdE2ETLSOpts("wrong", true)},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			d, err := NewEtcd(tc.opts)
			if err != nil {
				t.Fatalf("new etcd: %v", err)
			}
			defer d.Disconnect()

			if err := d.Connect(); err == nil {
				if err := d.Write(); err == nil {
					t.Error("expected the connection or the write to fail")
				}
			}
		})
	}
}
