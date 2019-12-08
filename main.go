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
	dnsListen        = app.Flag("dns-listen", "Address for listen.").Short('L').Default(":53").TCP()
	dnsProtocol      = app.Flag("dns-protocol", "Protocol for listen.").Default("udp").Enum("udp", "tcp")
	upstreams        = app.Flag("upstream", "Upstream DNS server for recursive resolve. (e.g. 8.8.8.8:53)").Short('u').PlaceHolder("ADDRESS").TCPList()
	upstreamTimeout  = app.Flag("upstream-timeout", "Timeout for recursive resolve.").Default("100ms").Duration()
	cacheDisabled    = app.Flag("disable-cache", "Disable cache for recursive resolve.").Bool()
	redisAddr        = app.Flag("redis", "Address of Redis server for sharing recursive resolver's cache. (e.g. 127.0.0.1:6379)").PlaceHolder("ADDRESS").TCP()
	redisPassword    = app.Flag("redis-password", "Password of Redis server.").PlaceHolder("PASSWORD").String()
	redisDatabase    = app.Flag("redis-database", "Database ID of Redis server.").PlaceHolder("ID").Int()
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
	_, err := app.Parse(os.Args[1:])
	app.FatalIfError(err, "")

	metrics := landns.NewMetrics(*metricsNamespace)

	if *sqlitePath == "" {
		*sqlitePath = ":memory:"
	}
	dynamicResolver, err := landns.NewSqliteResolver(*sqlitePath, metrics)
	app.FatalIfError(err, "dynamic-zone")

	resolvers, err := loadStatisResolvers(*configFiles)
	app.FatalIfError(err, "static-zone")

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
		var forwardResolver landns.Resolver = landns.NewForwardResolver(us, *upstreamTimeout, metrics)
		if !*cacheDisabled {
			if *redisAddr != nil {
				forwardResolver, err = landns.NewRedisCache(*redisAddr, *redisDatabase, *redisPassword, forwardResolver, metrics)
				app.FatalIfError(err, "recursive: Redis cache")
			} else {
				forwardResolver = landns.NewLocalCache(forwardResolver, metrics)
			}
		}
		resolver = landns.AlternateResolver{resolver, forwardResolver}
	}

	defer func() {
		err := resolver.Close()
		app.FatalIfError(err, "failed to close")
	}()

	server := landns.Server{
		Metrics:         metrics,
		DynamicResolver: dynamicResolver,
		Resolvers:       resolver,
	}

	log.Printf("API server listen on %s", *apiListen)
	log.Printf("DNS server listen on %s", *dnsListen)
	log.Fatalf(server.ListenAndServe(*apiListen, *dnsListen, *dnsProtocol).Error())
}
