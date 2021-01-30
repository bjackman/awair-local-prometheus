package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	awairAddress  = flag.String("awair-address", "", "Base URL of Awair local API server")
	listenAddress = flag.String("listen-address", ":8080", "Address to serve metrics on")
)

// The next few bits are more or less copied from
// https://godoc.org/github.com/prometheus/client_golang/prometheus#example-Collector

type Collector struct {
	awairBaseURL string
}

func (c *Collector) Describe(ch chan<- *prometheus.Desc) {
	// This bit is bad, it assumes that Collect will never fail to fetch
	// metrics. If it does fail at inopportune times, we violate the
	// contract of the Collector interface and things will presumably get
	// messed up in confusing ways.
	for _, name := range ExpectedMetrics {
		ch <- prometheus.NewDesc(name, "", nil, nil)
	}
}

func (c *Collector) Collect(ch chan<- prometheus.Metric) {
	data, err := GetAirData(c.awairBaseURL)
	if err != nil {
		// TODO: Count these errors. We need metrics for our metrics.
		log.Printf("Error getting data from Awair: %v", err)
		return
	}

	for name, val := range data.Metrics {
		desc := prometheus.NewDesc(name, "", nil, nil)
		m, err := prometheus.NewConstMetric(desc, prometheus.GaugeValue, val)
		if err != nil {
			// TODO: Count these errors. We need metrics for our metrics.
			log.Printf("NewConstMetric: %v", err)
			continue
		}

		ch <- prometheus.NewMetricWithTimestamp(data.Timestamp, m)
	}
}

// ExpectedMetrics is the list of fields (excluding "timestamp") I expect the
// Awair Local API to return.
var ExpectedMetrics = []string{
	"score", "dew_point", "temp", "humid", "abs_humid", "co2", "co2_est",
	"voc", "voc_baseline", "voc_h2_raw", "voc_ethanol_raw", "pm25",
	"pm10_est",
}

// AwairAirDataResponse represents the air data returned by the Local API.
type AirData struct {
	// Timestamp reported by the Awair device
	Timestamp time.Time
	// Metrics reported by the device. See ExpectedMetrics for the fields I
	// get from my device.
	Metrics map[string]float64
}

// This metric instrements this service itself.
var awairGetCounter = promauto.NewCounterVec(prometheus.CounterOpts{
	Name: "awair_gets",
	Help: "Calls to the Get function, reading from the Awair local API",
}, []string{"result"})

// GetAirData reads data from the AwairLocal API, parses it and returns it.
func GetAirData(baseURL string) (*AirData, error) {
	url := baseURL + "/air-data/latest"
	resp, err := http.Get(url)
	if err != nil {
		// awairGetCounter.Inc()
		awairGetCounter.With(prometheus.Labels{"result": "failed-get"}).Inc()
		return nil, fmt.Errorf("failed to GET from Awair at %q: %v", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode > 200 || resp.StatusCode > 299 {
		awairGetCounter.With(prometheus.Labels{"result": "failed-get"}).Inc()
		return nil, fmt.Errorf("Awair returned an error: %v", http.StatusText(resp.StatusCode))
	}

	var fields map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&fields)
	if err != nil {
		awairGetCounter.With(prometheus.Labels{"result": "failed-decode"}).Inc()
		return nil, fmt.Errorf("failed to JSON: %v", err)
	}

	data := AirData{Metrics: make(map[string]float64)}

	// Parse the timestamp field from the JSON
	ts, ok := fields["timestamp"]
	if !ok {
		awairGetCounter.With(prometheus.Labels{"result": "failed-decode"}).Inc()
		return nil, fmt.Errorf("no 'timestamp' field")
	}
	tsString, ok := ts.(string)
	if !ok {
		return nil, fmt.Errorf("'timestamp' field %v not a string", ts)
	}
	if data.Timestamp, err = time.Parse(time.RFC3339, tsString); err != nil {
		awairGetCounter.With(prometheus.Labels{"result": "failed-decode"}).Inc()
		return nil, fmt.Errorf("failed ot parse timestamp: %v", err)
	}
	delete(fields, "timestamp")
	// Now go through the other fields, they should all be float64s.
	for key, val := range fields {
		floatVal, ok := val.(float64)
		if !ok {
			log.Printf("Got non-float64 value %q (%v) in Awair response", key, val)
			continue
		}
		data.Metrics[key] = floatVal
	}

	awairGetCounter.With(prometheus.Labels{"result": "success"}).Inc()
	return &data, nil
}

func main() {
	flag.Parse()
	if len(*awairAddress) == 0 {
		log.Fatalf("--awair-address must be provided (%q)", *awairAddress)
	}

	// Add handler for metrics about this service.
	http.Handle("/metrics", promhttp.Handler())

	// Add handler for the actual air data metrics.
	reg := prometheus.NewPedanticRegistry()
	err := reg.Register(&Collector{awairBaseURL: *awairAddress})
	if err != nil {
		log.Fatalf("Failed to register Prometheus collector: %v", err)
	}
	http.Handle("/air-data", promhttp.HandlerFor(reg, promhttp.HandlerOpts{}))
	http.ListenAndServe(*listenAddress, nil)
}
