package driver

import (
	"context"
	"crypto/tls"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/valkey-io/valkey-go"
)

const (
	VALKEY_DRIVER = "valkey"
	VALKEY_PORT   = 6379
)

type ValkeyOpts struct {
	DSN        string
	Hosts      []string
	Port       int
	MasterSet  string
	Username   string
	Password   string
	Database   int
	Timeout    int
	Key        string
	Create     bool
	TLS        bool
	SkipVerify bool
	Logger     *slog.Logger
}

type Valkey struct {
	opts   ValkeyOpts
	co     valkey.ClientOption
	client valkey.Client
	logger *slog.Logger
}

func NewValkey(opts ValkeyOpts) (v *Valkey, err error) {
	if opts.Timeout == 0 {
		opts.Timeout = TIMEOUT
	}

	if opts.Key == "" {
		return nil, fmt.Errorf("key is required")
	}

	// Valkey hosts expect a port
	var hosts []string
	port := VALKEY_PORT
	if opts.Port != 0 {
		port = opts.Port
	}
	for _, h := range opts.Hosts {
		if !strings.Contains(h, ":") {
			hosts = append(hosts, h+":"+strconv.Itoa(port))
		} else {
			hosts = append(hosts, h)
		}
	}

	co := valkey.ClientOption{
		InitAddress: hosts,
		Username:    opts.Username,
		Password:    opts.Password,
		SelectDB:    opts.Database,
	}

	if opts.MasterSet != "" {
		co.Sentinel = valkey.SentinelOption{
			MasterSet: opts.MasterSet,
		}
	}

	if opts.TLS || opts.SkipVerify {
		co.TLSConfig = &tls.Config{}
		if opts.SkipVerify {
			co.TLSConfig.InsecureSkipVerify = true
		}
	}

	if opts.DSN != "" {
		co, err = valkey.ParseURL(opts.DSN)
		if err != nil {
			return nil, err
		}
	}

	var logger *slog.Logger
	if opts.Logger != nil {
		logger = opts.Logger.With("driver", VALKEY_DRIVER)
	} else {
		logger = slog.With("driver", VALKEY_DRIVER)
	}

	return &Valkey{
		opts:   opts,
		co:     co,
		logger: logger,
	}, nil
}

func (v *Valkey) Connect() error {
	v.logger.Debug("connecting")

	client, err := valkey.NewClient(v.co)
	if err != nil {
		return err
	}
	v.client = client
	v.logger.Debug("connected")
	return nil
}

func (v *Valkey) Read() error {
	v.logger.Debug("reading")

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(v.opts.Timeout)*time.Second)
	defer cancel()

	r, err := v.client.Do(ctx, v.client.B().Get().Key(v.opts.Key).Build()).ToString()

	if err == valkey.Nil {
		if v.opts.Create {
			return v.Write()
		} else {
			return fmt.Errorf("key does not exist")
		}
	}

	if err != nil {
		return err
	}

	v.logger.Debug("read", slog.Any("result", r))
	return nil
}

func (v *Valkey) Write() error {
	v.logger.Debug("writing")

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(v.opts.Timeout)*time.Second)
	defer cancel()

	ts := time.Now().Format(time.RFC3339)
	err := v.client.Do(ctx, v.client.B().Set().Key(v.opts.Key).Value(ts).Build()).Error()
	if err != valkey.Nil {
		return err
	}
	v.logger.Debug("written", slog.Any("ts", ts))
	return nil
}

func (v *Valkey) Disconnect() error {
	if v.client != nil {
		v.logger.Debug("disconnecting")
		v.client.Close()
		v.logger.Debug("disconnected")
	}
	return nil
}
