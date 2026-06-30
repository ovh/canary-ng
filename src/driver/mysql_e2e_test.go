//go:build e2e

package driver

import "testing"

func TestMysqlE2E(t *testing.T) {
	d, err := NewMysql(MysqlOpts{
		Host:                 e2eHost("MYSQL", "127.0.0.1"),
		Port:                 e2ePort("MYSQL", 3306),
		Username:             e2eEnv("MYSQL", "USERNAME", "canary"),
		Password:             e2eEnv("MYSQL", "PASSWORD", "canary"),
		Database:             e2eEnv("MYSQL", "DATABASE", "canary"),
		Table:                "canary_ng",
		Create:               true,
		AllowNativePasswords: true,
	})
	if err != nil {
		t.Fatalf("new mysql: %v", err)
	}

	runDriverE2E(t, d)
}
