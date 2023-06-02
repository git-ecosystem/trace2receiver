package trace2receiver

// RSDefinition captures the content of a custom ruleset YML file.
type RSDefinition struct {
	CmdMap   RSCmdMap   `mapstructure:"commands"`
	Defaults RSDefaults `mapstructure:"defaults"`
}

// RSCmdMap is used to map a Git command to a detail level.
// We DO NOT support mapping to another ruleset because we want
// to avoid circular dependencies.
//
// A command key should be in the format described in
// `trace2Dataset.setQualifiedExeVerbModeName()`.
//
// The value must be one of the `FSDetailLevel*Names`.
type RSCmdMap map[string]string

// RSDefaults defines default values for this custom ruleset.
type RSDefaults struct {

	// The default detail level to use when exec+verb+mode
	// lookup fails.
	DetailLevelName string `mapstructure:"detail"`
}
