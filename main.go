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
	app              = kingpin.New("landns", "A DNS server for developers for home use.")
	configFiles      = app.Flag("config", "Path to static-zone configuration file.").Short('c').PlaceHolder("PATH").ExistingFiles()
	sqlitePath       = app.Flag("sqlite", "Path to dynamic-zone sqlite3 database path. In default, dynamic-zone will not save to disk.").Short('s').PlaceHolder("PATH").String()
	apiListen        = app.Flag("api-listen", "Address for API and metrics.").Short('l').Default(":9353").TCP()
	dnsListen        = app.Flag("dns-listen", "Address for listen.").Default(":53").TCP()
	dnsProtocol      = app.Flag("dns-protocol", "Protocol for listen.").Default("udp").Enum("udp", "tcp")
	upstreams        = app.Flag("upstream", "Upstream DNS server for recurion resolve.").Short('u').TCPList()
	upstreamTimeout  = app.Flag("upstream-timeout", "Timeout for recursion resolve.").Default("100ms").Duration()
	metricsNamespace = app.Flag("metrics-namespace", "Namespace of prometheus metrics.").Default("landns").String()
)

func loadStatisResolvers(files []string) (resolver landns.ResolverSet, err error) {
	for _, path := range files {
		file, err := os.Open(path)
		if err != nil {
			return nil, err
		}
		defer (*file).Close()

		config, err := ioutil.ReadAll(file)
		if err != nil {
			return nil, err
		}

		r, err := landns.NewSimpleResolverFromConfig(config)
		if err != nil {
			return nil, err
		}

		resolver = append(resolver, r)
	}

	return resolver, nil
}

func main() {
	app.Parse(os.Args[1:])

	metrics := landns.NewMetrics(*metricsNamespace)

	if *sqlitePath == "" {
		*sqlitePath = ":memory:"
	}
	dynamicResolver, err := landns.NewSqliteResolver(*sqlitePath, metrics)
	if err != nil {
		log.Fatalf("dynamic-zone: %s", err)
	}

	resolvers, err := loadStatisResolvers(*configFiles)
	if err != nil {
		log.Fatalf("static-zone: %s", err)
	}
	resolvers = append(resolvers, dynamicResolver)

	var resolver landns.Resolver = resolvers
	if len(*upstreams) > 0 {
		us := make([]*net.UDPAddr, len(*upstreams))
		for i, u := range *upstreams {
			us[i] = &net.UDPAddr{
				IP:   u.IP,
				Port: u.Port,
				Zone: u.Zone,
			}
		}
		resolver = landns.AlternateResolver{
			resolver,
			landns.NewForwardResolver(us, *upstreamTimeout, metrics),
		}
	}

	server := landns.Server{
		Metrics:         metrics,
		DynamicResolver: dynamicResolver,
		Resolvers:       resolver,
	}

	log.Printf("API server listen on %s", *apiListen)
	log.Printf("DNS server listen on %s", *dnsListen)
	log.Fatalf(server.ListenAndServe(*apiListen, *dnsListen, *dnsProtocol).Error())
}
