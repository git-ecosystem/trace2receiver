# Config PII Settings

The PII settings contain privacy-related feature flags for the
`trace2receiver` component.  Currently, this includes flags to add
user and hostname data that may not be present in the original Trace2
data stream.  Later, it may include other flags to redact or not
redact sensitive data found within the Trace2 data stream.

NOTE: These flags may add GDPR-sensitive data to the OTEL telemetry
data stream.  Use them at your own risk.

The PII settings are specified inline under the
`receivers.trace2receiver.pii`
parameter in the main `config.yml` file.  Alternatively, you can use
the `${file:PATH}` syntax to reference an external YAML file.

For backwards compatibility, you can also specify a plain file path
string (without the `${file:}` wrapper) as the value of the `pii`
field, and the receiver will read and parse the YAML file at that
path.

## PII Settings Syntax

The PII settings have the following syntax:

```
pii:
  include:
    hostname: <bool>
    username: <bool>
```

### `include.hostname`

Add the system hostname using the `trace2.pii.hostname` attribute.

### `include.username`

Add the username associated with the Git command using the `trace2.pii.username`
attribute.
