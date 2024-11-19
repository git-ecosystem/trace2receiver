# Exporting Trace2 Data to Azure Monitor Application Insights

You can send Trace2 telemetry data from Git to Azure Monitor
Application Insights[^1] using the `azuremonitor` component[^2][^3].

Use the Azure Portal to create an Application Insights database and
enter the "instrumentation key" in your `config.yml` file.
A sample `config.yml` file is provided here to help you get started.

You can use the portal to visualize your telemetry data (both
individual span records or distributed traces using the end-to-end
transaction page).

You can also configure Azure Data Explorer to remotely access your
Application Insights database and view the span records.[^4]

Telemetry for Git commands (aka process spans) will appear in the
`requests` table.  Data for events internal to a command, such as
thread and region spans, will appear in the `dependencies` table.
(This separation is a feature of the `azuremonitor` exporter.)  So you
may need to do `union` or `join` Kusto queries to see all of the data
for an individual Git command.  However, it does make the Azure portal
`Application Map` feature more useful.

By default, Azure tries to strip out PII / GDPR data from incoming
telemetry and in some cases replaces it with less-specific data.[^5]
For example, the `azuremonitor` exporter adds many AppIns-level fields
to the telemetry data that is sent to Azure, such as `client_IP`.  In
the cloud, Azure may overwrite that field with `0.0.0.0` and add
`client_City`, `client_StateOrProvice`, and `client_CountryOrRegion`
fields.  See `DisableIpMasking` in [^5].

See also
[Config PII Settings](../../config-pii-settings.md).


[^1]: https://learn.microsoft.com/en-us/azure/azure-monitor/overview
[^2]: https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/exporter/azuremonitorexporter
[^3]: https://pkg.go.dev/github.com/open-telemetry/opentelemetry-collector-contrib/exporter/azuremonitorexporter
[^4]: https://learn.microsoft.com/en-us/azure/data-explorer/query-monitor-data
[^5]: https://learn.microsoft.com/en-us/azure/azure-monitor/app/ip-collection?tabs=framework%2Cnodejs
