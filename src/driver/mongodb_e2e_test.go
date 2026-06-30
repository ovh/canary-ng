//go:build e2e

package driver

import "testing"

func TestMongodbE2E(t *testing.T) {
	d, err := NewMongodb(MongodbOpts{
		Hosts:      []string{e2eHost("MONGODB", "127.0.0.1") + ":" + e2eEnv("MONGODB", "PORT", "27017")},
		Username:   e2eEnv("MONGODB", "USERNAME", "canary"),
		Password:   e2eEnv("MONGODB", "PASSWORD", "canary"),
		AuthSource: e2eEnv("MONGODB", "AUTHSOURCE", "admin"),
		Database:   e2eEnv("MONGODB", "DATABASE", "canary"),
		Collection: "canary_ng",
		Create:     true,
	})
	if err != nil {
		t.Fatalf("new mongodb: %v", err)
	}

	runDriverE2E(t, d)
}
