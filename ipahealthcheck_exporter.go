package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/log"
)

var (
	metricsPath        string
	ipahealthcheckPath string
	port               int

	ipahealthcheckStateDesc = prometheus.NewDesc(
		"ipa_healthcheck_state",
		"State of a IPA healthcheck (1: active, 0: inactive)",
		[]string{"uuid", "severity", "check"}, nil,
	)
)

type ipaCheck struct {
	Source   string
	Check    string
	Result   string
	Uuid     string
	When     string
	Duration string
}

type ipahealthcheckCollector struct {
	ipahealthcheckPath string
}

func init() {
	flag.StringVar(&metricsPath, "metrics-path", "/metrics", "Path under which to expose the metrics.")
	flag.StringVar(&ipahealthcheckPath, "ipahealthcheck-path", "/usr/bin/ipa-healthcheck", "Path to the ipa-healthcheck tool.")
	flag.IntVar(&port, "port", 9888, "Port on which to expose metrics.")
}

func (ic ipahealthcheckCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- ipahealthcheckStateDesc
}

func (ic ipahealthcheckCollector) Collect(ch chan<- prometheus.Metric) {
	log.Infof("Scraping metrics from %v", ic.ipahealthcheckPath)

	var checks []ipaCheck
	severityLevels := []string{"SUCCESS", "CRITICAL", "ERROR", "WARNING"}
	tmpFile, err := ioutil.TempFile("/dev/shm", "ipa-healthcheck.out")

	err = exec.Command(ic.ipahealthcheckPath, "--output-file", tmpFile.Name()).Run()
	if err != nil {
		log.Infof("ipa-healthcheck tool returned errors: ", err)
	}

	jsonChecksOutput, err := ioutil.ReadFile(tmpFile.Name())
	if err != nil {
		log.Fatal("Cannot read checks from ipa-healthcheck tool: ", err)
	}

	err = json.Unmarshal(jsonChecksOutput, &checks)
	if err != nil {
		log.Fatal("Cannot unmarshal checks from ipa-healthcheck tool: ", err)
		os.Exit(1)
	}

	for _, check := range checks {

		for _, level := range severityLevels {

			if level == check.Result {
				ch <- prometheus.MustNewConstMetric(ipahealthcheckStateDesc, prometheus.GaugeValue, 1.0, check.Uuid, strings.ToLower(level), check.Source+"."+check.Check)
			} else {
				ch <- prometheus.MustNewConstMetric(ipahealthcheckStateDesc, prometheus.GaugeValue, 0.0, check.Uuid, strings.ToLower(level), check.Source+"."+check.Check)

			}
		}
	}
}

func main() {

	flag.Parse()

	go func() {
		intChan := make(chan os.Signal)
		termChan := make(chan os.Signal)

		signal.Notify(intChan, syscall.SIGINT)
		signal.Notify(termChan, syscall.SIGTERM)

		select {
		case <-intChan:
			log.Infof("Received SIGINT, exiting")
			os.Exit(0)
		case <-termChan:
			log.Infof("Received SIGTERM, exiting")
			os.Exit(0)
		}
	}()

	collector := ipahealthcheckCollector{
		ipahealthcheckPath: ipahealthcheckPath,
	}

	registry := prometheus.NewPedanticRegistry()

	registry.MustRegister(
		collector,
		prometheus.NewProcessCollector(prometheus.ProcessCollectorOpts{}),
		prometheus.NewGoCollector(),
	)

	http.Handle(metricsPath, promhttp.HandlerFor(registry, promhttp.HandlerOpts{}))

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<html>
	            <head><title>IPA Healthcheck Exporter</title></head>
	            <body>
	            <h1>IPA Healthcheck Exporter</h1>
	            <p><a href='` + metricsPath + `'>Metrics</a></p>
	            </body>
	            </html>`))
	})

	log.Infof("ipa-healthcheck exporter listening on http://0.0.0.0:%d\n", port)

	if err := http.ListenAndServe(fmt.Sprintf(":%d", port), nil); err != nil {
		log.Fatal(err)
	}
}
