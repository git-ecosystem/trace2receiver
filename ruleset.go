package trace2receiver

import (
	"fmt"
	"os"

	"github.com/mitchellh/mapstructure"
	"gopkg.in/yaml.v2"
)

// RulesetDefinition captures the content of a custom ruleset YML file.
type RulesetDefinition struct {
	Commands RulesetCommands `mapstructure:"commands"`
	Defaults RulesetDefaults `mapstructure:"defaults"`
}

// RulesetCommands is used to map a Git command to a detail level.
// This allows us to have a different verbosity for different commands.
// For example, verbose for `git status` and drop for `git config`.
//
// We DO NOT support mapping to another ruleset because we want
// to avoid circular dependencies.
//
// A command key should be in the format described in
// `trace2Dataset.setQualifiedExeVerbModeName()`.
//
// The value must be one of [`DetailLevelDropName`, ... ].
type RulesetCommands map[string]string

// RulesetDefaults defines default values for this custom ruleset.
type RulesetDefaults struct {

	// The default detail level to use when exec+verb+mode
	// lookup fails.
	DetailLevelName string `mapstructure:"detail"`
}

// Parse a `ruleset.yml` and decode.
func parseRuleset(path string) (*RulesetDefinition, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("ruleset could not read '%s': '%s'",
			path, err.Error())
	}

	return parseRulesetFromBuffer(data, path)
}

// Parse a buffer containing the contents of a `ruleset.yml` and decode.
// This separation is primarily for writing test code.
func parseRulesetFromBuffer(data []byte, path string) (*RulesetDefinition, error) {
	m := make(map[interface{}]interface{})
	err := yaml.Unmarshal(data, &m)
	if err != nil {
		return nil, fmt.Errorf("ruleset could not parse YAML '%s': '%s'",
			path, err.Error())
	}

	rsdef := new(RulesetDefinition)
	err = mapstructure.Decode(m, rsdef)
	if err != nil {
		return nil, fmt.Errorf("ruleset could not decode '%s': '%s'",
			path, err.Error())
	}

	for k_cmd, v_dl := range rsdef.Commands {
		// Commands must map to detail levels and not to another ruleset (to
		// avoid lookup loops).
		_, err = getDetailLevel(v_dl)
		if len(k_cmd) == 0 || err != nil {
			return nil, fmt.Errorf("ruleset '%s' has invalid command '%s':'%s'",
				path, k_cmd, v_dl)
		}
	}

	if len(rsdef.Defaults.DetailLevelName) > 0 {
		// The rulset default detail level must be a detail level and not the
		// name of another ruleset (to avoid lookup loops).
		_, err = getDetailLevel(rsdef.Defaults.DetailLevelName)
		if err != nil {
			return nil, fmt.Errorf("ruleset '%s' has invalid default detail level",
				path)
		}
	} else {
		// If the custom ruleset did not define a ruleset-specific default
		// detail level, assume the builtin global default.
		rsdef.Defaults.DetailLevelName = DetailLevelDefaultName
	}

	return rsdef, nil
}
