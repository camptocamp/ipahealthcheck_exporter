# ipahealthcheck_exporter

Prometheus exporter for exposing ipa-healthcheck metrics. It's essentialliy a wrapper around the ipa-healthcheck command.


## Prerequisites

 * Freeipa 4.8.0 at least, since this exporter uses the tool ["freeipa-healthcheck"](https://github.com/freeipa/freeipa-healthcheck).

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
Usage of ./ipa-healthcheck_exporter:
  -ipahealthcheck-path string
    	Path to the ipa-healthcheck tool. (default "/usr/bin/ipa-healthcheck")
  -metrics-path string
    	Path under which to expose metrics. (default "/metrics")
  -port int
    	Port on which to expose metrics. (default 9888)
```

## Exported Metrics

| Metric Name                                         | Description                                                                     |
| --------------------------------------------------- | ------------------------------------------------------------------------------- |
| `ipa_healthcheck_state`                             | State of a IPA healthcheck (1: active, 0: inactive)"                            |


## Prometheus

### Alerting :

## Misc

We currently have to use the --output-file option of the ipa-healthcheck command and a temp file to parse the checks otherwise some warnings are written on stdout alongside the json output.
