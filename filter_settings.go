package trace2receiver

// FilterSettings describes how we should filter the OTLP output
// that we generate.  It also describes the special keys that we
// look for in the Trace2 event stream to help us decide how to
// filter data for a particular command.
type FilterSettings struct {
	NamespaceKeys FSKeyNames    `mapstructure:"keynames"`
	NicknameMap   FSNicknameMap `mapstructure:"nicknames"`
	RulesetMap    FSRulesetMap  `mapstructure:"rulesets"`
	Defaults      FSDefaults    `mapstructure:"defaults"`

	// The set of custom rulesets defined in YML are each parsed
	// and loaded into definitions so that we can use them.
	rulesetDefs map[string]*RSDefinition
}

// FSKeyNames defines the names of the Git config settings that
// will be used in `def_param` events to send repository/worktree
// data to us.  This lets a site have their own namespace for
// these keys.  Some of these keys will also be sent to the cloud.
type FSKeyNames struct {

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

// FSDefaults defines default filtering values.
type FSDefaults struct {

	// Ruleset defines the default ruleset or detail level to be
	// used when we receive data from a repo/worktree that does
	// not explicitly name one or does not have a nickname mapping.
	//
	// If not set, we default to the absolute default.
	RulesetName string `mapstructure:"ruleset"`
}

// FSNicknameMap is used to map a repo nickname to the name of the
// ruleset or detail-level that should be used.
//
// This table is optional.
type FSNicknameMap map[string]string

// FSRulesetMap is used to map a custom ruleset name to the pathname
// of the associated YML file.  This form is used when parsing the
// filter settings YML file.  We use this to create the real ruleset
// table (possibly with lazy loading).
type FSRulesetMap map[string]string

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
// $ git config --local otel.trace2.repoid "monorepo"
//
// Tell Git that my duplicate workrepo worktree is another
// instance of the same "monorepo" (so data from both repos
// can be aggregated in the cloud).
//
// $ cd /path/to/my/workrepo-copy/
// $ git config --local otel.trace2.repoid "monorepo"
//
//
//
// Tell Git that my privaterepo is an instance of "private"
// (or is a member of a group distinct from my other repos).
//
// $ cd /path/to/my/privaterepo
// $ git config --local otel.trace2.repoid "private"
//
// Tell Git that my other worktree should be filtered using
// "my-custom-ruleset" (regardless of whether there is a nickname
// defined for it).
//
// $ cd /path/to/my/otherrepo
// $ git config --local otel.trace2.ruleset "my-custom-ruleset"
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
