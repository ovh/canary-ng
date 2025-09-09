package driver

import (
	"fmt"
	"testing"
)

func TestMongodbURI(t *testing.T) {
	tests := []struct {
		name     string
		input    MongodbOpts
		expected string
	}{
		{"with uri", MongodbOpts{DSN: "mongodb://canary:password@127.0.0.1:27017/canary", Database: "canary", Collection: "canary"}, "mongodb://canary:password@127.0.0.1:27017/canary"},
		{"with single host", MongodbOpts{Hosts: []string{"127.0.0.1:27017"}, Port: 27017, Username: "canary", Password: "password", Database: "canary", Collection: "canary"}, "mongodb://127.0.0.1:27017/canary"},
		{"with multiple hosts", MongodbOpts{Hosts: []string{"192.168.0.1:27017", "192.168.0.2:27017"}, Username: "canary", Password: "password", Database: "canary", Collection: "canary"}, "mongodb://192.168.0.1:27017,192.168.0.2:27017/canary"},
		{"with scheme", MongodbOpts{Scheme: "mongodb+srv", Hosts: []string{"127.0.0.1"}, Username: "canary", Password: "password", Database: "canary", Collection: "canary"}, "mongodb+srv://127.0.0.1/canary"},
		{"with tls", MongodbOpts{Hosts: []string{"127.0.0.1"}, Database: "canary", Collection: "canary", TLS: true}, "mongodb://127.0.0.1/canary?tls=true"},
		{"with insecure tls", MongodbOpts{Hosts: []string{"127.0.0.1"}, Database: "canary", Collection: "canary", TLS: true, TLSInsecure: true}, "mongodb://127.0.0.1/canary?tls=true&tlsInsecure=true"},
	}

	for _, tc := range tests {
		t.Run(fmt.Sprintf(tc.name), func(t *testing.T) {
			m, err := NewMongodb(tc.input)
			if err != nil {
				t.Errorf("could not create mongodb: %v", err)
			}

			uri, err := m.parseURI()
			if err != nil {
				t.Errorf("could not generate mongodb uri: %v", err)
			}

			if uri.String() != tc.expected {
				t.Errorf("got %s, expect %s", uri.String(), tc.expected)
			}

			t.Logf("got %s", uri.String())
		})
	}
}

func TestMongodbTimeout(t *testing.T) {
	tests := []struct {
		name     string
		input    MongodbOpts
		expected int
	}{
		{"without timeout", MongodbOpts{DSN: "mongodb://127.0.0.1:27017/canary", Database: "canary", Collection: "canary"}, TIMEOUT},
		{"with timeout", MongodbOpts{Timeout: 3, DSN: "mongodb://127.0.0.1:27017/canary", Database: "canary", Collection: "canary"}, 3}}

	for _, tc := range tests {
		t.Run(fmt.Sprintf(tc.name), func(t *testing.T) {
			m, err := NewMongodb(tc.input)
			if err != nil {
				t.Errorf("could not create mongodb: %v", err)
			}

			if m.opts.Timeout != tc.expected {
				t.Errorf("got %d, expect %d", m.opts.Timeout, tc.expected)
			}

			t.Logf("got %d", m.opts.Timeout)
		})
	}
}
