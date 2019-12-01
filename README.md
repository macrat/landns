Landns
======

[![GitHub Actions](https://github.com/macrat/landns/workflows/Test%20and%20Build/badge.svg)](https://github.com/macrat/landns/actions)
[![codecov](https://codecov.io/gh/macrat/landns/branch/master/graph/badge.svg)](https://codecov.io/gh/macrat/landns)
[![Go Report Card](https://goreportcard.com/badge/github.com/macrat/landns)](https://goreportcard.com/report/github.com/macrat/landns)
![License](https://img.shields.io/github/license/macrat/landns)

A DNS server for developers for home use.

## How to use

### Install server

``` shell
$ go get github.com/macrat/landns
```

### Use as static DNS server

Make setting file like this.

``` yaml
ttl: 600

address:
  router.local: [192.168.1.1]
  servers.example.com:
    - 192.168.1.10
    - 192.168.1.11
    - 192.168.1.12

cname:
  gateway.local: [router.local]

text:
  message.local:
    - hello
    - world

services:
  example.com:
    - service: http
      port: 80
      proto: tcp    # optional (default: tcp)
      priority: 10  # optional (default: 0)
      weight: 5     # optional (default: 0)
      target: servers.example.com
    - service: ftp
      port: 21
      target: servers.example.com
```

And then, execute server.

``` shell
$ sudo landns --static-zone path/to/config.yml
```

### Use as dynamic DNS server

First, execute server.

``` shell
$ sudo landns --dynamic-zone path/to/database.db
```

Then, set records with API.

``` shell
$ curl http://localhost:9353/api/v1/record/address -H 'Content-Type: application/json' -d '{"router.local": [{"ttl": 600, "address": "192.168.1.1"}]}'

$ curl http://localhost:9353/api/v1/record/cname -H 'Content-Type: application/json' -d '{"gateway.local": [{"ttl": 600, "target": "router.local"}]}'

$ curl http://localhost:9353/api/v1/record/text -H 'Content-Type: application/json' -d '{"message.local": [{"ttl": 600, "text": "hello_world"}]}'

$ curl http://localhost:9353/api/v1/record/service -H 'Content-Type: application/json' -d '{"_web._tcp.example.com": [{"ttl": 600, "target": "servers.example.com", "port": 80, "priority": 10, "weight": 5}]}'
```

You can get records with API.

``` shell
$ curl http://localhost:9353/api/v1/record | jq
{
  "address": {
    "router.local.": [
      {
        "ttl": 600,
        "address": "192.168.1.1"
      }
    ]
  },
  "cname": {
    "gateway.local.": [
      {
        "ttl": 600,
        "target": "router.local."
      }
    ]
  },
  "text": {
    "message.local.": [
      {
        "ttl": 600,
        "text": "hello_world"
      }
    ]
  },
  "_web._tcp.example.com.": [
    {
      "ttl": 3600,
      "port": 80,
      "proirity": 10,
      "weight": 5,
      "target": "servers.example.com."
    }
  ]
}

$ curl http://localhost:9353/api/v1/record/address | jq
{
  "router.local.": [
    {
      "ttl": 600,
      "address": "192.168.1.1"
    }
  ]
}

$ curl http://localhost:9353/api/v1/record/address/router.local. | jq
[
  {
    "ttl": 600,
    "address": "192.168.1.1"
  }
]
```

#### CLI client

``` shell
$ go get github.com/macrat/landns/landnsctl

$ landnsctl addr example.com --set 192.168.1.1
$ landnsctl addr example.com
- address: 192.168.1.1
  ttl: 3600
```

### Get metrics (with prometheus)

Landns serve metrics for [Prometheus](https://prometheus.io) by default in port 9353.
