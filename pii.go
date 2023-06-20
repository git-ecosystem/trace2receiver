package trace2receiver

import (
	"fmt"
	"os"

	"github.com/mitchellh/mapstructure"
	"gopkg.in/yaml.v2"
)

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

func parsePII(path string) (*PiiSettings, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("could not read PII settings '%s': '%s'",
			path, err.Error())
	}

	return parsePIIFromBuffer(data, path)
}

func parsePIIFromBuffer(data []byte, path string) (*PiiSettings, error) {
	m := make(map[interface{}]interface{})
	err := yaml.Unmarshal(data, &m)
	if err != nil {
		return nil, fmt.Errorf("could not parse PII YAML '%s': '%s'",
			path, err.Error())
	}

	pii := new(PiiSettings)
	err = mapstructure.Decode(m, pii)
	if err != nil {
		return nil, fmt.Errorf("could not decode PII settings '%s': '%s'",
			path, err.Error())
	}

	return pii, nil
}
