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
	Keynames        FilterKeynames   `mapstructure:"keynames"`
	Nicknames       FilterNicknames  `mapstructure:"nicknames"`
	Rulesets        FilterRulesets   `mapstructure:"rulesets"`
	Defaults        FilterDefaults   `mapstructure:"defaults"`
	ImportantEvents []ImportantEventRule `mapstructure:"important_events"`

	// The set of custom rulesets defined in YML are each parsed
	// and loaded into definitions so that we can use them.
	rulesetDefs map[string]*RulesetDefinition
}

// ImportantEventRule defines a rule for promoting values from data events
// that match a specific (category, key prefix) pair into the process
// summary, regardless of the active detail level. This lets operators
// guarantee that certain data event values are always captured and
// surfaced in the OTEL process span even when verbose telemetry is
// disabled. Multiple matching values are collected into an array.
type ImportantEventRule struct {
	// Category is the data event category to match (exact match)
	Category string `mapstructure:"category"`

	// KeyPrefix is the string prefix to match at the beginning of
	// the data event's key field
	KeyPrefix string `mapstructure:"key_prefix"`

	// FieldName is the name of the field in the summary object
	// where matched values will be stored (always as an array)
	FieldName string `mapstructure:"field_name"`
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

	fieldNames := make(map[string]bool)
	for i, rule := range fs.ImportantEvents {
		if len(rule.Category) == 0 {
			return nil, fmt.Errorf("important_events[%d]: category cannot be empty", i)
		}
		if len(rule.KeyPrefix) == 0 {
			return nil, fmt.Errorf("important_events[%d]: key_prefix cannot be empty", i)
		}
		if len(rule.FieldName) == 0 {
			return nil, fmt.Errorf("important_events[%d]: field_name cannot be empty", i)
		}
		if fieldNames[rule.FieldName] {
			return nil, fmt.Errorf("important_events[%d]: duplicate field_name '%s'", i, rule.FieldName)
		}
		fieldNames[rule.FieldName] = true
	}

	return fs, nil
}

// apply__important_events checks if a data event matches any configured
// important_events rules and appends the event's value to the
// importantEvents map if a match is found. Matching events are captured
// regardless of nesting level or detail level.
func apply__important_events(tr2 *trace2Dataset, category string, key string, value interface{}) {
	if tr2.process.importantEvents == nil {
		return
	}

	if tr2.rcvr_base == nil || tr2.rcvr_base.RcvrConfig == nil {
		return
	}

	fs := tr2.rcvr_base.RcvrConfig.filterSettings
	if fs == nil {
		return
	}

	for _, rule := range fs.ImportantEvents {
		if category == rule.Category && strings.HasPrefix(key, rule.KeyPrefix) {
			tr2.process.importantEvents[rule.FieldName] = append(
				tr2.process.importantEvents[rule.FieldName], value)
		}
	}
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
