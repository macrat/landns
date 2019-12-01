package main

import (
	"io/ioutil"
	"log"
	"net"
	"os"

	"github.com/alecthomas/kingpin"

	"github.com/macrat/landns/lib-landns"
)

var (
	configFiles      = kingpin.Flag("static-zone", "Path to static-zone configuration file.").Short('s').PlaceHolder("PATH").ExistingFiles()
	sqlitePath       = kingpin.Flag("dynamic-zone", "Path to dynamic-zone database path.").Short('d').PlaceHolder("PATH").String()
	apiListen        = kingpin.Flag("api-listen", "Address for API and metrics.").Short('l').Default(":9353").TCP()
	dnsListen        = kingpin.Flag("dns-listen", "Address for listen.").Default(":53").TCP()
	dnsProtocol      = kingpin.Flag("dns-protocol", "Protocol for listen.").Default("udp").Enum("udp", "tcp")
	upstreams        = kingpin.Flag("upstream", "Upstream DNS server for recurion resolve.").TCPList()
	upstreamTimeout  = kingpin.Flag("upstream-timeout", "Timeout for recursion resolve.").Default("100ms").Duration()
	metricsNamespace = kingpin.Flag("metrics-namespace", "Namespace of prometheus metrics.").Default("landns").String()
)

func main() {
	kingpin.Parse()

	metrics := landns.NewMetrics(*metricsNamespace)

	if *sqlitePath == "" {
		*sqlitePath = ":memory:"
	}
	dynamicResolver, err := landns.NewSqliteResolver(*sqlitePath, metrics)
	if err != nil {
		log.Fatalf("dynamic-zone: %s", err)
	}

	resolverSet := landns.ResolverSet{}
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

		resolverSet = append(resolverSet, r)
	}

	var staticResolver landns.Resolver = resolverSet

	if len(*upstreams) > 0 {
		us := make([]*net.UDPAddr, len(*upstreams))
		for i, u := range *upstreams {
			us[i] = &net.UDPAddr{
				IP:   u.IP,
				Port: u.Port,
				Zone: u.Zone,
			}
		}
		staticResolver = landns.AlternateResolver{
			staticResolver,
			landns.NewForwardResolver(us, *upstreamTimeout, metrics),
		}
	}

	server := landns.Server{
		Metrics:         metrics,
		DynamicResolver: dynamicResolver,
		StaticResolver:  staticResolver,
	}

	log.Printf("API server listen on %s", *apiListen)
	log.Printf("DNS server listen on %s", *dnsListen)
	log.Fatalf(server.ListenAndServe(*apiListen, *dnsListen, *dnsProtocol).Error())
}
