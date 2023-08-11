## Trace2 Receiver

The `trace2receiver` project is a
[trace receiver](https://opentelemetry.io/docs/collector/trace-receiver/)
component library for an
[OpenTelemetry Custom Collector](https://opentelemetry.io/docs/collector/)
daemon.  It receives
[Git Trace2](https://git-scm.com/docs/api-trace2#_the_event_format_target)
telemetry from local Git commands, translates it into an OpenTelemetry
format, and forwards it to other OpenTelemetry components.

This component is useful it you want to collect performance data for
Git commands, aggregate data from multiple users to create performance
dashboards, build distributed traces of nested Git commands, or
understand how the size and shape of your Git repositories affect
command performance.


## Background

This project is a GOLANG static library component that must be linked
into an OpenTelemetry Custom Collector along with other pipeline and
exporter components to process and forward the telemetry data to a
data store, such as Azure Monitor or another
[OTLP](https://opentelemetry.io/docs/specs/otel/protocol/)
aware cloud provider.

Setup and configuration details are provided in the
[Docs](./Docs/README.md).

This project is under active development, and loves contributions from the community.
Check out the
[CONTRIBUTING](./CONTRIBUTING.md)
guide for details on getting started.


## Requirements

This project is written in GOLANG and uses
[OpenTelemetry](https://opentelemetry.io/docs/getting-started/dev/)
libraries and tools.  See the OpenTelemetry documentation for more
information.

This project runs on Linux, macOS, and Windows.



## License

This project is licensed under the terms of the MIT open source license.
Please refer to [LICENSE](./LICENSE) for the full terms.


## Maintainers

See [CODEOWNERS](./CODEOWNERS) for a list of current project maintainers.


## Support

See [SUPPORT](./SUPPORT.md) for instructions on how to file bugs, make feature
requests, or seek help.
