package trace2receiver

// Settings to enable/disable possibly GDPR-sensitive fields
// in the telemetry output.
type PiiSettings struct {
	// Lookup system hostname and add to process span.
	IncludeHostname bool `yaml:"hostname"`

	// Lookup the client username and add to process span.
	IncludeUsername bool `yaml:"username"`
}
