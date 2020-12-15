module github.com/macrat/landns

go 1.15

require (
	github.com/alecthomas/kingpin v2.2.6+incompatible
	github.com/go-yaml/yaml v2.1.0+incompatible
	github.com/gomodule/redigo v1.8.3
	github.com/macrat/landns/client/go-client v0.0.0-00010101000000-000000000000
	github.com/mattn/go-sqlite3 v1.14.5
	github.com/miekg/dns v1.1.35
	github.com/prometheus/client_golang v1.8.0
	github.com/sirupsen/logrus v1.7.0
	go.etcd.io/etcd v0.0.0-20191023171146-3cf2f69b5738
)

replace github.com/macrat/landns/client/go-client => ./client/go-client