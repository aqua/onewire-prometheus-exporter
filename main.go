package main

import (
	"flag"
	"log"
	"math"
	"net/http"
	"strconv"

	"github.com/aqua/raspberrypi/onewire"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var listen = flag.String("listen", ":9456", "(Host and) port to listen on for Prometheus export")
var onlyDevice = flag.String("only-device", "", "Only report the given device ID (for testing)")

func export(devices []*onewire.DS18S20) {
	for i := range devices {
		func(d *onewire.DS18S20) {
			prometheus.MustRegister(prometheus.NewGaugeFunc(
				prometheus.GaugeOpts{
					Namespace: "onewire",
					Name:      "temperature_degrees_celsius",
					Help:      "Temperature sampled from a single sensor, in degrees celsius",
					ConstLabels: prometheus.Labels{
						"id":     strconv.FormatUint(d.Id, 10),
						"device": d.HumanId(),
						"model":  d.Model(),
					},
				},
				func() float64 {
					md, err := d.Read()
					if err != nil {
						log.Printf("Error reading %s: %v", d.HumanId(), err)
						return math.NaN()
					}
					return float64(md) / 1000.
				}))
		}(devices[i])
	}
}

func main() {
	flag.Parse()

	ids, err := onewire.Scan()
	if err != nil {
		log.Fatalf("Unable to scan for 1wire devices: %v", err)
	}
	if *onlyDevice != "" {
		ids = []string{*onlyDevice}
	}

	devices := []*onewire.DS18S20{}
	for _, id := range ids {
		func(ID string) {
			if d, err := onewire.NewDS18S20(ID); err != nil {
				log.Printf("Error opening %s as a DS18S20 device (%v); skipping it", ID, err)
			} else {
				devices = append(devices, d)
			}
		}(id)
	}
	export(devices)

	http.Handle("/metrics", promhttp.Handler())
	log.Fatal(http.ListenAndServe(*listen, nil))
}
