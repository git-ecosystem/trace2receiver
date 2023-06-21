package trace2receiver

import (
	"fmt"
	"os"

	"github.com/mitchellh/mapstructure"
	"gopkg.in/yaml.v2"
)

type MyYmlFileTypes interface {
	RulesetDefinition | FilterSettings | PiiSettings
}

type MyYmlParseBufferFn[T MyYmlFileTypes] func(data []byte, path string) (*T, error)

func parseYmlFile[T MyYmlFileTypes](path string, fnPB MyYmlParseBufferFn[T]) (*T, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("could not read YML '%s': '%s'",
			path, err.Error())
	}

	return fnPB(data, path)
}

func parseYmlBuffer[T MyYmlFileTypes](data []byte, path string) (*T, error) {
	m := make(map[interface{}]interface{})
	err := yaml.Unmarshal(data, &m)
	if err != nil {
		return nil, fmt.Errorf("could not parse YAML '%s': '%s'",
			path, err.Error())
	}

	p := new(T)
	err = mapstructure.Decode(m, p)
	if err != nil {
		return nil, fmt.Errorf("could not decode '%s': '%s'",
			path, err.Error())
	}

	return p, nil
}
