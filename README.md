# `trace2receiver` README



## About `trace2receiver`

This directory contains the source for the `trace2receiver`
component library.

* It is designed to be used within an
[OpenTelemetry (OTEL) Collector](https://opentelemetry.io/docs/collector/)
service daemon.
* It is an instance of a
[Trace Receiver](https://opentelemetry.io/docs/collector/trace-receiver/).
* It is responsible for listening to Trace2 telemetry data from Git
commands, translating it into OTEL in-memory data structures,
and forwarding it to other collector components.  For example, export
components to generate
[OTLP](https://opentelemetry.io/docs/specs/otel/protocol/) format
telemetry and transmit it to the cloud.



## Generating a new Custom Collector

If you don't have an OTEL Collector service daemon, you can use the
OTEL Collector Builder tool to
[generate a new custom collector](./Docs/generate-custom-collector.md)
to contain the `trace2receiver` component.



## Configure a Custom Collector

After building your custom collector and statically linking all of the
required components, you can run your collector service daemon.
It requires a
[`config.yml` configuration file](./Docs/configure-custom-collector.md)
to specify which (of the linked) components you actually want to use
and how they should be connected and configured.

This `config.yml` file will be read in by the collector when it starts up,
so you should plan to distribute it with the executable.

If you want to change your `config.yml` or any of the filter or
privacy files that it references, you'll need to stop and restart your
collector service daemon, since these files are only read during startup.



## Appendix: Caveats

### Unmonitored Git Commands

Long-running Git commands like `git fsmonitor--daemon run` that
operate background are incompatible with the `trace2receiver` because
they are designed to run for days and the OTEL telemetry is only
generated when the process exits.  The receiver automatically drops
the pipe/socket connection from such daemon commands as quickly as
possible to avoid wasting resources.



### Updating Filter Specifications

There have been requests to have the receiver periodically poll
some web endpoint for updated filter specifications.  This is
outside of the scope of the `trace2receiver` component, since it
operates as a component within an unknown OTEL Custom Collector.

This functionality can be easily provided by an Administrator
cron script to poll a web service that they own and restart the
collector as necessary.
