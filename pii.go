package trace2receiver

// Settings to enable/disable possibly GDPR-sensitive fields
// in the telemetry output.
type PiiSettings struct {
	Include PiiInclude `mapstructure:"include"`
}

type PiiInclude struct {
	// Lookup system hostname and add to process span.
	Hostname bool `mapstructure:"hostname"`

	// Lookup the client username and add to process span.
	Username bool `mapstructure:"username"`
}

func parsePiiFile(path string) (*PiiSettings, error) {
	return parseYmlFile[PiiSettings](path, parsePiiFromBuffer)
}

func parsePiiFromBuffer(data []byte, path string) (*PiiSettings, error) {
	pii, err := parseYmlBuffer[PiiSettings](data, path)
	if err != nil {
		return nil, err
	}

	// TODO insert any post-parse validation or data structure setup here.

	return pii, nil
}
