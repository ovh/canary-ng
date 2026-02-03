package driver

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	mysql "github.com/go-sql-driver/mysql"
)

const (
	MYSQL_DRIVER                       = "mysql"
	MYSQL_TABLE_NOT_FOUND_ERROR_PREFIX = "Error 1146 (42S02)"
)

type MysqlOpts struct {
	DSN                  string
	Host                 string
	Port                 int
	Username             string
	Password             string
	Database             string
	TLSConfig            string
	AllowNativePasswords bool
	Timeout              int
	Table                string
	Create               bool
	Logger               *slog.Logger
}

type Mysql struct {
	db     *sql.DB
	opts   MysqlOpts
	logger *slog.Logger
}

func NewMysql(opts MysqlOpts) (*Mysql, error) {
	if opts.Timeout == 0 {
		opts.Timeout = TIMEOUT
	}

	if opts.Table == "" {
		return nil, fmt.Errorf("table name is required")
	}

	if opts.DSN == "" {
		host := opts.Host
		if opts.Port > 0 && !strings.Contains(host, ":") {
			host = host + ":" + strconv.Itoa(opts.Port)
		}

		config := &mysql.Config{
			Net:                  "tcp", // TODO: detect if host is a socket
			User:                 opts.Username,
			Passwd:               opts.Password,
			Addr:                 host,
			AllowNativePasswords: opts.AllowNativePasswords,
			DBName:               opts.Database,
			TLSConfig:            opts.TLSConfig,
		}
		opts.DSN = config.FormatDSN()
	}

	var logger *slog.Logger
	if opts.Logger != nil {
		logger = opts.Logger.With("driver", MYSQL_DRIVER)
	} else {
		logger = slog.With("driver", MYSQL_DRIVER)
	}

	return &Mysql{
		opts:   opts,
		logger: logger,
	}, nil
}

func (m *Mysql) Connect() error {
	m.logger.Debug("openning connection")
	db, err := sql.Open("mysql", m.opts.DSN)
	if err != nil {
		return err
	}

	// https://github.com/go-sql-driver/mysql?tab=readme-ov-file#important-settings
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(1 * time.Minute)

	m.logger.Debug("ping")
	err = db.Ping()
	if err != nil {
		return err
	}
	m.db = db

	m.logger.Debug("connected")
	return nil
}

func (m *Mysql) Read() error {
	m.logger.Debug("reading")

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(m.opts.Timeout)*time.Second)
	defer cancel()

	var ts string
	err := m.db.QueryRowContext(ctx, fmt.Sprintf("SELECT ts FROM `%s` WHERE id = 1", m.opts.Table)).Scan(&ts)
	if err != nil {
		if strings.HasPrefix(err.Error(), MYSQL_TABLE_NOT_FOUND_ERROR_PREFIX) && m.opts.Create {
			return m.Write()
		}
		return err
	}

	m.logger.Debug("read", slog.Any("ts", ts))
	return nil
}

func (m *Mysql) Write() error {
	m.logger.Debug("writing")
	err := m.insert()
	if err != nil && strings.HasPrefix(err.Error(), MYSQL_TABLE_NOT_FOUND_ERROR_PREFIX) && m.opts.Create {
		if err = m.createTable(); err != nil {
			return err
		}
		return m.insert()
	}

	m.logger.Debug("written")
	return nil
}

func (m *Mysql) insert() error {
	m.logger.Debug("inserting")

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(m.opts.Timeout)*time.Second)
	defer cancel()

	_, err := m.db.ExecContext(ctx, fmt.Sprintf("REPLACE INTO `%s` (id, ts) VALUES (1, now())", m.opts.Table))
	if err != nil {
		return err
	}
	m.logger.Debug("inserted")
	return nil
}

func (m *Mysql) createTable() error {
	m.logger.Debug("creating table")

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(m.opts.Timeout)*time.Second)
	defer cancel()

	_, err := m.db.ExecContext(ctx, fmt.Sprintf("CREATE TABLE `%s` (id TINYINT PRIMARY KEY, ts TIMESTAMP NOT NULL)", m.opts.Table))
	if err != nil {
		return err
	}
	m.logger.Debug("created")
	return nil
}

func (m *Mysql) Disconnect() error {
	if m.db != nil {
		m.logger.Debug("disconnecting")
		if err := m.db.Close(); err != nil {
			return err
		}
		m.logger.Debug("disconnected")
	}
	return nil
}
