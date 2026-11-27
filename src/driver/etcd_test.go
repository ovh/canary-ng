package driver

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
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

func writeEtcdTLSFiles(t *testing.T) (caFile, certFile, keyFile string) {
	t.Helper()
	dir := t.TempDir()

	caKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate ca key: %v", err)
	}
	caTemplate := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "ca"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		IsCA:                  true,
		KeyUsage:              x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
	}
	caDER, err := x509.CreateCertificate(rand.Reader, caTemplate, caTemplate, &caKey.PublicKey, caKey)
	if err != nil {
		t.Fatalf("create ca: %v", err)
	}

	clientKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate client key: %v", err)
	}
	clientTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject:      pkix.Name{CommonName: "canary_ng"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}
	clientDER, err := x509.CreateCertificate(rand.Reader, clientTemplate, caTemplate, &clientKey.PublicKey, caKey)
	if err != nil {
		t.Fatalf("create client certificate: %v", err)
	}
	clientKeyDER, err := x509.MarshalECPrivateKey(clientKey)
	if err != nil {
		t.Fatalf("marshal client key: %v", err)
	}

	write := func(name, blockType string, der []byte) string {
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, pem.EncodeToMemory(&pem.Block{Type: blockType, Bytes: der}), 0o600); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
		return path
	}

	return write("ca.crt", "CERTIFICATE", caDER),
		write("tls.crt", "CERTIFICATE", clientDER),
		write("tls.key", "EC PRIVATE KEY", clientKeyDER)
}

func TestEtcdTLSFiles(t *testing.T) {
	caFile, certFile, keyFile := writeEtcdTLSFiles(t)

	tests := []struct {
		name         string
		input        EtcdOpts
		rootCAs      bool
		certificates int
		skipVerify   bool
	}{
		{"with ca file", EtcdOpts{CAFile: caFile}, true, 0, false},
		{"with client certificate", EtcdOpts{CertFile: certFile, KeyFile: keyFile}, false, 1, false},
		{"with ca file and client certificate", EtcdOpts{CAFile: caFile, CertFile: certFile, KeyFile: keyFile}, true, 1, false},
		{"with client certificate and skip verify", EtcdOpts{CertFile: certFile, KeyFile: keyFile, SkipVerify: true}, false, 1, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tc.input.Hosts = []string{"127.0.0.1"}
			tc.input.Key = "canary_ng"

			e, err := NewEtcd(tc.input)
			if err != nil {
				t.Fatalf("could not create etcd: %v", err)
			}
			if e.co.TLS == nil {
				t.Fatal("expected TLS to be enabled")
			}
			if (e.co.TLS.RootCAs != nil) != tc.rootCAs {
				t.Errorf("got root CAs %t, expect %t", e.co.TLS.RootCAs != nil, tc.rootCAs)
			}
			if len(e.co.TLS.Certificates) != tc.certificates {
				t.Errorf("got %d certificates, expect %d", len(e.co.TLS.Certificates), tc.certificates)
			}
			if e.co.TLS.InsecureSkipVerify != tc.skipVerify {
				t.Errorf("got skip verify %t, expect %t", e.co.TLS.InsecureSkipVerify, tc.skipVerify)
			}
		})
	}
}

func TestEtcdTLSFilesErrors(t *testing.T) {
	caFile, certFile, keyFile := writeEtcdTLSFiles(t)
	missing := filepath.Join(t.TempDir(), "missing")

	tests := []struct {
		name  string
		input EtcdOpts
	}{
		{"with certificate without key", EtcdOpts{CertFile: certFile}},
		{"with key without certificate", EtcdOpts{KeyFile: keyFile}},
		{"with missing ca file", EtcdOpts{CAFile: missing}},
		{"with ca file without certificate", EtcdOpts{CAFile: keyFile}},
		{"with missing certificate", EtcdOpts{CertFile: missing, KeyFile: keyFile}},
		{"with mismatched certificate and key", EtcdOpts{CertFile: caFile, KeyFile: keyFile}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tc.input.Hosts = []string{"127.0.0.1"}
			tc.input.Key = "canary_ng"

			if _, err := NewEtcd(tc.input); err == nil {
				t.Error("expected error")
			}
		})
	}
}

func TestEtcdWithoutTLS(t *testing.T) {
	e, err := NewEtcd(EtcdOpts{Hosts: []string{"127.0.0.1"}, Key: "canary_ng"})
	if err != nil {
		t.Fatalf("could not create etcd: %v", err)
	}
	if e.co.TLS != nil {
		t.Error("expected TLS to be disabled")
	}
}
