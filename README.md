# tollgate

A layer-4 (TCP) load balancer written in Go: multiple balancing strategies, YAML config with CIDR-based client matching and cluster-based backend grouping, hot-reload without dropping connections, and a health/metrics side-channel for load-aware balancing.

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
go build -o bin/tollgate ./cmd/tollgate
./bin/tollgate -c path/to/config.yaml
```

Or with Docker (see [Running with Docker Compose](#running-with-docker-compose) below for a full multi-container example):

```sh
docker build -t tollgate .
docker run -v "path/to/config.yaml:/tollgate/config.yaml" -p 9080:9080 tollgate
```

## Configuration

By default the program looks for `config.yaml` in the working directory; override with `-c`/`-config`.

- `backends`: a set of `id` + `host:port` endpoints — `host` may be an IP or a hostname (Docker/Compose service names, `host.docker.internal`, etc.); hostnames are resolved on every dial, so DNS changes are picked up live
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

Needs two terminals: one for the dummy backends, one for tollgate itself, both left running.

Terminal 1:
```sh
go run ./examples/loadbalancertest/backends       # starts 4 dummy HTTP backends (srv-1..4) on :8081-:8084
```

Terminal 2:
```sh
go run ./cmd/tollgate -c examples/loadbalancertest/config.yaml
```

Terminal 3 (or any other):
```sh
curl localhost:9080   # round robin across main-api
curl localhost:9081   # least connections across main-api + priority-api
curl localhost:9082   # source IP hash
curl localhost:9083   # adaptive (no metrics pushed in this example, so it falls back to least-loaded)
```

Ctrl+C in terminals 1 and 2 when done.

Edit `examples/loadbalancertest/config.yaml` while the load balancer is running to see hot-reload pick up the change without dropping the process.

`examples/loadbalancertest/config-8servers.yaml` is the same setup scaled to 8 backends, on a separate port range so it can run alongside the 4-backend config: swap it in for either terminal 2 command above (`-c examples/loadbalancertest/config-8servers.yaml`), then hit `:9180`-`:9183` instead of `:9080`-`:9083`.

## Running with Docker Compose

`examples/dockercompose` runs tollgate and 4 backend containers on a shared Compose network, with backend addresses given as Compose service names (`srv-1:8081`, etc.) rather than IPs — this exercises hostname resolution end-to-end.

```sh
cd examples/dockercompose
docker compose up -d --build
sleep 1                # give the containers a moment to finish starting
curl localhost:9080    # round robin across srv-1..4, resolved via Docker's embedded DNS
docker compose down
cd ../..
```

## Running tests

```sh
go test ./...
go test -race ./...
```

## Benchmarks

Measured with the load-test tools in `bench/`, against the `examples/loadbalancertest` setup (round-robin rule, 4 backends), on an Apple M-series machine over loopback. Loopback numbers overstate real-network throughput/latency (no NIC, no real RTT) but are representative of proxy overhead and concurrency headroom, which is what these tools isolate.

Needs the backends and tollgate running first (see [Local testing](#local-testing-no-docker-needed) above, terminals 1 and 2), then in a third terminal:

```sh
go run ./bench/loadtest -target http://localhost:9080 -c 200 -d 10s
go run ./bench/connstress -addr localhost:9080 -n 10000 -hold 300ms
```

**Throughput/latency** (`bench/loadtest`, HTTP keep-alive, 10s runs):

| Concurrency | Throughput   | p50     | p90     | p99     |
|-------------|--------------|---------|---------|---------|
| 200         | 84,523 req/s | 2.2ms   | 3.5ms   | 6.7ms   |
| 1,000       | 51,575 req/s | 19.2ms  | 21.2ms  | 38.5ms  |

Zero request failures at either concurrency level.

**Concurrent connections** (`bench/connstress`, each connection opened simultaneously, held ~300ms, then closed):

| Attempted | Succeeded | Wall time |
|-----------|-----------|-----------|
| 5,000     | 5,000     | 1.0s      |
| 10,000    | 10,000    | 4.4s      |

`connstress` counts much above 10,000 will start failing on a single machine — that's the OS's ephemeral port pool running out (loopback testing double-counts ports, since client-to-tollgate and tollgate-to-backend connections share the same local pool), not a limit in tollgate itself. A real deployment, with clients, tollgate, and backends on separate machines, doesn't share a port pool this way.

`go test -race ./...` passes clean on the balancer test suite.
