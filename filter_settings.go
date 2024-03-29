package trace2receiver

import (
	"fmt"
	"strings"
)

// FilterSettings describes how we should filter the OTLP output
// that we generate.  It also describes the special keys that we
// look for in the Trace2 event stream to help us decide how to
// filter data for a particular command.
type FilterSettings struct {
	Keynames  FilterKeynames  `mapstructure:"keynames"`
	Nicknames FilterNicknames `mapstructure:"nicknames"`
	Rulesets  FilterRulesets  `mapstructure:"rulesets"`
	Defaults  FilterDefaults  `mapstructure:"defaults"`

	// The set of custom rulesets defined in YML are each parsed
	// and loaded into definitions so that we can use them.
	rulesetDefs map[string]*RulesetDefinition
}

// FilterKeynames defines the names of the Git config settings that
// will be used in `def_param` events to send repository/worktree
// data to us.  This lets a site have their own namespace for
// these keys.  Some of these keys will also be sent to the cloud.
type FilterKeynames struct {

	// NicknameKey defines the Git config setting that can be used
	// to send an optional user-friendly id or nickname for a repo
	// or worktree.
	//
	// We can use the nickname to decide how to filter data
	// for the repo and to identify the repo in the cloud (and
	// possibly without exposing any PII or the actualy identity
	// of the repo/worktree).
	//
	// This can eliminate the need to rely on `remote.origin.url`
	// or the worktree root directory to identify (or guess at
	// the identity of) the repo.
	NicknameKey string `mapstructure:"nickname_key"`

	// RuleSetKey defines the Git config setting that can be used
	// to optionally send the name of the desired filter ruleset.
	// This value overrides any implied ruleset associated with
	// the RepoIdKey.
	RulesetKey string `mapstructure:"ruleset_key"`
}

// FilterDefaults defines default filtering values.
type FilterDefaults struct {

	// Ruleset defines the default ruleset or detail level to be
	// used when we receive data from a repo/worktree that does
	// not explicitly name one or does not have a nickname mapping.
	//
	// If not set, we default to the absolute default.
	RulesetName string `mapstructure:"ruleset"`
}

// FilterNicknames is used to map a repo nickname to the name of the
// ruleset or detail-level that should be used.
//
// This table is optional.
type FilterNicknames map[string]string

// FilterRulesets is used to map a custom ruleset name to the pathname
// of the associated YML file.  This form is used when parsing the
// filter settings YML file.  We use this to create the real ruleset
// table (possibly with lazy loading).
type FilterRulesets map[string]string

// Parse `filter.yml` in decode.
func parseFilterSettings(path string) (*FilterSettings, error) {
	return parseYmlFile[FilterSettings](path, parseFilterSettingsFromBuffer)
}

// Parse a buffer containing the contents of a `filter.yml` and decode.
func parseFilterSettingsFromBuffer(data []byte, path string) (*FilterSettings, error) {
	fs, err := parseYmlBuffer[FilterSettings](data, path)
	if err != nil {
		return nil, err
	}

	// After parsing the YML and populating the `mapstructure` fields, we need
	// to validate them and/or build internal structures from them.

	// For each custom ruleset [<name> -> <path>] in the table (the map[string]string),
	// create a peer entry in the internal [<name> -> <rsdef>] table and preload
	// the various `ruleset.yml` files.
	fs.rulesetDefs = make(map[string]*RulesetDefinition)
	for k_rs_name, v_rs_path := range fs.Rulesets {
		if !strings.HasPrefix(k_rs_name, "rs:") || len(k_rs_name) < 4 || len(v_rs_path) == 0 {
			return nil, fmt.Errorf("ruleset has invalid name or pathname'%s':'%s'", k_rs_name, v_rs_path)
		}

		fs.rulesetDefs[k_rs_name], err = parseRulesetFile(v_rs_path)
		if err != nil {
			return nil, err
		}
	}

	return fs, nil
}

// Add a ruleset to the filter settings.  This is primarily for writing test code.
func (fs *FilterSettings) addRuleset(rs_name string, path string, rsdef *RulesetDefinition) {
	if fs.Rulesets == nil {
		fs.Rulesets = make(FilterRulesets)
	}
	fs.Rulesets[rs_name] = path

	if fs.rulesetDefs == nil {
		fs.rulesetDefs = make(map[string]*RulesetDefinition)
	}
	fs.rulesetDefs[rs_name] = rsdef
}

// For example:
//
// Tell Git to send a `def_param` for all config settings with
// the `otel.trace2.*` namespace.
//
// $ git config --system trace2.configparams "otel.trace2.*"
//
// Tell Git that my workrepo worktree is an instance of "monorepo"
// (regardless what the origin URL or worktree root directory
// names are).
//
// $ cd /path/to/my/workrepo/
// $ git config --local otel.trace2.nickname "monorepo"
//
// Tell Git that my duplicate workrepo worktree is another
// instance of the same "monorepo" (so data from both repos
// can be aggregated in the cloud).
//
// $ cd /path/to/my/workrepo-copy/
// $ git config --local otel.trace2.nickname "monorepo"
//
//
//
// Tell Git that my privaterepo is an instance of "private"
// (or is a member of a group distinct from my other repos).
//
// $ cd /path/to/my/privaterepo
// $ git config --local otel.trace2.nickname "private"
//
// Tell Git that my other worktree should be filtered using
// the "rs:xyz" ruleset (regardless of whether there is a nickname
// defined for the worktree).
//
// $ cd /path/to/my/otherrepo
// $ git config --local otel.trace2.ruleset "rs:xyz"
//
//
// filter.yml
// ==========
// keynames:
//   nickname_key: "otel.trace2.nickname"
//   ruleset_key: "otel.trace2.ruleset"
//
// nicknames:
//   "monorepo": "dl:verbose"
//   "private":  "dl:drop"
//
// rulesets:
//   "rs:status": "./rulesets/rs-status.yml"
//   "rs:xyz":    "./rulesets/rs-xyz.yml"
//
// defaults:
//   ruleset: "dl:summary"
//
//
// rulesets/rs-status.yml
// ======================
// commands:
//   "git:status": "dl:verbose"
//
// defaults:
//   detail: "dl:drop"
//
//
// rulesets/rs-xyz.yml
// ===================
// commands:
//   "git:fetch": "dl:verbose"
//   "git:pull": "dl:verbose"
//   "git:status": "dl:summary"
//
// defaults:
//   detail: "dl:drop"
