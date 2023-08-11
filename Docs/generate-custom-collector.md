# Generating a Custom Collector

An
[OTEL Custom Collector](https://github.com/open-telemetry/opentelemetry-collector/blob/main/docs/design.md#opentelemetry-collector-architecture)
is an instance of a stock table-driven service daemon that can receive
telemetry data from a variety of sources, transform or process the
data in some way, and export/relay the data to a variety of cloud
services.



## Generating Source Code for a Customer Collector

Source code for a custom collector is generated using the
[OTEL Collector Builder (OCB)](https://github.com/open-telemetry/opentelemetry-collector/tree/main/cmd/builder)
tool.  A `builder-config.yml` configuration file specifies the set
of supported
["receiver"](https://github.com/open-telemetry/opentelemetry-collector/blob/main/docs/design.md#receivers),
["processor"](https://github.com/open-telemetry/opentelemetry-collector/blob/main/docs/design.md#processors),
and
["exporter"](https://github.com/open-telemetry/opentelemetry-collector/blob/main/docs/design.md#exporters)
components that will be statically linked into the resulting custom collector executable.

The above link shows how to install the OCB tool, create a
`builder-config.yml` file, and run the tool generate your custom
collector source code.



### Example `builder-config.yml`

Your `builder-config.yml` file should list all of the components that
you want to use.  These will be statically linked into your collector's
executable.  For example:

```
dist:
  module: <my-module-name>
  name: <my-executable-name>
  output_path: <my-generated-source-directory>
  ...

exporters:
  - gomod: github.com/open-telemetry/opentelemetry-collector-contrib/exporter/azuremonitorexporter v0.76.1
  - import: go.opentelemetry.io/collector/exporter/loggingexporter
    gomod: go.opentelemetry.io/collector v0.76.1
  - import: go.opentelemetry.io/collector/exporter/otlpexporter
    gomod: go.opentelemetry.io/collector v0.76.1

receivers:
  - import: go.opentelemetry.io/collector/receiver/otlpreceiver
    gomod: go.opentelemetry.io/collector v0.76.1
  - gomod: github.com/git-ecosystem/trace2receiver v0.0.0

processors:
  - import: go.opentelemetry.io/collector/processor/batchprocessor
    gomod: go.opentelemetry.io/collector v0.76.1
```

Here we reference stock (core) OTLP and Logging components,
the Azure Monitor component from the
[OTEL Collector Contrib](https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main)
collection,
and the `trace2receiver` component from this repository.

All of these component libraries will be statically linked into your
custom collector.



### Running the Builder Tool


```
$ ~/go/bin/builder --config ./builder-config.yml --skip-compilation --skip-get-modules
$ cd <my-generated-source-directory>
$ go build
```
