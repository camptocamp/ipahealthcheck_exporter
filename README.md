# ipahealthcheck-exporter

Prometheus exporter for exposing ipa-healthcheck metrics. It's essentialliy a wrapper around the ipa-healthcheck command.


## Prerequisites :

 * Freeipa 4.8.0 ate least, since this exporter uses the tool ["node_exporter"](https://github.com/freeipa/freeipa-healthcheck).

## Running :

You can run the exporter locally :

```sh
# ./ipa-healthcheck_exporter 
INFO[0000] ipa-healthcheck exporter listening on http://0.0.0.0:9888  source="ipahealthcheck-exporter.go:139"
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



## Misc

We currently have to use the --output-file option of the ipa-healthcheck command and a temp file to parse the checks otherwise some warnings are written on stdout alongside the json output.

