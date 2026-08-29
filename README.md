# enginewhy (Go)

A layer-4 (TCP) load balancer written in Go: multiple balancing strategies, YAML config with CIDR-based client matching and cluster-based backend grouping, hot-reload without dropping connections, and a health/metrics side-channel for load-aware balancing.

Design ported from a friend's Rust implementation ([nnhphong/enginewhy](https://github.com/nnhphong/enginewhy)), rewritten independently in Go. One deviation: that project's `LeastConnections` strategy was an empty stub — this one actually implements it.

## Features

- Multiple load balancing algorithms
  - Round Robin
  - Source IP Hash
  - Least Connections
  - Adaptive (weighted by backend CPU/mem/net/io, based on [this paper](https://www.wcse.org/WCSE_2018/W110.pdf))
- Cluster-based backend grouping and CIDR-based client matching, with longest-prefix-match rule resolution
- YAML config
- Hot-reload: editing the config file rebuilds routing without interrupting existing listeners
- A health-check side-channel: backends can push CPU/mem/net/io metrics as newline-delimited JSON, consumed by the Adaptive strategy

## Getting started

Requires Go 1.24+.

```sh
go build -o bin/enginewhy ./cmd/enginewhy
./bin/enginewhy -c path/to/config.yaml
```

Or with Docker:

```sh
docker build -t enginewhy .
docker run -v "path/to/config.yaml:/enginewhy/config.yaml" -p 8080:8080 enginewhy
```

## Configuration

By default the program looks for `config.yaml` in the working directory; override with `-c`/`-config`.

- `backends`: a set of `id` + `ip:port` endpoints
- `clusters`: named groups of backend ids
- `rules`: each rule matches one or more `client_cidr:port` patterns against a set of `targets` (backend ids or cluster names), balanced with a `strategy`

```yaml
healthcheck_addr: "0.0.0.0:9000"

backends:
  - id: "srv-1"
    ip: "127.0.0.1:8081"
  - id: "srv-2"
    ip: "127.0.0.1:8082"

clusters:
  main-api:
    - "srv-1"
    - "srv-2"

rules:
  - clients:
      - "0.0.0.0/0:8080"
    targets:
      - "main-api"
    strategy:
      type: "RoundRobin"

  - clients:
      - "10.0.0.0/24:8080"
    targets:
      - "main-api"
    strategy:
      type: "Adaptive"
      coefficients: [0.4, 0.3, 0.2, 0.1]
      alpha: 0.8
```

An incoming client is matched against whichever rule has the longest matching CIDR prefix on the connecting port.

## Local testing (no Docker needed)

```sh
go run ./examples/loadbalancertest/backends       # starts 4 dummy HTTP backends (srv-1..4) on :8081-:8084
go run ./cmd/enginewhy -c examples/loadbalancertest/config.yaml
curl localhost:9080   # round robin across main-api
curl localhost:9081   # least connections across main-api + priority-api
curl localhost:9082   # source IP hash
curl localhost:9083   # adaptive (no metrics pushed in this example, so it falls back to least-loaded)
```

Edit `examples/loadbalancertest/config.yaml` while the load balancer is running to see hot-reload pick up the change without dropping the process.

## Running tests

```sh
go test ./...
```
