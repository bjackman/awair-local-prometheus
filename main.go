package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	awairAddress = flag.String("awair-address", "", "Base URL of Awair local API server")
)

// The next few bits are more or less copied from
// https://godoc.org/github.com/prometheus/client_golang/prometheus#example-Collector

var (
	awairScoreDesc = prometheus.NewDesc("awair_score", "Awair's 'Score' metric", nil, nil)
)

type Collector struct {
	awairBaseURL string
}

func (c *Collector) Describe(ch chan<- *prometheus.Desc) {
	// This bit is bad, it assumes that Collect will never fail to fetch
	// metrics. If it does fail at inopportune times, we violate the
	// contract of the Collector interface and things will presumably get
	// messed up in confusing ways.
	prometheus.DescribeByCollect(c, ch)
}

func (c *Collector) Collect(ch chan<- prometheus.Metric) {
	data, err := Get(c.awairBaseURL)
	if err != nil {
		// TODO: Count these errors. We need metrics for our metrics.
		log.Printf("Error getting data from Awair: %v", err)
		return
	}

	m, err := prometheus.NewConstMetric(awairScoreDesc, prometheus.GaugeValue, float64(data.Score))
	if err != nil {
		// TODO: Count these errors. We need metrics for our metrics.
		log.Printf("NewConstMetric: %v", err)
		return
	}
	ch <- m
}

// AwairAirDataResponse is the structure we expect from the /air/data-latest
// endpoints of the Awair local API.
type AirDataResponse struct {
	Timestamp     time.Time `json:"timestamp" `
	Score         int64     `json:"score"`
	DewPoint      float64   `json:"dew_point"`
	Temp          float64   `json:"temp"`
	Humid         float64   `json:"humid"`
	AbsHumid      float64   `json:"abs_humid"`
	Co2           int64     `json:"co2"`
	Co2Est        int64     `json:"co2_est"`
	Voc           int64     `json:"voc"`
	VocBaseline   int64     `json:"voc_baseline"`
	VocH2Raw      int64     `json:"voc_h2_raw"`
	VocEthanolRaw int64     `json:"voc_ethanol_raw"`
	Pm25          int64     `json:"pm25"`
	Pm10Est       int64     `json:"pm10_est"`
}

func Get(baseURL string) (*AirDataResponse, error) {
	url := baseURL + "/air-data/latest"
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to GET from Awair at %q: %v", url, err)
	}
	defer resp.Body.Close()

	fmt.Println("Response status:", resp.Status)

	if resp.StatusCode > 200 || resp.StatusCode > 299 {
		return nil, fmt.Errorf("Awair returned an error: %v", http.StatusText(resp.StatusCode))
	}

	data := AirDataResponse{}
	err = json.NewDecoder(resp.Body).Decode(&data)
	if err != nil {
		return nil, fmt.Errorf("failed to decode JSON: %v", err)
	}

	return &data, nil
}

func main() {
	flag.Parse()
	if len(*awairAddress) == 0 {
		log.Fatalf("--awair-address must be provided (%q)", *awairAddress)
	}

	err := prometheus.Register(&Collector{awairBaseURL: *awairAddress})
	if err != nil {
		log.Fatalf("Failed to register Prometheus collector: %v", err)
	}

	http.Handle("/metrics", promhttp.Handler())
	http.ListenAndServe(":8080", nil)
}
