package driver

import (
	"context"
	"crypto/tls"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
)

const (
	ETCD_DRIVER = "etcd"
	ETCD_PORT   = 2379
)

type EtcdOpts struct {
	Hosts      []string
	Port       int
	Username   string
	Password   string
	Timeout    int
	Key        string
	Create     bool
	TLS        bool
	SkipVerify bool
	Logger     *slog.Logger
}

type Etcd struct {
	opts   EtcdOpts
	co     clientv3.Config
	client *clientv3.Client
	logger *slog.Logger
}

func NewEtcd(opts EtcdOpts) (e *Etcd, err error) {
	if opts.Timeout == 0 {
		opts.Timeout = TIMEOUT
	}

	if opts.Key == "" {
		return nil, fmt.Errorf("key is required")
	}

	co := clientv3.Config{
		Endpoints:   buildEtcdEndpoints(opts.Hosts, opts.Port),
		DialTimeout: time.Duration(opts.Timeout) * time.Second,
		Username:    opts.Username,
		Password:    opts.Password,
	}

	if opts.TLS || opts.SkipVerify {
		co.TLS = &tls.Config{}
		if opts.SkipVerify {
			co.TLS.InsecureSkipVerify = true
		}
	}

	var logger *slog.Logger
	if opts.Logger != nil {
		logger = opts.Logger.With("driver", ETCD_DRIVER)
	} else {
		logger = slog.With("driver", ETCD_DRIVER)
	}

	return &Etcd{
		opts:   opts,
		co:     co,
		logger: logger,
	}, nil
}

// etcd endpoints expect a port
func buildEtcdEndpoints(hosts []string, port int) []string {
	if port == 0 {
		port = ETCD_PORT
	}
	endpoints := make([]string, 0, len(hosts))
	for _, h := range hosts {
		if !strings.Contains(h, ":") {
			endpoints = append(endpoints, h+":"+strconv.Itoa(port))
		} else {
			endpoints = append(endpoints, h)
		}
	}
	return endpoints
}

func (e *Etcd) Connect() error {
	e.logger.Debug("connecting")

	client, err := clientv3.New(e.co)
	if err != nil {
		return err
	}
	e.client = client

	// clientv3.New dials lazily, so probe an endpoint to surface connection
	// errors here instead of on the first read or write
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(e.opts.Timeout)*time.Second)
	defer cancel()
	if _, err := client.Status(ctx, e.co.Endpoints[0]); err != nil {
		return err
	}

	e.logger.Debug("connected")
	return nil
}

func (e *Etcd) Read() error {
	e.logger.Debug("reading")

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(e.opts.Timeout)*time.Second)
	defer cancel()

	resp, err := e.client.Get(ctx, e.opts.Key)
	if err != nil {
		return err
	}

	if resp.Count == 0 {
		if e.opts.Create {
			return e.Write()
		}
		return fmt.Errorf("key does not exist")
	}

	e.logger.Debug("read", slog.Any("result", string(resp.Kvs[0].Value)))
	return nil
}

func (e *Etcd) Write() error {
	e.logger.Debug("writing")

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(e.opts.Timeout)*time.Second)
	defer cancel()

	ts := time.Now().Format(time.RFC3339)
	if _, err := e.client.Put(ctx, e.opts.Key, ts); err != nil {
		return err
	}
	e.logger.Debug("written", slog.Any("ts", ts))
	return nil
}

func (e *Etcd) Disconnect() error {
	if e.client != nil {
		e.logger.Debug("disconnecting")
		e.client.Close()
		e.logger.Debug("disconnected")
	}
	return nil
}
