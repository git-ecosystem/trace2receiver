# Config PII Settings

The PII Settings file contains privacy-related feature flags for the
`trace2receiver` component.  Currently, this includes flags to add
user and hostname data that may not be present in the original Trace2
data stream.  Later, it may include other flags to redact or not
redact sensitive data found within the Trace2 data stream.

NOTE: These flags may add GDPR-sensitive data to the OTEL telemetry
data stream.  Use them at your own risk.

The PII settings pathname is set in the
`receivers.trace2receiver.pii_settings`
parameter in the main `config.yml` file.

## `pii.yml` Syntax

The PII settings file has the following syntax:

```
include:
  hostname: <bool>
  username: <bool>
```

### `include.hostname`

Add the system hostname using the `trace2.pii.hostname` attribute.

### `include.username`

Add the username associated with the Git command using the `trace2.pii.username`
attribute.
