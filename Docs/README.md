# Building and Configuration


## Generating a new Custom Collector

If you don't have an OTEL Custom Collector service daemon, you can use
the OpenTelemetry Custom Collector Builder tool to
[generate a new custom collector](./generate-custom-collector.md)
to contain the `trace2receiver` component.


## Configure Your Custom Collector

After building your custom collector and statically linking all of the
required components, you can run your custom collector.  It is
intended to be a long-running service daemon managed by the OS, such
as `launchctl(1)` on macOS, `systemd(1)` on Linux, or the Control
Panel Service Manager on Windows.  However, it is helpful to run it
interactively while you work on your configuration.

The collector requires a
[`config.yml` configuration file](./configure-custom-collector.md)
to specify which (of the linked) components you actually want to use
and how they should be connected and configured.

This `config.yml` file will be read in by the collector when it starts up,
so you should plan to distribute it with the executable.

If you want to change your `config.yml` or any of the filter or
privacy files that it references, you'll need to stop and restart your
collector service daemon, since these files are only read during startup.

_All pathnames in the `config.yml` file should be absolute paths rather
than relative paths to avoid startup working directory confusion when
run by the OS service manager._


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
