package driver

import (
	"context"
	"crypto/tls"
	"fmt"
	"log/slog"
	"strconv"
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
	Port       int
	Database   string
	Username   string
	Password   string
	Timeout    int
	Secure     bool
	SkipVerify bool
	Logger     *slog.Logger
	Table      string
	Create     bool
	Cluster    string
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
		hosts := []string{}
		for _, h := range c.opts.Hosts {
			if !strings.Contains(h, ":") && c.opts.Port != 0 {
				hosts = append(hosts, h+":"+strconv.Itoa(c.opts.Port))
			} else {
				hosts = append(hosts, h)
			}
		}
		opts.Addr = hosts
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
	if c.opts.Cluster != "" {
		if err = c.createReplicatedTable(); err != nil {
			return err
		}
		if err = c.createDistributedTable(); err != nil {
			return err
		}
		return nil
	} else {
		return c.createLocalTable()
	}
}

func (c *Clickhouse) createLocalTable() (err error) {
	c.logger.Debug("creating local table")
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(c.opts.Timeout)*time.Second)
	defer cancel()
	query := fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (id int, ts DateTime64(3)) ENGINE ReplacingMergeTree ORDER BY id PRIMARY KEY id", c.opts.Table)
	return c.conn.Exec(ctx, query)
}

func (c *Clickhouse) createReplicatedTable() (err error) {
	c.logger.Debug("creating replicated table")
	if c.opts.Cluster == "" {
		return fmt.Errorf("cluster is not defined")
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(c.opts.Timeout)*time.Second)
	defer cancel()
	query := fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s_chunk ON CLUSTER '%s' (id int, ts DateTime64(3)) ENGINE ReplicatedReplacingMergeTree ORDER BY id PRIMARY KEY id", c.opts.Table, c.opts.Cluster)
	return c.conn.Exec(ctx, query)
}

func (c *Clickhouse) createDistributedTable() (err error) {
	c.logger.Debug("creating distributed table")
	if c.opts.Cluster == "" {
		return fmt.Errorf("cluster is not defined")
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(c.opts.Timeout)*time.Second)
	defer cancel()
	query := fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s ON CLUSTER '%s' (id int, ts DateTime64(3)) ENGINE Distributed('%s', '%s', %s_chunk, rand())", c.opts.Table, c.opts.Cluster, c.opts.Cluster, c.opts.Database, c.opts.Table)
	return c.conn.Exec(ctx, query)
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
