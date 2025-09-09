package discover

import (
	"fmt"
	"log/slog"

	"github.com/hashicorp/consul/api"
	"github.com/ovh/canary-ng/utils"
)

const (
	CONSUL_ADDRESS = "127.0.0.1:8500"
)

type ConsulOpts struct {
	NodeMeta    map[string]string
	ReturnMeta  string
	ReturnMetas []string
	Token       string
	Addresses   []string
	Scheme      string
	SkipVerify  bool
	Datacenter  string
}

type Consul struct {
	nodeMeta    map[string]string
	returnMetas []string
	clients     []*api.Client
}

func NewConsul(opts ConsulOpts) (*Consul, error) {
	// Used only at startup, the connection doesn't need to be in a pool to be reused
	config := api.DefaultNonPooledConfig()

	if len(opts.Addresses) == 0 {
		opts.Addresses = []string{CONSUL_ADDRESS}
	}
	if opts.Token != "" {
		config.Token = opts.Token
	}
	if opts.Datacenter != "" {
		config.Datacenter = opts.Datacenter
	}
	if opts.Scheme != "" {
		config.Scheme = opts.Scheme
	}
	if opts.Scheme == "https" {
		tlsConfig := *&api.TLSConfig{}
		if opts.SkipVerify {
			tlsConfig.InsecureSkipVerify = true
		}
		config.TLSConfig = tlsConfig
	}

	var clients []*api.Client
	for _, address := range opts.Addresses {
		config.Address = address
		client, err := api.NewClient(config)
		if err != nil {
			return nil, err
		}
		clients = append(clients, client)
	}

	var returnMetas []string
	if len(opts.ReturnMetas) > 0 {
		returnMetas = opts.ReturnMetas
	} else if opts.ReturnMeta != "" {
		returnMetas = []string{opts.ReturnMeta}
	}

	return &Consul{
		nodeMeta:    opts.NodeMeta,
		returnMetas: returnMetas,
		clients:     clients,
	}, nil
}

func (c *Consul) Discover() (hosts []string, err error) {
	var ok bool
	var nodes []*api.Node
	for _, client := range c.clients {
		nodes, _, err = client.Catalog().Nodes(&api.QueryOptions{NodeMeta: c.nodeMeta})
		if err == nil {
			ok = true
			break
		}
		slog.Warn("could not query consul catalog", slog.Any("error", err))
	}
	if !ok {
		return []string{}, fmt.Errorf("all consul clients failed")
	}

	if len(nodes) > 0 {
		slog.Debug("nodes discovered", slog.Any("nodes", nodes))
	}

	for _, node := range nodes {
		if len(c.returnMetas) > 0 {
			for _, returnMeta := range c.returnMetas {
				if meta, ok := node.Meta[returnMeta]; ok && !utils.In(hosts, meta) {
					hosts = append(hosts, meta)
				}
			}
		} else {
			hosts = append(hosts, node.Address)
		}
	}

	if len(hosts) > 0 {
		slog.Debug("hosts discovered", slog.Any("hosts", hosts))
	}
	return hosts, nil
}
