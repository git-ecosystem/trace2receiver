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
  - gomod: github.com/github/trace2receiver v0.0.0

processors:
  - import: go.opentelemetry.io/collector/processor/batchprocessor
    gomod: go.opentelemetry.io/collector v0.76.1
```

Here we reference stock (core) OTLP and Logging components,
the Azure Monitor component from the
[OTEL Collector Contrib](https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main)
collection,
and the `trace2receiver` component from a private repository.

All of these component libraries will be statically linked into your
custom collector.



### Referencing `trace2receiver` as a Private Component

_The current plan (as of 2023/07/14) is to open source the
 `trace2receiver` component and move it to a public repo or contribute
 it to OTEL "contrib" collection so that it can be referenced like any
 other published GOLANG module.  However, it is presently in a private
 repository not visible to the normal GOLANG tooling and that can
 cause problems when generating and compiling a custom collector using
 it._

*TODO* When published, remove this entire section.

```
$ ~/go/bin/builder --config ./builder-config.yml --skip-compilation --skip-get-modules
$ cd <my-generated-source-directory>
$ go mod tidy
```

You may see errors of the form:

```
% go mod tidy
go: downloading github.com/github/trace2receiver v0.0.0
<my-generated-source-directory> imports
	github.com/github/trace2receiver: github.com/github/trace2receiver@v0.0.0: verifying module: github.com/github/trace2receiver@v0.0.0: reading https://sum.golang.org/lookup/github.com/github/trace2receiver@v0.0.0: 404 Not Found
	server response:
	not found: github.com/github/trace2receiver@v0.0.0: invalid version: git ls-remote -q origin in /tmp/gopath/pkg/mod/cache/vcs/39f0793859cb29d57838fc7a3fb50d849118c1bb22c3f093e7950bbf6b087b58: exit status 128:
		fatal: could not read Username for 'https://github.com': terminal prompts disabled
	Confirm the import path was entered correctly.
	If this is a private repository, see https://golang.org/doc/faq#git_https for additional information.
```

This happens because `go` is trying to find the private `trace2receiver` repository
in Google's public database cache.

Try adding:

```
% GOPRIVATE=github.com/github/trace2receiver
% export GOPRIVATE
% go mod tidy
```

Then build your custom collector as usual.

For more information, see:
* https://www.digitalocean.com/community/tutorials/how-to-use-a-private-go-module-in-your-own-project#configuring-go-to-access-private-modules
* https://www.digitalocean.com/community/tutorials/how-to-use-a-private-go-module-in-your-own-project
