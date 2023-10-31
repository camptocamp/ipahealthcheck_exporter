# ipahealthcheck_exporter

Prometheus exporter for exposing ipa-healthcheck metrics. It's essentially a wrapper around the ipa-healthcheck command.


## Prerequisites

 * The tool ["freeipa-healthcheck"](https://github.com/freeipa/freeipa-healthcheck) with the provided systemd timer enabled.

## Running

You can run the exporter locally :

```sh
# ./ipa-healthcheck_exporter 
INFO[0000] ipa-healthcheck exporter listening on http://0.0.0.0:9888  source="ipahealthcheck_exporter.go:139"
```

Or with a systemd service :

```
[Unit]
Description=Prometheus ipahealthcheck_exporter
Wants=basic.target
After=basic.target network.target

[Service]
User=ipahealthcheck-exporter
Group=ipahealthcheck-exporter
ExecStart=/usr/local/bin/ipahealthcheck_exporter

ExecReload=/bin/kill -HUP $MAINPID
KillMode=process
Restart=always

[Install]
WantedBy=multi-user.target
```

The following arguments are supported :

```sh
# ./ipa-healthcheck_exporter -h
Usage of ./ipahealthcheck_exporter:
  -address string
    	Address on which to expose metrics. (default "0.0.0.0")
  -ipahealthcheck-log-path string
    	Path to the ipa-healthcheck log file. (default "/var/log/ipa/healthcheck/healthcheck.log")
  -ipahealthcheck-path string
    	Path to the ipa-healthcheck binary. (default "/usr/bin/ipa-healthcheck")
  -metrics-path string
    	Path under which to expose the metrics. (default "/metrics")
  -port int
    	Port on which to expose metrics. (default 9888)
  -sudo
    	Use privilege escalation to run the health checks
  -v	Verbose mode (print more logs)
```

## Exported Metrics

```
# HELP ipa_cert_expiration Expiration date of the certificates in warning or error state (unix timestamp)
# TYPE ipa_cert_expiration gauge
ipa_cert_expiration{certificate_request_id="20200626075943"} 1.604761504e+09
...
# HELP ipa_dogtag_connectivity_check Check to verify dogtag basic connectivity. (1: success, 0: error)
# TYPE ipa_dogtag_connectivity_check gauge
ipa_dogtag_connectivity_check{ipahealthcheck="DogtagCertsConnectivityCheck"} 1
# HELP ipa_replication_check Replication checks (1: success, 0: error)
# TYPE ipa_replication_check gauge
ipa_replication_check{ipahealthcheck="ReplicationConflictCheck"} 1
# HELP ipa_service_state State of the services monitored by IPA healthcheck (1: running, 0: not running)
# TYPE ipa_service_state gauge
ipa_service_state{service="certmonger"} 1
ipa_service_state{service="httpd"} 1
...
```

## Prometheus

### Labels

The exporter labels are the following :
 * severity : Severity of the check ("success", "critical", "error", "warning)
 * source : Source (plugin / list of checks) of the check
 * check : Name of the check

### Alerts

### Alerting rules

Here are an example of two alerting rules to receive alerts when a check is in a bad state :

```
alert: IPAHealthcheckIsCritical
expr: ipa_healthcheck_state{severity="critical"} == 1
for: 5m
labels:
  severity: critical
annotations:
  description: "A IPA healthcheck is in critical state ( {{ $labels.source }} / {{ $labels.check }} )"
alert: IPAHealthcheckIsError
expr: ipa_healthcheck_state{severity="error"} == 1
for: 5m
labels:
  severity: error
annotations:
  description: A IPA healthcheck is in error state : ( {{ $labels.source }} / {{ $labels.check }} )" 
```

## Misc

When a check is in error you can rerun it on the server to have more informations about the problem with the following command :

```sh
# ipa-healthcheck --source <label "source"> --check <label "check">
```

We currently have to use the --output-file option of the ipa-healthcheck command and a temp file to parse the checks otherwise some warnings are written on stdout alongside the json output.

TODO :
 * Our own direct scraping mechanism (via ipalib) to not be tied to ipa-healthcheck and better performance.
