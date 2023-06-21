package trace2receiver

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// Each of the "TEST/*" pathnames are a fake placeholder to make
// the `FilterSettings` data structure happy.  We do everything
// in memory here.
var x_fs_path string = "TEST/fs.yml"
var x_rs_path string = "TEST/rs.yml"

var x_qn = QualifiedNames{
	exe:         "c",
	exeVerb:     "c:v",
	exeVerbMode: "c:v#m",
}

// //////////////////////////////////////////////////////////////

var x_fs_empty_yml string = `
`

// If filter settings is empty, we always get the global
// builtin default detail level.
func Test_Empty_FilterSettings(t *testing.T) {
	params := make(map[string]string)

	fs := x_TryLoadFilterSettings(t, x_fs_empty_yml, x_fs_path)

	dl, dl_debug := computeDetailLevel(fs, params, x_qn)

	assert.Equal(t, DetailLevelSummary, dl) // the inherited global default
	assert.Equal(t, "[builtin-default -> dl:summary]", dl_debug)
}

// //////////////////////////////////////////////////////////////

var x_fs_default_yml string = `
defaults:
  ruleset: "dl:verbose"
`

// The filter settings overrides the global builtin default detail
// level.
func Test_Default_FilterSettings(t *testing.T) {
	params := make(map[string]string)

	fs := x_TryLoadFilterSettings(t, x_fs_default_yml, x_fs_path)

	dl, dl_debug := computeDetailLevel(fs, params, x_qn)

	assert.Equal(t, DetailLevelVerbose, dl)
	assert.Equal(t, "[default-ruleset -> dl:verbose]", dl_debug)
}

// //////////////////////////////////////////////////////////////

var x_fs_rsdef0_yml string = `
rulesets:
  # "rs:rsdef0": "TEST/rs.yml" (use addRuleset())

defaults:
  ruleset: "rs:rsdef0"
`

var x_rs_rsdef0_name string = "rs:rsdef0"

var x_rs_rsdef0_yml string = `
defaults:
  detail: "dl:process"
`

// The filter settings overrides the global builtin default detail
// level, but uses a ruleset indirection.  Since the ruleset does
// not define any command mappings, it falls thru to the ruleset default.
func Test_RSDef0_FilterSettings(t *testing.T) {
	params := make(map[string]string)

	fs := x_TryLoadFilterSettings(t, x_fs_rsdef0_yml, x_fs_path)
	x_TryLoadRuleset(t, fs, x_rs_rsdef0_name, x_rs_path, x_rs_rsdef0_yml)

	dl, dl_debug := computeDetailLevel(fs, params, x_qn)

	assert.Equal(t, DetailLevelProcess, dl)
	assert.Equal(t, "[default-ruleset -> rs:rsdef0]/[command -> c:v#m]/[ruleset-default -> dl:process]", dl_debug)
}

// //////////////////////////////////////////////////////////////

var x_rkey string = "otel.trace2.ruleset" // must match ruleset_key in the following

var x_fs_key_yml string = `
keynames:
  ruleset_key: "otel.trace2.ruleset"

rulesets:
  # "rs:rsdef0": "TEST/rs.yml" (use addRuleset())
  # "rs:rsdef1": "TEST/rs.yml" (use addRuleset())

defaults:
  ruleset: "rs:rsdef0"
`

var x_rs_rsdef1_name string = "rs:rsdef1"

var x_rs_rsdef1_yml string = `
defaults:
  detail: "dl:summary"
`

// The filter settings defines a `ruleset_key` as a way to use a Git
// config value to request a specific ruleset.  The filter settings
// defines two rulesets by name.
//
// Verify that lookups default to rsdef0 when no key is provided
// and then that we get rsdef1 when requested.
//
// If the requested ruleset does not exist, fall back to the global
// builtin default detail level.
func Test_RulesetKey_FilterSettings(t *testing.T) {
	params := make(map[string]string)

	fs := x_TryLoadFilterSettings(t, x_fs_key_yml, x_fs_path)
	x_TryLoadRuleset(t, fs, x_rs_rsdef0_name, x_rs_path, x_rs_rsdef0_yml)
	x_TryLoadRuleset(t, fs, x_rs_rsdef1_name, x_rs_path, x_rs_rsdef1_yml)

	dl, dl_debug := computeDetailLevel(fs, params, x_qn)

	assert.Equal(t, dl, DetailLevelProcess)
	assert.Equal(t, dl_debug, "[default-ruleset -> rs:rsdef0]/[command -> c:v#m]/[ruleset-default -> dl:process]")

	params[x_rkey] = x_rs_rsdef1_name // set the Git config key

	dl, dl_debug = computeDetailLevel(fs, params, x_qn)

	assert.Equal(t, DetailLevelSummary, dl)
	assert.Equal(t, "[rskey -> rs:rsdef1]/[command -> c:v#m]/[ruleset-default -> dl:summary]", dl_debug)

	params[x_rkey] += "-bogus" // set the Git config key to an unknown ruleset

	dl, dl_debug = computeDetailLevel(fs, params, x_qn)

	assert.Equal(t, DetailLevelSummary, dl)
	assert.Equal(t, "[rskey -> rs:rsdef1-bogus]/[rs:rsdef1-bogus -> INVALID]/[builtin-default -> dl:summary]", dl_debug)
}

// //////////////////////////////////////////////////////////////

var x_nnkey string = "otel.trace2.nickname"
var x_nn string = "monorepo"

var x_fs_nnkey_yml string = `
keynames:
  nickname_key: "otel.trace2.nickname"

nicknames:
  "monorepo": "rs:rsdef1"

rulesets:
  # "rs:rsdef0": "TEST/rs.yml" (use addRuleset())
  # "rs:rsdef1": "TEST/rs.yml" (use addRuleset())

defaults:
  ruleset: "rs:rsdef0"
`

// The filter settings defines a `nickname_key` as a way to use a Git
// config value to declare that a worktree is an instance of some repo.
// The filter settings defines a table to map nicknames to rulesets.
// And it defines two rulesets.
//
// Verify that lookups default to rsdef0 when no nickname is provided
// and then that we get the rsdef1 ruleset when the nickname is used.
func Test_NicknameKey_FilterSettings(t *testing.T) {
	params := make(map[string]string)

	fs := x_TryLoadFilterSettings(t, x_fs_nnkey_yml, x_fs_path)
	x_TryLoadRuleset(t, fs, x_rs_rsdef0_name, x_rs_path, x_rs_rsdef0_yml)
	x_TryLoadRuleset(t, fs, x_rs_rsdef1_name, x_rs_path, x_rs_rsdef1_yml)

	dl, dl_debug := computeDetailLevel(fs, params, x_qn)

	assert.Equal(t, dl, DetailLevelProcess)
	assert.Equal(t, dl_debug, "[default-ruleset -> rs:rsdef0]/[command -> c:v#m]/[ruleset-default -> dl:process]")

	params[x_nnkey] = x_nn // set the Git config key

	dl, dl_debug = computeDetailLevel(fs, params, x_qn)

	assert.Equal(t, DetailLevelSummary, dl)
	assert.Equal(t, "[nickname -> monorepo]/[monorepo -> rs:rsdef1]/[command -> c:v#m]/[ruleset-default -> dl:summary]", dl_debug)

	params[x_nnkey] += "-bogus" // set the Git config key to an unknown nickname

	dl, dl_debug = computeDetailLevel(fs, params, x_qn)

	assert.Equal(t, DetailLevelProcess, dl)
	assert.Equal(t, "[nickname -> monorepo-bogus]/[monorepo-bogus -> UNKNOWN]/[default-ruleset -> rs:rsdef0]/[command -> c:v#m]/[ruleset-default -> dl:process]", dl_debug)
}

// //////////////////////////////////////////////////////////////

var x_fs_rscmd0_yml string = `
rulesets:
  # "rs:rscmd0": "TEST/rs.yml" (use addRuleset())

defaults:
  ruleset: "rs:rscmd0"
`

var x_rs_rscmd0_name string = "rs:rscmd0"

var x_rs_rscmd0_yml string = `
commands:
  "c:v#m": "dl:drop"
  "c:v":   "dl:summary"
  "c":     "dl:process"

defaults:
  detail: "dl:verbose"
`

// The filter settings overrides the global builtin default detail
// level, but uses a ruleset indirection.  The rscmd0 ruleset defines
// command mappings.  Verify that each command variation gets mapped
// correctly.
func Test_RSCmd0_FilterSettings(t *testing.T) {
	params := make(map[string]string)

	fs := x_TryLoadFilterSettings(t, x_fs_rscmd0_yml, x_fs_path)
	x_TryLoadRuleset(t, fs, x_rs_rscmd0_name, x_rs_path, x_rs_rscmd0_yml)

	var qn1 = QualifiedNames{
		exe:         "c",
		exeVerb:     "c:v",
		exeVerbMode: "c:v#m",
	}

	dl, dl_debug := computeDetailLevel(fs, params, qn1)

	assert.Equal(t, DetailLevelDrop, dl)
	assert.Equal(t, "[default-ruleset -> rs:rscmd0]/[command -> c:v#m]/[c:v#m -> dl:drop]", dl_debug)

	qn1.exeVerbMode = "c:v#ZZ" // change the mode to get verb fallback

	dl, dl_debug = computeDetailLevel(fs, params, qn1)

	assert.Equal(t, DetailLevelSummary, dl)
	assert.Equal(t, "[default-ruleset -> rs:rscmd0]/[command -> c:v#ZZ]/[c:v -> dl:summary]", dl_debug)

	qn1.exeVerb = "c:YY" // change the verb to get exe fallback
	qn1.exeVerbMode = "c:YY#ZZ"

	dl, dl_debug = computeDetailLevel(fs, params, qn1)

	assert.Equal(t, DetailLevelProcess, dl)
	assert.Equal(t, "[default-ruleset -> rs:rscmd0]/[command -> c:YY#ZZ]/[c -> dl:process]", dl_debug)

	qn1.exe = "XX" // change the exe to get ruleset default fallback
	qn1.exeVerb = "XX:YY"
	qn1.exeVerbMode = "XX:YY#ZZ"

	dl, dl_debug = computeDetailLevel(fs, params, qn1)

	assert.Equal(t, DetailLevelVerbose, dl)
	assert.Equal(t, "[default-ruleset -> rs:rscmd0]/[command -> XX:YY#ZZ]/[ruleset-default -> dl:verbose]", dl_debug)
}

// //////////////////////////////////////////////////////////////

func x_TryLoadFilterSettings(t *testing.T, yml string, path string) *FilterSettings {
	fs, err := parseFilterSettingsFromBuffer([]byte(yml), path)
	if err != nil {
		t.Fatalf("parseFilterSettings(%s): %s", path, err.Error())
	}
	return fs
}

func x_TryLoadRuleset(t *testing.T, fs *FilterSettings, name string, path string, yml string) {
	rs, err := parseRulesetFromBuffer([]byte(yml), path)
	if err != nil {
		t.Fatalf("parseRuleset(%s): %s", path, err.Error())
	}

	fs.addRuleset(name, path, rs)
}

// //////////////////////////////////////////////////////////////

func Test_Nil_Nil_FilterSettings(t *testing.T) {

	dl, dl_debug := computeDetailLevel(nil, nil, x_qn)

	assert.Equal(t, DetailLevelSummary, dl)
	assert.Equal(t, "[builtin-default -> dl:summary]", dl_debug)
}

func Test_FSEmpty_Nil_FilterSettings(t *testing.T) {

	fs := x_TryLoadFilterSettings(t, x_fs_empty_yml, x_fs_path)

	dl, dl_debug := computeDetailLevel(fs, nil, x_qn)

	assert.Equal(t, DetailLevelSummary, dl)
	assert.Equal(t, "[builtin-default -> dl:summary]", dl_debug)
}

func Test_FSNNKey_Nil_FilterSettings(t *testing.T) {

	fs := x_TryLoadFilterSettings(t, x_fs_nnkey_yml, x_fs_path)
	x_TryLoadRuleset(t, fs, x_rs_rsdef0_name, x_rs_path, x_rs_rsdef0_yml)
	x_TryLoadRuleset(t, fs, x_rs_rsdef1_name, x_rs_path, x_rs_rsdef1_yml)

	dl, dl_debug := computeDetailLevel(fs, nil, x_qn)

	assert.Equal(t, DetailLevelProcess, dl)
	assert.Equal(t, "[default-ruleset -> rs:rsdef0]/[command -> c:v#m]/[ruleset-default -> dl:process]", dl_debug)
}
