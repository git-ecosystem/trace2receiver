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
	// Lookup system hostname and add to process span.
	IncludeHostname bool `mapstructure:"hostname"`

	// Lookup the client username and add to process span.
	IncludeUsername bool `mapstructure:"username"`
}

func parsePII(path string) (*PiiSettings, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("pii_settings could not read '%s': '%s'",
			path, err.Error())
	}

	m := make(map[interface{}]interface{})
	err = yaml.Unmarshal(data, &m)
	if err != nil {
		return nil, fmt.Errorf("pii_settings could not parse YAML '%s': '%s'",
			path, err.Error())
	}

	pii := new(PiiSettings)
	err = mapstructure.Decode(m, pii)
	if err != nil {
		return nil, fmt.Errorf("pii_settings could not decode '%s': '%s'",
			path, err.Error())
	}

	return pii, nil
}
