package driver

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

const (
	POSTGRESQL_DRIVER                       = "postgresql"
	POSTGRESQL_TABLE_NOT_FOUND_ERROR_SUFFIX = "(SQLSTATE 42P01)"
	POSTGRESQL_NO_ROWS_ERROR                = "no rows in result set"
)

type PostgresqlOpts struct {
	DSN      string
	Hosts    []string
	Port     int
	Username string
	Password string
	Database string
	Timeout  int
	SSLMode  string
	Table    string
	Create   bool
	Logger   *slog.Logger
}

type Postgresql struct {
	conn   *pgx.Conn
	opts   PostgresqlOpts
	logger *slog.Logger
}

func NewPostgresql(opts PostgresqlOpts) (*Postgresql, error) {
	if opts.Timeout == 0 {
		opts.Timeout = TIMEOUT
	}

	if opts.Table == "" {
		return nil, fmt.Errorf("table name is required")
	}

	var logger *slog.Logger
	if opts.Logger != nil {
		logger = opts.Logger.With("driver", POSTGRESQL_DRIVER)
	} else {
		logger = slog.With("driver", POSTGRESQL_DRIVER)
	}

	p := &Postgresql{
		opts:   opts,
		logger: logger,
	}

	dsn, err := p.parseDSN()
	if err != nil {
		return nil, err
	}
	p.opts.DSN = dsn.String()

	return p, nil
}

// Parse connection strings according to the libpq format
// See https://www.postgresql.org/docs/current/libpq-connect.html#LIBPQ-CONNSTRING-URIS
func (p *Postgresql) parseDSN() (*url.URL, error) {
	if p.opts.DSN != "" {
		return url.Parse(p.opts.DSN)
	}

	uri := "postgresql://"

	// userspec
	if p.opts.Username != "" {
		uri += p.opts.Username
		if p.opts.Password != "" {
			uri += fmt.Sprintf(":%s", p.opts.Password)
		}
		uri += "@"
	}

	// hostspec
	if len(p.opts.Hosts) > 0 {
		hosts := p.opts.Hosts
		if p.opts.Port > 0 {
			hosts = []string{}
			for _, h := range p.opts.Hosts {
				if !strings.Contains(h, ":") {
					hosts = append(hosts, h+":"+strconv.Itoa(p.opts.Port))
				} else {
					hosts = append(hosts, h)
				}
			}
		}
		uri += strings.Join(hosts, ",")
	}

	// dbname
	if p.opts.Database != "" {
		uri += fmt.Sprintf("/%s", p.opts.Database)
	}

	url, err := url.Parse(uri)
	if err != nil {
		return nil, err
	}

	// paramspec
	paramspec := url.Query()
	if p.opts.SSLMode != "" {
		paramspec.Add("sslmode", p.opts.SSLMode)
		url.RawQuery = paramspec.Encode()
	}

	return url, nil
}

func (p *Postgresql) Connect() error {
	p.logger.Debug("connecting")

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(p.opts.Timeout)*time.Second)
	defer cancel()

	conn, err := pgx.Connect(ctx, p.opts.DSN)
	if err != nil {
		return err
	}
	p.conn = conn

	p.logger.Debug("connected")
	return nil
}

func (p *Postgresql) Read() error {
	p.logger.Debug("reading")

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(p.opts.Timeout)*time.Second)
	defer cancel()

	var ts string
	err := p.conn.QueryRow(ctx, fmt.Sprintf("SELECT to_char(ts, 'YYYY-MM-DD HH24:MI:SSOF') FROM %s WHERE id = 1", p.opts.Table)).Scan(&ts)
	if err != nil {
		if strings.HasSuffix(err.Error(), POSTGRESQL_TABLE_NOT_FOUND_ERROR_SUFFIX) && p.opts.Create {
			return p.Write()
		}
		return err
	}

	p.logger.Debug("read", slog.Any("ts", ts))
	return nil
}

func (p *Postgresql) Write() error {
	p.logger.Debug("writing")
	err := p.insert()
	if err != nil && strings.HasSuffix(err.Error(), POSTGRESQL_TABLE_NOT_FOUND_ERROR_SUFFIX) && p.opts.Create {
		if err = p.createTable(); err != nil {
			return err
		}
		return p.insert()
	}

	p.logger.Debug("written")
	return nil
}

func (p *Postgresql) insert() error {
	p.logger.Debug("inserting")

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(p.opts.Timeout)*time.Second)
	defer cancel()

	_, err := p.conn.Exec(ctx, fmt.Sprintf("INSERT INTO %s (id, ts) VALUES (1, now()) ON CONFLICT (id) DO UPDATE SET ts = now()", p.opts.Table))
	if err != nil {
		return err
	}
	p.logger.Debug("inserted")
	return nil
}

func (p *Postgresql) createTable() error {
	p.logger.Debug("creating table")

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(p.opts.Timeout)*time.Second)
	defer cancel()

	_, err := p.conn.Exec(ctx, fmt.Sprintf("CREATE TABLE %s (id smallint primary key, ts timestamp with time zone)", p.opts.Table))
	if err != nil {
		return err
	}
	p.logger.Debug("created")
	return nil
}

func (p *Postgresql) Disconnect() error {
	if p.conn != nil {
		p.logger.Debug("disconnecting")

		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(p.opts.Timeout)*time.Second)
		defer cancel()

		err := p.conn.Close(ctx)
		if err != nil {
			return err
		}
		p.logger.Debug("disconnected")
	}
	return nil
}
