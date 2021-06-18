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
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/log"
)

var (
	metricsPath           string
	ipahealthcheckPath    string
	ipahealthcheckLogPath string
	port                  int

	ipahealthcheckServiceStateDesc = prometheus.NewDesc(
		"ipa_service_state",
		"State of the services monitored by IPA healthcheck (1: running, 0: not running)",
		[]string{"service"}, nil,
	)

	ipahealthcheckDogtagCheckDesc = prometheus.NewDesc(
		"ipa_dogtag_connectivity_check",
		"Check to verify dogtag basic connectivity. (1: success, 0: error)",
		[]string{"ipahealthcheck"}, nil,
	)

	ipahealthcheckReplicationCheckDesc = prometheus.NewDesc(
		"ipa_replication_check",
		"Replication checks (1: success, 0: error)",
		[]string{"ipahealthcheck"}, nil,
	)

	ipahealthcheckCertExpirationDesc = prometheus.NewDesc(
		"ipa_cert_expiration",
		"Expiration date of the certificates in warning or error state (unix timestamp)",
		[]string{"certificate_request_id"}, nil,
	)

	scrapedChecks = map[string]scrapedCheck{
		"ReplicationCheck": {
			scrape:      true,
			metricsDesc: ipahealthcheckReplicationCheckDesc,
		},
		"DogtagCertsConnectivityCheck": {
			scrape:      true,
			metricsDesc: ipahealthcheckDogtagCheckDesc,
		},
	}
)

type ipaCheck struct {
	Source string
	Check  string
	Result string
	Kw     map[string]interface{}
}

type scrapedCheck struct {
	scrape      bool
	metricsDesc *prometheus.Desc
}

type ipahealthcheckCollector struct {
	ipahealthcheckPath    string
	ipahealthcheckLogPath string
}

func init() {
	flag.StringVar(&metricsPath, "metrics-path", "/metrics", "Path under which to expose the metrics.")
	flag.StringVar(&ipahealthcheckPath, "ipahealthcheck-path", "/usr/bin/ipa-healthcheck", "Path to the ipa-healthcheck binary.")
	flag.StringVar(&ipahealthcheckLogPath, "ipahealthcheck-log-path", "/var/log/ipa/healthcheck/healthcheck.log", "Path to the ipa-healthcheck log file.")
	flag.IntVar(&port, "port", 9888, "Port on which to expose metrics.")
}

func (ic ipahealthcheckCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- ipahealthcheckServiceStateDesc
	ch <- ipahealthcheckDogtagCheckDesc
	ch <- ipahealthcheckReplicationCheckDesc
	ch <- ipahealthcheckCertExpirationDesc
}

func (ic ipahealthcheckCollector) Collect(ch chan<- prometheus.Metric) {
	log.Infof("Scraping metrics from %v", ic.ipahealthcheckPath)

	var checks []ipaCheck
	tmpFile, err := ioutil.TempFile("/dev/shm", "ipa-healthcheck.out")
	if err != nil {
		log.Fatal("Cannot write ipa-healthcheck output for parsing: ", err)
	}

	cmd := exec.Command(ic.ipahealthcheckPath, "--source", "ipahealthcheck.meta.services", "--output-file", tmpFile.Name())
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	if err != nil {
		log.Infof("ipa-healthcheck tool returned errors: %v", err)
	}

	jsonChecksOutput, err := ioutil.ReadFile(tmpFile.Name())
	if err != nil {
		log.Fatal("Cannot read file from ipa-healthcheck tool: ", err)
		os.Exit(1)
	}

	err = json.Unmarshal(jsonChecksOutput, &checks)
	if err != nil {
		log.Fatal("Cannot unmarshal json from ipa-healthcheck output: ", err)
		os.Exit(1)
	}

	for _, check := range checks {

		if check.Result == "SUCCESS" {
			ch <- prometheus.MustNewConstMetric(ipahealthcheckServiceStateDesc, prometheus.GaugeValue, 1.0, check.Check)
		} else {
			ch <- prometheus.MustNewConstMetric(ipahealthcheckServiceStateDesc, prometheus.GaugeValue, 0.0, check.Check)
		}
	}

	log.Infof("Scraping metrics from %v", ic.ipahealthcheckLogPath)

	jsonChecksOutput, err = ioutil.ReadFile(ic.ipahealthcheckLogPath)
	if err != nil {
		log.Error("Cannot read ipa-healthcheck log file: ", err)
	}

	err = json.Unmarshal(jsonChecksOutput, &checks)
	if err != nil {
		log.Error("Cannot unmarshal json from ipa-healthcheck log file: ", err)
	}

	for _, check := range checks {

		if scrapedChecks[check.Check].scrape {

			if check.Result == "SUCCESS" {
				ch <- prometheus.MustNewConstMetric(scrapedChecks[check.Check].metricsDesc, prometheus.GaugeValue, 1.0, check.Check)
			} else {
				ch <- prometheus.MustNewConstMetric(scrapedChecks[check.Check].metricsDesc, prometheus.GaugeValue, 0.0, check.Check)
			}
		}

		if check.Source == "ipahealthcheck.ipa.certs" && check.Check == "IPACertmongerExpirationCheck" {

			if check.Result == "WARNING" || check.Result == "ERROR" {

				timestamp, err := time.Parse("20060102150405Z", check.Kw["expiration_date"].(string))

				if err != nil {
					log.Infof("A problem occured while getting the certificate expiration (request id : %v) : %v", check.Kw["key"].(string), err)
				} else {
					ch <- prometheus.MustNewConstMetric(ipahealthcheckCertExpirationDesc, prometheus.GaugeValue, float64(timestamp.Unix()), check.Kw["key"].(string))
				}
			}
		}

	}

	defer os.Remove(tmpFile.Name())
}

func main() {

	flag.Parse()

	go func() {
		intChan := make(chan os.Signal, 1)
		termChan := make(chan os.Signal, 1)

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
		ipahealthcheckPath:    ipahealthcheckPath,
		ipahealthcheckLogPath: ipahealthcheckLogPath,
	}

	registry := prometheus.NewPedanticRegistry()

	registry.MustRegister(
		collector,
		prometheus.NewProcessCollector(prometheus.ProcessCollectorOpts{}),
		prometheus.NewGoCollector(),
	)

	http.Handle(metricsPath, promhttp.HandlerFor(registry, promhttp.HandlerOpts{}))

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		_, err := w.Write([]byte(`<html>
	            <head><title>IPA Healthcheck Exporter</title></head>
	            <body>
	            <h1>IPA Healthcheck Exporter</h1>
	            <p><a href='` + metricsPath + `'>Metrics</a></p>
	            </body>
	            </html>`))
		if err != nil {
			log.Infof("An error occured while writing http response: %v", err)
		}
	})

	log.Infof("ipa-healthcheck exporter listening on http://0.0.0.0:%d\n", port)

	if err := http.ListenAndServe(fmt.Sprintf(":%d", port), nil); err != nil {
		log.Fatal(err)
	}
}
