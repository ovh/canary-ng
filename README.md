# Canary NG

Canary NG is a lightweight probe that continuously measures the latency of your
datastores and exposes the results as Prometheus histograms, so you can track
Service Level Objectives (SLO) and catch slowdowns before your users do.

It connects to a database, runs a small read and/or write query on a fixed
interval, times each step, and publishes the numbers on a `/metrics` endpoint.
Point Prometheus at it, drop in the provided Grafana dashboard, and you have
black-box latency monitoring for your data layer.

## How it works

For every job, Canary NG runs this cycle on the configured `interval`:

1. **connect** to the target
2. **read** and/or **write** a row/key (depending on `query_type`)
3. **disconnect**

Each step is timed and recorded under the `canary_ng_duration` histogram,
labelled by job name and query step (`connect`, `read`, `write`, `disconnect`).
Failures increment a counter instead of skewing the latency numbers.

## Quick start

### With Docker Compose

The repository ships a full demo stack (all supported databases, Consul,
Prometheus and Grafana) so you can see Canary NG working end to end.

The bundled databases use TLS, so generate a self-signed certificate first:

```bash
mkdir -p docker/tls
openssl req -x509 -newkey ec -pkeyopt ec_paramgen_curve:secp384r1 \
  -days 3650 -nodes -keyout docker/tls/db.key -out docker/tls/db.crt
```

Then bring the stack up:

```bash
docker compose up -d --build
```

And open:

- Canary NG metrics: <http://localhost:8080/metrics>
- Grafana dashboard: <http://localhost:3000>

### From source

```bash
make build
./bin/canary-ng -config canary-ng.yaml
```

Useful flags:

| Flag | Description |
|------|-------------|
| `-config <file>` | Path to the configuration file (default `canary-ng.yaml`) |
| `-verbose` | Log at `info` level |
| `-debug` | Log at `debug` level |
| `-quiet` | Log at `error` level only |
| `-version` | Print version and exit |

The binary always loads its configuration from the `-config` flag (default
`canary-ng.yaml`). The container image adds a convenience wrapper on top. Its
entrypoint reads the `CONFIG_PATH` environment variable (default
`/etc/canary-ng.yaml`) and passes it to `-config`. `CONFIG_PATH` has no effect
when running the binary directly.

## Configuration

Create a `canary-ng.yaml`. This measures PostgreSQL read latency every 4
seconds and creates the probe table if it does not exist:

```yaml
jobs:
  - name: postgresql_ro
    type: postgresql
    query_type: read
    dsn: "postgres://user:***@127.0.0.1:5432/canary?sslmode=require"
    table: canary_ng
    interval: 4
    create: true
```

Run it and scrape `http://localhost:8080/metrics`. A working
[`canary-ng.yaml.example`](canary-ng.yaml.example) with every supported driver
is included in the repository.

## Metrics

Canary NG exposes these Prometheus metrics (default names, all configurable):

| Metric | Type | Description |
|--------|------|-------------|
| `canary_ng_duration` | histogram | Latency of each step, labelled by job and query type |
| `canary_ng_failures` | counter | Number of failed executions |
| `canary_ng_jobs` | counter | Total job executions, including failures |
| `canary_ng_queries` | counter | Total query executions, including failures |

Example: 99th-percentile PostgreSQL read latency over 5 minutes:

```promql
histogram_quantile(0.99,
  sum by (le) (
    rate(canary_ng_duration_bucket{job_name="postgresql_ro", query="read"}[5m])
  )
)
```

## Grafana dashboard

A ready-to-use dashboard lives at
[`docker/grafana/dashboards/canary-ng.json`](docker/grafana/dashboards/canary-ng.json).
It plots latency histograms, failures, and job and query counters.

To import it into an existing Grafana:

1. Go to **Dashboards** > **New** > **Import**.
2. Upload `docker/grafana/dashboards/canary-ng.json` (or paste its content).
3. Select a Prometheus data source scraping Canary NG and click **Import**.

---

# Configuration reference

## Global

* `listen_addr` (string): host address to listen for the HTTP service (default `:8080`)
* `route` (string): name of the HTTP route to expose metrics (default `/metrics`)
* `jobs` (list): see Jobs below
* `job_label_name` (string): name of the Prometheus label registering the job name (default `job_name`)
* `buckets` ([]float64): list of thresholds in seconds to define Prometheus buckets
* `duration_metric` (string): name of the metric registering the duration histogram (default `canary_ng_duration`)
* `failures_metric` (string): name of the metric registering the failures counter (default `canary_ng_failures`)
* `jobs_metric` (string): name of the metric registering the job execution counter (default `canary_ng_jobs`)
* `queries_metric` (string): name of the metric registering the queries counter (default `canary_ng_queries`)
* `query_labels`:
    * `name` (string): name of the label registering the query name (default `query`)
    * `connect_value` (string): name of the connect query
    * `read_value` (string): name of the read query
    * `write_value` (string): name of the write query
    * `disconnect_value` (string): name of the disconnect query
* `log_level` (string): level of logging (`debug`, `info`, `warn` (default), `error`)
* `log_format` (string): format of log messages (`text` (default), `json`)

## Jobs

* `name` (string): name of the job
* `type` (string): name of the driver to use to perform queries (`clickhouse`, `etcd`, `mongodb`, `mysql`, `postgresql`, `valkey`)
* `query_type` (string): type of queries to measure (`read`, `write`, `read_write`)
* `hosts_discovery`: see "Host discovery" section
* `timeout` (int): number of second(s) before returning an error
* `interval` (int): number of second(s) to wait before next execution
* `job_per_host` (bool): create a job for each discovered host
* `prefix_name_with_host` (bool): add host to the job name (when using host discovery for example)
* `name_separator` (string): character to use to separate host and job name (used when `prefix_name_with_host` is enabled)
* `cache_hostnames` (bool): resolve hostnames at startup to exclude DNS resolution time from measurements (ignored for `mongodb+srv` scheme, disabled by default)

## Drivers

### ClickHouse

* `dsn` (string): connection string (ex: `clickhouse://***:***@127.0.0.1:9440/canary_clickhouse?secure=true&skip_verify=true`)
* `hosts` ([]string): list of hosts
* `port` (int): connect to this port
* `cluster` (string): name of the cluster in a replicated setup
* `username` (string): user name used for authentication
* `password` (string): password used for authentication
* `secure` (bool): use TLS for the connection
* `skip_verify` (bool): skip verification of the TLS certificate
* `database` (string): name of the database
* `table` (string): name of the table
* `create` (bool): create table if it doesn't exist (used by `read` queries)

### Etcd

* `hosts` ([]string): list of hosts
* `port` (int): connect to this port. If not defined, use ports from the `hosts` list or 2379.
* `username` (string): user name used for authentication
* `password` (string): password used for authentication
* `tls` (bool): use TLS for the connection
* `skip_verify` (bool): skip verification of the TLS certificate
* `key` (string): name of the key
* `create` (bool): write to key if it doesn't exist (used by `read` queries)

### MongoDB

 * `dsn` (string): connection string (ex: `mongodb://127.0.0.1:27017/canary_mongodb?tls=true&tlsInsecure=true`)
 * `scheme` (string): scheme for the connection (`mongodb`, `mongodb+srv`)
 * `hosts` ([]string): list of hosts
 * `port` (int): connect to this port
 * `username` (string): user name used for authentication
 * `password` (string): password used for authentication
 * `tls` (bool): use TLS for the connection
 * `tls_insecure` (bool): skip verification of the TLS certificate
 * `database` (string): name of the database
 * `collection` (string): name of collection
 * `create` (bool): create collection if it doesn't exist (used by `read` queries)

### MySQL

* `dsn` (string): connection string (ex: `***:***@tcp(127.0.0.1:3306)/canary_mysql?tls=skip-verify`)
* `host` (string): host address
* `port` (int): connect to this port
* `username` (string): user name used for authentication
* `password` (string): password used for authentication
* `tls_config` (string): use TLS for the connection (`false`, `true`, `skip-verify`, `preferred`)
* `allow_native_passwords` (bool): allow to connect using the mysql_native_password authentication plugin
* `database` (string): name of the database
* `table` (string): name of the table
* `create` (bool): create table if it doesn't exist (used by `read` queries)

### PostgreSQL

* `dsn` (string): connection string (ex: `postgres://***:***@127.0.0.1:5432/canary_postgresql?sslmode=require`)
* `hosts` ([]string): list of hosts
* `port` (int): connect to this port
* `username` (string): user name used for authentication
* `password` (string): password used for authentication
* `sslmode` (string): use TLS for the connection (see [available modes](https://www.postgresql.org/docs/current/libpq-ssl.html#LIBPQ-SSL-SSLMODE-STATEMENTS))
* `database` (string): name of the database
* `table` (string): name of the table
* `create` (bool): create table if it doesn't exist (used by `read` queries)

### Valkey

* `dsn` (ex: `rediss://127.0.0.1:6380/0`)
* `hosts` ([]string): list of hosts
* `port` (int): connect to this port. If not defined, use ports from the `hosts` list or 6379.
* `master_set` (string): enable Sentinel mode and connect to this master set name
* `username` (string): user name used for authentication
* `password` (string): password used for authentication
* `tls` (bool): use TLS for the connection
* `skip_verify` (bool): skip verification of the TLS certificate
* `database` (int): database number
* `key` (string): name of the key
* `create` (bool): write to key if it doesn't exist (used by `read` queries)

## Host discovery

Canary NG is able to discover a list of hosts instead of defining `host` or `hosts` in each job configuration.

Discovery runs periodically: hosts that appear are added and hosts that
disappear are removed without restarting the application. Use `interval` to
control how often discovery runs (defaults to 60 seconds). When discovery fails
or returns no host, the currently running jobs are kept so a transient outage
does not interrupt monitoring.

Example:

```yaml
jobs:
  host_discovery:
    type: consul
    interval: 60
    token: ***
    node_meta:
      dbms_type: postgresql
    return_meta: vip
```

### Consul

* `addresses` ([]string): list of Consul servers addresses
* `scheme` (string): URI scheme of Consul servers (`http`, `https`)
* `skip_verify` (bool): skip verification of the TLS certificate
* `datacenter` (string): name of the Consul datacenter
* `token` (string): token used for authentication
* `node_meta` (map[string][string]): return list of nodes matching these node meta
* `return_meta` (string): instead of returning the node IP address (by default), return the value of the node meta available on the node
* `return_metas` ([]string): same as `return_meta` but with multiple values (`return_meta` is ignored if `return_metas` is configured)

## Contributing

Contributions are welcome. See [CONTRIBUTING.md](CONTRIBUTING.md) for how to
build, test and submit changes.

## License

Licensed under the [Apache License, Version 2.0](LICENSE). Copyright OVH.
