Landns
======

[![GitHub Actions](https://github.com/macrat/landns/workflows/Test%20and%20Build/badge.svg)](https://github.com/macrat/landns/actions)
[![codecov](https://codecov.io/gh/macrat/landns/branch/master/graph/badge.svg)](https://codecov.io/gh/macrat/landns)
[![Go Report Card](https://goreportcard.com/badge/github.com/macrat/landns)](https://goreportcard.com/report/github.com/macrat/landns)
[![License](https://img.shields.io/github/license/macrat/landns)](https://github.com/macrat/landns/blob/master/LICENSE)
[![Docker Cloud Automated build](https://img.shields.io/docker/cloud/automated/macrat/landns?logo=docker&logoColor=white)](https://hub.docker.com/repository/docker/macrat/landns)

A DNS server for developers for home use.

**BE CAREFUL**: this product is not stable yet. may change options and behavior.


## Features

- Serve addresses from the YAML style configuration file.

- Serve addresses from a database that operatable with RESTFUL API.

- Recursion resolve and caching addresses to local memory or [Redis server](https://redis.io).

- Built-in metrics exporter for [Prometheus](https://prometheus.io).


## How to use

### Install server

Please use `go get`.

``` shell
$ go get github.com/macrat/landns
```

Or, you can use docker.

``` shell
$ docker run -p 9353:9353/tcp -p 53:53/udp macrat/landns
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
$ sudo landns --config path/to/config.yml
```

### Use as dynamic DNS server

First, execute server.

``` shell
$ sudo landns --sqlite path/to/database.db
```

Dynamic settings that set by REST API will store to specified database if given `--sqlite` option.
REST API will work if not gven it, but settings will lose when the server stopped.

Then, operate records with API.

``` shell
$ curl http://localhost:9353/api/v1 -d 'www.example.com 600 IN A 192.168.1.1'
; 200: add:1 delete:0

$ curl http://localhost:9353/api/v1 -d 'ftp.example.com 600 IN CNAME www.example.com'
; 200: add:1 delete:0

$ curl http://localhost:9353/api/v1
www.example.com 600 IN A 192.168.1.1 ; ID:1
1.1.168.192.in-addr.arpa. 600 IN PTR www.example.com ; ID:2
ftp.example.com 600 IN CNAME www.example.com ; ID:3

$ curl http://localhost:9353/api/v1/suffix/com/example
www.example.com 600 IN A 192.168.1.1 ; ID:1
ftp.example.com 600 IN CNAME www.example.com ; ID:3

$ curl http://localhost:9353/api/v1/suffix/example.com
www.example.com 600 IN A 192.168.1.1 ; ID:1
ftp.example.com 600 IN CNAME www.example.com ; ID:3

$ curl http://localhost:9353/api/v1/glob/w*ample.com
www.example.com 600 IN A 192.168.1.1 ; ID:1
```

```
$ cat config.zone
router.service. 600 IN A 192.168.1.1
gateway.service. 600 IN CNAME router.local.
alice.pc.local. 600 IN A 192.168.1.10

$ curl http://localhost:9353/api/v1 --data-binary @config.zone
; 200: add:3 delete:0

$ curl http://localhost:9353/api/v1
router.service. 600 IN A 192.168.1.1 ; ID:1
1.1.168.192.in-addr.arpa. 600 IN PTR router.service. ; ID:2
gateway.service. 600 IN CNAME router.local. ; ID:3
alice.pc.local. 600 IN A 192.168.1.10 ; ID:4
10.1.168.192.in-addr.arpa. 600 IN PTR alice.pc.local. ; ID:5
```

There are 3 ways to remove records.

``` shell
$ curl http://localhost:9353/api/v1 -X DELETE -d 'router.service. 600 IN A 192.168.1.1 ; ID:1'  # Use DELETE method
; 200: add:0 delete:1

$ curl http://localhost:9353/api/v1
gateway.service. 600 IN CNAME router.local. ; ID:3
alice.pc.local. 600 IN A 192.168.1.10 ; ID:4
10.1.168.192.in-addr.arpa. 600 IN PTR alice.pc.local. ; ID:5

$ curl http://localhost:9353/api/v1 -X POST -d ';gateway.service. 600 IN CNAME router.local. ; ID:3'  # Use comment style
; 200: add:0 delete:1

$ curl http://localhost:9353/api/v1
alice.pc.local. 600 IN A 192.168.1.10 ; ID:4
10.1.168.192.in-addr.arpa. 600 IN PTR alice.pc.local. ; ID:5

$ curl http://localhost:9353/api/v1/id/4 -X DELETE  # Use DELETE method with ID
; 200: ok

$ curl http://localhost:9353/api/v1
10.1.168.192.in-addr.arpa. 600 IN PTR alice.pc.local. ; ID:5
```

### Get metrics (with prometheus)

Landns serve metrics for Prometheus by default in port 9353.
