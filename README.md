# Canary NG

Measure latency with various handlers and expose Prometheus histograms to
measure Service Level Objectives (SLO).

# Installation

## TLS

```
openssl req -x509 -newkey ec -pkeyopt ec_paramgen_curve:secp384r1 -days 3650 -nodes -keyout docker/tls/db.key -out docker/tls/db.crt
```

# Configuration

## Global

* `listen_addr` (string): host address to listen for the HTTP service
* `route` (string): name of the HTTP route to expose metrics
* `jobs` (list): see Jobs below
* `job_label_name` (string): name of the Prometheus label registering the job name
* `buckets` ([]float64): list of thresholds in seconds to define Prometheus buckets
* `duration_metric` (string): name of the metric registering the duration histogram
* `failures_metric` (string): name of the metric registering the failures counter
* `jobs_metric` (string): name of the metric registering the job execution counter
* `queries_metric` (string): name of the metric registering the queries counter
* `query_labels`:
    * `name` (string): name of the label registering the query name
    * `connect_value` (string): name of the connect query
    * `read_value` (string): name of the read query
    * `write_value` (string): name of the write query
    * `disconnect_value` (string): name of the disconnect query
* `log_level` (string): level of logging (`debug`, `info`, `warn` (default), `error`)
* `log_format` (string): format of log messages (`text` (default), `json`)

## Jobs

* `name` (string): name of the job
* `type` (string): name of the driver to use to perform queries (`clickhouse`, `mongodb`, `mysql`, `postgresql`, `valkey`)
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

Example:

```yaml
jobs:
  host_discovery:
    type: consul
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