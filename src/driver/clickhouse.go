package driver

import (
	"context"
	"crypto/tls"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

const (
	CLICKHOUSE_DRIVER                       = "clickhouse"
	CLICKHOUSE_TABLE_NOT_FOUND_ERROR_PREFIX = "code: 60,"
)

type ClickhousebOpts struct {
	DSN        string
	Hosts      []string
	Database   string
	Username   string
	Password   string
	Timeout    int
	Secure     bool
	SkipVerify bool
	Logger     *slog.Logger
	Table      string
	Create     bool
	Replicated bool
}

type Clickhouse struct {
	opts   ClickhousebOpts
	conn   driver.Conn
	logger *slog.Logger
}

func NewClickhouse(opts ClickhousebOpts) (c *Clickhouse, err error) {
	if opts.Timeout == 0 {
		opts.Timeout = TIMEOUT
	}

	if opts.Table == "" {
		return nil, fmt.Errorf("table name is required")
	}

	var logger *slog.Logger
	if opts.Logger != nil {
		logger = opts.Logger.With("driver", CLICKHOUSE_DRIVER)
	} else {
		logger = slog.With("driver", CLICKHOUSE_DRIVER)
	}

	c = &Clickhouse{
		opts:   opts,
		logger: logger,
	}

	return c, nil
}

func (c *Clickhouse) Connect() (err error) {
	opts := &clickhouse.Options{
		Auth: clickhouse.Auth{
			Database: "default",
			Username: "default",
			Password: "",
		},
	}

	if len(c.opts.Hosts) > 0 {
		opts.Addr = c.opts.Hosts
	}
	if c.opts.Database != "" {
		opts.Auth.Database = c.opts.Database
	}
	if c.opts.Username != "" {
		opts.Auth.Username = c.opts.Username
	}
	if c.opts.Password != "" {
		opts.Auth.Password = c.opts.Password
	}

	if c.opts.Secure {
		tlsConfig := &tls.Config{}
		if c.opts.SkipVerify {
			tlsConfig.InsecureSkipVerify = true
		}
		opts.TLS = tlsConfig
	}

	if c.opts.DSN != "" {
		c.logger.Debug("parsing dsn")
		opts, err = clickhouse.ParseDSN(c.opts.DSN)
		if err != nil {
			return err
		}
	}

	c.logger.Debug("openning connection")
	conn, err := clickhouse.Open(opts)
	if err != nil {
		return err
	}

	c.logger.Debug("pinging")
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(c.opts.Timeout)*time.Second)
	defer cancel()
	if err = conn.Ping(ctx); err != nil {
		return err
	}

	c.conn = conn
	c.logger.Debug("connected")
	return nil
}

func (c *Clickhouse) Read() (err error) {
	c.logger.Debug("reading")
	var ts string
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(c.opts.Timeout)*time.Second)
	defer cancel()
	if err = c.conn.QueryRow(ctx, fmt.Sprintf("SELECT formatDateTime(ts, '%%Y-%%m-%%d %%H:%%i:%%s%%z') FROM %s", c.opts.Table)).Scan(&ts); err != nil {
		if strings.HasPrefix(err.Error(), CLICKHOUSE_TABLE_NOT_FOUND_ERROR_PREFIX) && c.opts.Create {
			if err = c.Write(); err != nil {
				return err
			}
		}
		return err
	}

	c.logger.Debug("read", slog.Any("ts", ts))
	return nil
}

func (c *Clickhouse) Write() (err error) {
	c.logger.Debug("writing")
	err = c.insert()
	if err != nil && strings.HasPrefix(err.Error(), CLICKHOUSE_TABLE_NOT_FOUND_ERROR_PREFIX) && c.opts.Create {
		if err = c.createTable(); err != nil {
			return err
		}
		return c.insert()
	}

	c.logger.Debug("written")
	return nil
}

func (c *Clickhouse) insert() (err error) {
	c.logger.Debug("inserting")
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(c.opts.Timeout)*time.Second)
	defer cancel()
	return c.conn.Exec(ctx, fmt.Sprintf("INSERT INTO %s (id, ts) VALUES (1, now64())", c.opts.Table))
}

func (c *Clickhouse) createTable() (err error) {
	c.logger.Debug("creating table")
	engine := "ReplacingMergeTree"
	if c.opts.Replicated {
		engine = "ReplicatedReplacingMergeTree"
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(c.opts.Timeout)*time.Second)
	defer cancel()
	return c.conn.Exec(ctx, fmt.Sprintf("CREATE TABLE %s (id int, ts DateTime64(3)) ENGINE %s ORDER BY id PRIMARY KEY id", c.opts.Table, engine))
}

func (c *Clickhouse) Disconnect() (err error) {
	if c.conn != nil {
		c.logger.Debug("disconnecting")
		err := c.conn.Close()
		if err != nil {
			return err
		}
		c.logger.Debug("disconnected")
	}
	return nil
}
