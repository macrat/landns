package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"github.com/alecthomas/kingpin"
	"github.com/miekg/dns"

	"github.com/macrat/landns/lib-landns"
)

var (
	configFiles      = kingpin.Flag("static-zone", "Path to static-zone configuration file.").Short('s').PlaceHolder("PATH").ExistingFiles()
	sqlitePath       = kingpin.Flag("dynamic-zone", "Path to dynamic-zone database path.").Short('d').PlaceHolder("PATH").String()
	apiListen        = kingpin.Flag("api-listen", "Address for API and metrics.").Short('l').Default(":9353").TCP()
	dnsListen        = kingpin.Flag("dns-listen", "Address for listen.").Default(":53").TCP()
	dnsProtocol      = kingpin.Flag("dns-protocol", "Protocol for listen.").Default("udp").Enum("udp", "tcp")
	metricsNamespace = kingpin.Flag("metrics-namespace", "Namespace of prometheus metrics.").Default("landns").String()
)

func main() {
	kingpin.Parse()

	metrics := landns.NewMetrics(*metricsNamespace)
	if err := metrics.Register(); err != nil {
		log.Fatalf("metrics: %s", err.Error())
	}
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "<h1>Landns</h1><a href=\"/metrics\">metrics</a>")
	})
	http.Handle("/metrics", metrics.HTTPHandler())

	resolvers := landns.ResolverSet{}

	if *sqlitePath != "" {
		sql, err := landns.NewSqliteResolver(*sqlitePath, metrics)
		if err != nil {
			log.Fatalf("dynamic-zone: %s", err)
		}
		http.Handle("/api/", http.StripPrefix("/api", landns.NewDynamicAPI(sql).Handler()))
		resolvers = append(resolvers, sql)
	}

	if len(*configFiles) > 0 {
		for _, path := range *configFiles {
			file, err := os.Open(path)
			if err != nil {
				log.Fatalf("static-zone: %s", err)
			}
			defer (*file).Close()

			config, err := ioutil.ReadAll(file)
			if err != nil {
				log.Fatalf("static-zone: %s", err)
			}

			r, err := landns.NewSimpleResolverFromConfig(config)
			if err != nil {
				log.Fatalf("static-zone: %s", err)
			}

			resolvers = append(resolvers, r)
		}
	}

	handler := landns.Handler{resolvers, metrics}

	go func() {
		log.Printf("API server listen on %s", *apiListen)
		log.Fatalf(http.ListenAndServe((*apiListen).String(), nil).Error())
	}()

	log.Printf("DNS server listen on %s", *dnsListen)
	log.Fatalf(dns.ListenAndServe((*dnsListen).String(), *dnsProtocol, handler).Error())
}
