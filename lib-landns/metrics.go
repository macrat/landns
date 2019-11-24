package landns

import (
	"fmt"
	"net/http"

	"github.com/miekg/dns"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func qtypeToString(qtype uint16) string {
	switch qtype {
	case dns.TypeA:
		return "A"
	case dns.TypeNS:
		return "NS"
	case dns.TypeCNAME:
		return "CNAME"
	case dns.TypePTR:
		return "PTR"
	case dns.TypeMX:
		return "MX"
	case dns.TypeTXT:
		return "TXT"
	case dns.TypeAAAA:
		return "AAAA"
	case dns.TypeSRV:
		return "SRV"
	default:
		return ""
	}
}

type Metrics struct {
	queryCount      prometheus.Counter
	skipCount       prometheus.Counter
	resolveCounters map[string]prometheus.Counter
	missCounters    map[string]prometheus.Counter
	errorCounters   map[string]prometheus.Counter
	resolveTime     prometheus.Gauge
}

func newCounter(namespace, name string, labels prometheus.Labels) prometheus.Counter {
	return prometheus.NewCounter(prometheus.CounterOpts{
		Namespace:   namespace,
		Name:        fmt.Sprintf("%s_count", name),
		ConstLabels: labels,
	})
}

func NewMetrics(namespace string) *Metrics {
	resolves := map[string]prometheus.Counter{}
	misses := map[string]prometheus.Counter{}
	errors := map[string]prometheus.Counter{}

	for _, qtype := range []string{"A", "AAAA", "PTR", "SRV", "TXT"} {
		resolves[qtype] = newCounter(namespace, "resolve", prometheus.Labels{"type": qtype, "result": "hit"})
		misses[qtype] = newCounter(namespace, "resolve", prometheus.Labels{"type": qtype, "result": "miss"})
		errors[qtype] = newCounter(namespace, "resolve_error", prometheus.Labels{"type": qtype})
	}

	return &Metrics{
		queryCount: newCounter(namespace, "received_message", prometheus.Labels{"type": "query"}),
		skipCount:  newCounter(namespace, "received_message", prometheus.Labels{"type": "another"}),

		resolveCounters: resolves,
		missCounters:    misses,
		errorCounters:   errors,

		resolveTime: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "resolve_duration_seconds",
		}),
	}
}

func (m *Metrics) Register() error {
	return prometheus.Register(m)
}

func (m *Metrics) HTTPHandler() http.Handler {
	return promhttp.Handler()
}

func (m *Metrics) Describe(ch chan<- *prometheus.Desc) {
	m.queryCount.Describe(ch)
	m.skipCount.Describe(ch)

	for _, c := range m.resolveCounters {
		c.Describe(ch)
	}
	for _, c := range m.missCounters {
		c.Describe(ch)
	}
	for _, c := range m.errorCounters {
		c.Describe(ch)
	}

	m.resolveTime.Describe(ch)
}

func (m *Metrics) Collect(ch chan<- prometheus.Metric) {
	m.queryCount.Collect(ch)
	m.skipCount.Collect(ch)

	for _, c := range m.resolveCounters {
		c.Collect(ch)
	}
	for _, c := range m.missCounters {
		c.Collect(ch)
	}
	for _, c := range m.errorCounters {
		c.Collect(ch)
	}

	m.resolveTime.Collect(ch)
}

func (m *Metrics) makeTimer(skipped bool) func(*dns.Msg) {
	timer := prometheus.NewTimer(prometheus.ObserverFunc(m.resolveTime.Set))
	return func(response *dns.Msg) {
		timer.ObserveDuration()

		counters := m.resolveCounters
		if len(response.Answer) == 0 {
			counters = m.missCounters
		}

		for _, q := range response.Question {
			if counter, ok := counters[qtypeToString(q.Qtype)]; ok {
				counter.Inc()
			}
		}
	}
}

func (m *Metrics) Start(request *dns.Msg) func(*dns.Msg) {
	if request.Opcode == dns.OpcodeQuery {
		m.queryCount.Inc()
		return m.makeTimer(false)
	} else {
		m.skipCount.Inc()
		return m.makeTimer(true)
	}
}

func (m *Metrics) Error(q dns.Question, err error) {
	if counter, ok := m.errorCounters[qtypeToString(q.Qtype)]; ok {
		counter.Inc()
	}
}
