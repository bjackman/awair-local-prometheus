package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	awairAddress = flag.String("awair-address", "", "Base URL of Awair local API server")
)

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

func main() {
	flag.Parse()
	if len(*awairAddress) == 0 {
		log.Fatalf("--awair-address must be provided (%q)", *awairAddress)
	}

	http.Handle("/metrics", promhttp.Handler())

	url := *awairAddress + "/air-data/latest"
	resp, err := http.Get(url)
	if err != nil {
		log.Fatalf("Failed to GET %q: %v", url, err)
	}
	defer resp.Body.Close()

	fmt.Println("Response status:", resp.Status)

	data := AirDataResponse{}
	err = json.NewDecoder(resp.Body).Decode(&data)
	if err != nil {
		log.Fatalf("Failed to decode JSON: %v", err)
	}
	log.Printf("%+v", data)

	http.ListenAndServe(":8080", nil)
}
