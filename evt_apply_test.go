package trace2receiver

// Tests in this file are concerned with whether the Trace2 JSON
// event messages are correctly converted and populated in a
// `trace2Dataset`.
//
// We do not worry about whether JSON values are missing or of the
// wrong type (since that has been tested at lower levels).
//
// Our concern here is the correct accumulation of multiple JSON
// events into the dataset.  This dataset will be the basis for
// the trace/span set (tested at a higer level).

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// Well-known values for mostly constant fields in the data stream.

var x_sid string = "20230130T174853.894364Z-H0f5a2227-P000048b6"
var x_file string = "foo.c"
var x_ln int = 42
var x_main string = "main"
var x_version_evt string = "3"
var x_version_exe string = "1.2.3.4"
var x_exit_code int64 = 13
var x_error_1_msg string = "an error message 1"
var x_error_2_msg string = "an error message 2"
var x_error_1_fmt string = "an %s message 1"
var x_error_2_fmt string = "an %s message 2"
var x_cmd_path string = "/a/cmd/path"
var x_cmd_name string = "xyz"
var x_cmd_hierarchy string = "abc/def"
var x_cmd_mode string = "x-mode"
var x_alias_key string = "alias-key"
var x_repo_1_worktree string = "/a/b/c/repo-1"
var x_repo_3_worktree string = "/a/b/c/repo-3"

var x_param_system_foo string = "system-foo"
var x_param_global_foo string = "global-foo"
var x_param_local_foo string = "local-foo"

var x_param_system_bar string = "system-bar"
var x_param_global_bar string = "global-bar"
var x_param_local_bar string = "local-bar"

var x_time_zero time.Time = time.Now().UTC()
var x_time_now time.Time = x_time_zero

// Are the two times very close?  We may lose some precision when
// we round-trip time to a JSON string and convert it back to a
// time value.  For example, we want to ask if the dataset recorded
// the version.start time as we set it in the "version" event, but
// that could be fuzzy if one of the platforms handles nanoseconds
// differently.  So consider them equal if they are within an epsilon.
func times_are_within_epsilon(t1 time.Time, t2 time.Time) bool {
	d := t1.Sub(t2)
	return d.Abs().Microseconds() < 5
}

// Build snippets of JSON so that we can send raw JSON into the parser.

func x_make_common(event_name string, thread_name string) string {
	s := fmt.Sprintf(`"event":"%s"`, event_name)
	s += fmt.Sprintf(`,"sid":"%s"`, x_sid)
	s += fmt.Sprintf(`,"file":"%s"`, x_file)
	s += fmt.Sprintf(`,"line":%d`, x_ln)
	s += fmt.Sprintf(`,"thread":"%s"`, thread_name)

	s += fmt.Sprintf(`,"time":"%s"`, x_time_now.Format("2006-01-02T15:04:05.999999Z"))
	// auto-advance clock so that it looks like time is advancing in each JSON event.
	x_time_now = x_time_now.Add(time.Second * 1)

	return s
}
func x_make_t_abs() float64 {
	diff := x_time_now.Sub(x_time_zero)
	us := diff.Microseconds()
	f := (float64)(us) / 1000000.0

	return f
}
func x_make_version() string {
	// The `version` event is always the first event, so let's use that
	// to reset `x_time_now` at the beginning of each unit test.
	x_time_now = x_time_zero

	return fmt.Sprintf(`{%s,"evt":"%s","exe":"%s"}`,
		x_make_common(
			"version",
			x_main),
		x_version_evt,
		x_version_exe)
}
func x_make_start_av(av string) string {
	return fmt.Sprintf(`{%s,"t_abs":%.6f,"argv":%s}`,
		x_make_common(
			"start",
			x_main),
		x_make_t_abs(),
		av)
}
func x_make_start_argv3(a0 string, a1 string, a2 string) string {
	return x_make_start_av(
		fmt.Sprintf(`["%s","%s","%s"]`, a0, a1, a2))
}
func x_make_start_argv1(a0 string) string {
	return x_make_start_av(
		fmt.Sprintf(`["%s"]`, a0))
}
func x_make_start() string {
	return x_make_start_argv3("cmdarg0", "cmdarg1", "cmdarg2")
}

func x_make_atexit() string {
	return fmt.Sprintf(`{%s,"t_abs":%.6f,"code":%d}`,
		x_make_common(
			"atexit",
			x_main),
		x_make_t_abs(),
		x_exit_code)
}
func x_make_error(m string, f string) string {
	return fmt.Sprintf(`{%s,"msg":"%s","fmt":"%s"}`,
		x_make_common(
			"error",
			x_main),
		m,
		f)
}
func x_make_cmd_path() string {
	return fmt.Sprintf(`{%s,"path":"%s"}`,
		x_make_common(
			"cmd_path",
			x_main),
		x_cmd_path)
}
func x_make_cmd_ancestry() string {
	return fmt.Sprintf(`{%s,"ancestry":%s}`,
		x_make_common(
			"cmd_ancestry",
			x_main),
		`["a0","a1","a2"]`)
}
func x_make_cmd_name_nh(n string, h string) string {
	return fmt.Sprintf(`{%s,"name":"%s","hierarchy":"%s"}`,
		x_make_common(
			"cmd_name",
			x_main),
		n,
		h)
}
func x_make_cmd_name() string {
	return x_make_cmd_name_nh(
		x_cmd_name,
		x_cmd_hierarchy)
}
func x_make_cmd_mode() string {
	return fmt.Sprintf(`{%s,"name":"%s"}`,
		x_make_common(
			"cmd_mode",
			x_main),
		x_cmd_mode)
}
func x_make_alias() string {
	return fmt.Sprintf(`{%s,"alias":"%s","argv":%s}`,
		x_make_common(
			"alias",
			x_main),
		x_alias_key,
		`["v0","v1"]`)
}
func x_make_def_repo(id int64, wt string) string {
	return fmt.Sprintf(`{%s,"repo":%d,"worktree":"%s"}`,
		x_make_common(
			"def_repo",
			x_main),
		id,
		wt)
}
func x_make_def_param(scope string, param string, value string) string {
	return fmt.Sprintf(`{%s,"scope":"%s","param":"%s","value":"%s"}`,
		x_make_common(
			"def_param",
			x_main),
		scope,
		param,
		value)
}
func x_make_child_start(id int64, class string, a0 string, a1 string) string {
	return fmt.Sprintf(`{%s,"child_id":%d,"child_class":"%s","use_shell":%s,"argv":%s}`,
		x_make_common(
			"child_start",
			x_main),
		id,
		class,
		"false", // we don't care about "use_shell", but it is required in the format
		fmt.Sprintf(`["%s","%s"]`, a0, a1))
}
func x_make_hook_child_start(id int64, class string, hook string, a0 string, a1 string) string {
	return fmt.Sprintf(`{%s,"child_id":%d,"child_class":"%s","hook_name":"%s","use_shell":%s,"argv":%s}`,
		x_make_common(
			"child_start",
			x_main),
		id,
		class,
		hook,
		"false", // we don't care about "use_shell", but it is required in the format
		fmt.Sprintf(`["%s","%s"]`, a0, a1))
}
func x_make_child_exit(id int64, pid int64, code int64) string {
	return fmt.Sprintf(`{%s,"child_id":%d,"pid":%d,"code":%d,"t_rel":%.6f}`,
		x_make_common(
			"child_exit",
			x_main),
		id,
		pid,
		code,
		1.0)
}
func x_make_exec(id int64, exe string, a0 string, a1 string) string {
	return fmt.Sprintf(`{%s,"exec_id":%d,"exe":"%s","argv":%s}`,
		x_make_common(
			"exec",
			x_main),
		id,
		exe,
		fmt.Sprintf(`["%s","%s"]`, a0, a1))
}
func x_make_region_enter(thread_name string, nesting int64, category string, label string, msg string) string {
	return fmt.Sprintf(`{%s,"nesting":%d,"category":"%s","label":"%s","msg":"%s"}`,
		x_make_common(
			"region_enter",
			thread_name),
		nesting,
		category,
		label,
		msg)
}
func x_make_region_leave(thread_name string, nesting int64, category string, label string, msg string) string {
	return fmt.Sprintf(`{%s,"nesting":%d,"category":"%s","label":"%s","msg":"%s","t_rel":%.6f}`,
		x_make_common(
			"region_leave",
			thread_name),
		nesting,
		category,
		label,
		msg,
		1.0)
}
func x_make_data_string(thread_name string, nesting int64, category string, key string, value string) string {
	return fmt.Sprintf(`{%s,"nesting":%d,"category":"%s","key":"%s","value":"%s","t_abs":%.6f,"t_rel":%.6f}`,
		x_make_common(
			"data",
			thread_name),
		nesting,
		category,
		key,
		value,
		x_make_t_abs(),
		1.0)
}
func x_make_data_intmax(thread_name string, nesting int64, category string, key string, value int64) string {
	return fmt.Sprintf(`{%s,"nesting":%d,"category":"%s","key":"%s","value":%d,"t_abs":%.6f,"t_rel":%.6f}`,
		x_make_common(
			"data",
			thread_name),
		nesting,
		category,
		key,
		value,
		x_make_t_abs(),
		1.0)
}
func x_make_data_json(thread_name string, nesting int64, category string, key string, value string) string {
	return fmt.Sprintf(`{%s,"nesting":%d,"category":"%s","key":"%s","value":%s,"t_abs":%.6f,"t_rel":%.6f}`,
		x_make_common(
			"data_json",
			thread_name),
		nesting,
		category,
		key,
		value,
		x_make_t_abs(),
		1.0)
}
func x_make_timer(category string, name string, intervals int64, t_total float64, t_min float64, t_max float64) string {
	return fmt.Sprintf(`{%s,"category":"%s","name":"%s","intervals":%d,"t_total":%.6f,"t_min":%.6f,"t_max":%.6f}`,
		x_make_common(
			"timer",
			x_main),
		category,
		name,
		intervals,
		t_total,
		t_min,
		t_max)
}
func x_make_counter(category string, name string, count int64) string {
	return fmt.Sprintf(`{%s,"category":"%s","name":"%s","count":%d}`,
		x_make_common(
			"counter",
			x_main),
		category,
		name,
		count)
}
func x_make_thread_start(thread_name string) string {
	return fmt.Sprintf(`{%s}`,
		x_make_common(
			"thread_start",
			thread_name))
}
func x_make_thread_exit(thread_name string) string {
	return fmt.Sprintf(`{%s,"t_rel":%.6f}`,
		x_make_common(
			"thread_exit",
			thread_name),
		1.0)
}

// Verify most of the common events that set process-level fields.
func Test_Dataset_Basic(t *testing.T) {

	var events []string = []string{
		x_make_version(),
		x_make_start(),
		x_make_error(x_error_1_msg, x_error_1_fmt),
		x_make_cmd_path(),
		x_make_cmd_ancestry(),
		x_make_cmd_name(),
		x_make_cmd_mode(),
		x_make_alias(),

		x_make_def_repo(1, x_repo_1_worktree),
		x_make_def_repo(3, x_repo_3_worktree), // non-contiguous

		x_make_atexit(), // Should be last
	}

	tr2, sufficient, _ := load_test_dataset(t, events)
	assert.True(t, sufficient, "have sufficient data")

	assert.Equal(t, tr2.trace2SID, x_sid)

	assert.Equal(t, tr2.process.evtVersion, x_version_evt)
	assert.Equal(t, tr2.process.exeVersion, x_version_exe)
	assert.True(t,
		times_are_within_epsilon(tr2.process.mainThread.lifetime.startTime,
			x_time_zero))

	assert.Equal(t, len(tr2.process.cmdArgv), 3)
	assert.Equal(t, tr2.process.cmdArgv[0], "cmdarg0")
	assert.Equal(t, tr2.process.cmdArgv[1], "cmdarg1")
	assert.Equal(t, tr2.process.cmdArgv[2], "cmdarg2")

	assert.Equal(t, tr2.process.exeErrorMsg, x_error_1_msg)
	assert.Equal(t, tr2.process.exeErrorFmt, x_error_1_fmt)

	// ignore cmd_path

	assert.Equal(t, len(tr2.process.cmdAncestry), 3)
	assert.Equal(t, tr2.process.cmdAncestry[0], "a0")
	assert.Equal(t, tr2.process.cmdAncestry[1], "a1")
	assert.Equal(t, tr2.process.cmdAncestry[2], "a2")

	assert.Equal(t, tr2.process.cmdVerb, x_cmd_name)
	assert.Equal(t, tr2.process.cmdHierarchy, x_cmd_hierarchy)

	assert.Equal(t, tr2.process.cmdMode, x_cmd_mode)

	assert.Equal(t, tr2.process.cmdAliasKey, x_alias_key)
	assert.Equal(t, len(tr2.process.cmdAliasValue), 2)

	assert.Equal(t, tr2.process.cmdAliasValue[0], "v0")
	assert.Equal(t, tr2.process.cmdAliasValue[1], "v1")

	// repoSet is a Map/Set with integer keys, rather than an Array.
	assert.Equal(t, len(tr2.process.repoSet), 2)
	assert.Equal(t, tr2.process.repoSet[1], x_repo_1_worktree)
	assert.Equal(t, tr2.process.repoSet[3], x_repo_3_worktree)

	assert.Equal(t, tr2.process.exeExitCode, x_exit_code)
	assert.Less(t, tr2.process.mainThread.lifetime.startTime, tr2.process.mainThread.lifetime.endTime)
}

// Verify that we don't strip unknown suffixes off of argv.
// GCM spawns `<something>.UI` of class `ui_helper` to prompt the user, for example.
// It might be OK to strip off an `.exe`, but not other suffixes.  Note that suffix
// stripping happens at the trace2dataset level and not at this evt_apply level.
func Test_Dataset_Argv0_with_Dot(t *testing.T) {

	var events []string = []string{
		x_make_version(),
		x_make_start_argv3("cmdarg0.foobar", "abc", "def"),
		x_make_cmd_path(),

		x_make_atexit(), // Should be last
	}

	tr2, sufficient, _ := load_test_dataset(t, events)
	assert.True(t, sufficient, "have sufficient data")

	assert.Equal(t, tr2.trace2SID, x_sid)

	assert.Equal(t, tr2.process.evtVersion, x_version_evt)
	assert.Equal(t, tr2.process.exeVersion, x_version_exe)
	assert.True(t,
		times_are_within_epsilon(tr2.process.mainThread.lifetime.startTime,
			x_time_zero))

	assert.Equal(t, len(tr2.process.cmdArgv), 3)
	assert.Equal(t, tr2.process.cmdArgv[0], "cmdarg0.foobar")
	assert.Equal(t, tr2.process.cmdArgv[1], "abc")
	assert.Equal(t, tr2.process.cmdArgv[2], "def")

	assert.Equal(t, tr2.process.exeExitCode, x_exit_code)
	assert.Less(t, tr2.process.mainThread.lifetime.startTime, tr2.process.mainThread.lifetime.endTime)
}

// Git can emit multiple "error" events, but only report the
// first.  (Because I'm not sure it is worth building an array
// of them.)
func Test_Dataset_Error(t *testing.T) {

	var events []string = []string{
		x_make_version(),
		x_make_start(),

		x_make_error(x_error_1_msg, x_error_1_fmt),
		x_make_error(x_error_2_msg, x_error_2_fmt),

		x_make_atexit(), // Should be last
	}

	tr2, sufficient, _ := load_test_dataset(t, events)
	assert.True(t, sufficient, "have sufficient data")

	// Only the first error message is reported.
	assert.Equal(t, tr2.process.exeErrorMsg, x_error_1_msg)
	assert.Equal(t, tr2.process.exeErrorFmt, x_error_1_fmt)
}

// Verify def_param are captured (without priority concerns).
func Test_Dataset_DefParam_Easy(t *testing.T) {

	var events []string = []string{
		x_make_version(),
		x_make_start(),

		x_make_def_param("system", "foo", x_param_system_foo),
		x_make_def_param("global", "bar", x_param_global_bar),

		x_make_atexit(), // Should be last
	}

	tr2, sufficient, _ := load_test_dataset(t, events)
	assert.True(t, sufficient, "have sufficient data")

	assert.NotNil(t, tr2.process.paramSetValues["foo"])
	assert.Equal(t, tr2.process.paramSetValues["foo"], x_param_system_foo)

	assert.NotNil(t, tr2.process.paramSetValues["bar"])
	assert.Equal(t, tr2.process.paramSetValues["bar"], x_param_global_bar)
}

// Verify that we remember the highest priority value when
// multiple def_param events are sent for the same parameter.
func Test_Dataset_DefParam_Scoped(t *testing.T) {

	var events []string = []string{
		x_make_version(),
		x_make_start(),

		// regardless of the order of the events, the local value should win.
		x_make_def_param("system", "foo", x_param_system_foo),
		x_make_def_param("global", "foo", x_param_global_foo),
		x_make_def_param("local", "foo", x_param_local_foo),

		x_make_def_param("local", "bar", x_param_local_bar),
		x_make_def_param("global", "bar", x_param_global_bar),
		x_make_def_param("system", "bar", x_param_system_bar),

		x_make_atexit(), // Should be last
	}

	tr2, sufficient, _ := load_test_dataset(t, events)
	assert.True(t, sufficient, "have sufficient data")

	assert.Equal(t, len(tr2.process.paramSetValues), 2)

	assert.NotNil(t, tr2.process.paramSetValues["foo"])
	assert.Equal(t, tr2.process.paramSetValues["foo"], x_param_local_foo)

	assert.NotNil(t, tr2.process.paramSetValues["bar"])
	assert.Equal(t, tr2.process.paramSetValues["bar"], x_param_local_bar)
}

// Verify that when multiple def_param events are sent for the same parameter
// with the SAME scope, the last one wins (matching Git's behavior).
// This happens when using includeIf to include config files.
func Test_Dataset_DefParam_SameScope_LastWins(t *testing.T) {

	var events []string = []string{
		x_make_version(),
		x_make_start(),

		// Multiple values for "ruleset" with the same scope (local).
		// The last one should win, matching git's "last one wins" behavior.
		x_make_def_param("local", "ruleset", "dl:drop"),
		x_make_def_param("local", "ruleset", "dl:verbose"),

		// Also test with global scope
		x_make_def_param("global", "nickname", "first"),
		x_make_def_param("global", "nickname", "second"),
		x_make_def_param("global", "nickname", "last"),

		x_make_atexit(), // Should be last
	}

	tr2, sufficient, _ := load_test_dataset(t, events)
	assert.True(t, sufficient, "have sufficient data")

	assert.Equal(t, len(tr2.process.paramSetValues), 2)

	// For same-priority configs, the LAST one should win (not first)
	assert.NotNil(t, tr2.process.paramSetValues["ruleset"])
	assert.Equal(t, tr2.process.paramSetValues["ruleset"], "dl:verbose")

	assert.NotNil(t, tr2.process.paramSetValues["nickname"])
	assert.Equal(t, tr2.process.paramSetValues["nickname"], "last")
}

func Test_Dataset_ChildProcesses(t *testing.T) {

	var events []string = []string{
		x_make_version(),
		x_make_start(),

		x_make_child_start(0, "class-0", "aa0", "bb0"),
		x_make_child_start(3, "class-3", "aa3", "bb3"), // non-contiguous ids
		x_make_hook_child_start(5, "hook", "my-hook", "hh50", "hh51"),

		x_make_child_exit(0, 123, 0),
		x_make_child_exit(3, 456, 33),
		x_make_child_exit(5, 789, 55),

		x_make_atexit(), // Should be last
	}

	tr2, sufficient, _ := load_test_dataset(t, events)
	assert.True(t, sufficient, "have sufficient data")

	assert.Equal(t, len(tr2.children), 3)

	assert.NotNil(t, tr2.children[0])
	assert.Equal(t, tr2.children[0].class, "class-0")
	assert.Equal(t, tr2.children[0].argv[0], "aa0")
	assert.Equal(t, tr2.children[0].pid, int64(123))
	assert.Equal(t, tr2.children[0].exitcode, int64(0))
	assert.Equal(t, tr2.children[0].lifetime.displayName, "child(class:class-0)")
	assert.Less(t, tr2.children[0].lifetime.startTime, tr2.children[0].lifetime.endTime)
	assert.Equal(t, tr2.children[0].lifetime.parentSpanID, tr2.process.mainThread.lifetime.selfSpanID)

	assert.NotNil(t, tr2.children[3])
	assert.Equal(t, tr2.children[3].class, "class-3")
	assert.Equal(t, tr2.children[3].argv[0], "aa3")
	assert.Equal(t, tr2.children[3].pid, int64(456))
	assert.Equal(t, tr2.children[3].exitcode, int64(33))
	assert.Equal(t, tr2.children[3].lifetime.displayName, "child(class:class-3)")
	assert.Less(t, tr2.children[3].lifetime.startTime, tr2.children[3].lifetime.endTime)
	assert.Equal(t, tr2.children[3].lifetime.parentSpanID, tr2.process.mainThread.lifetime.selfSpanID)

	assert.NotNil(t, tr2.children[5])
	assert.Equal(t, tr2.children[5].class, "hook")
	assert.Equal(t, tr2.children[5].argv[0], "hh50")
	assert.Equal(t, tr2.children[5].pid, int64(789))
	assert.Equal(t, tr2.children[5].exitcode, int64(55))
	assert.Equal(t, tr2.children[5].hookname, "my-hook")
	assert.Equal(t, tr2.children[5].lifetime.displayName, "child(hook:my-hook)")
	assert.Less(t, tr2.children[5].lifetime.startTime, tr2.children[0].lifetime.endTime)
	assert.Equal(t, tr2.children[5].lifetime.parentSpanID, tr2.process.mainThread.lifetime.selfSpanID)

	assert.NotEqual(t, tr2.children[3].lifetime.selfSpanID, tr2.children[0].lifetime.selfSpanID)

	// TODO Consider testing other child-classes and the display name construction.
	// Especially "cred".
}
func Test_Dataset_Regions_Main(t *testing.T) {

	var events []string = []string{
		x_make_version(),
		x_make_start(),

		x_make_region_enter(x_main, 1, "cat", "l1", "m1"),
		x_make_region_enter(x_main, 2, "cat", "l2", "m2"),
		x_make_region_enter(x_main, 3, "cat", "l3", "m3"),

		x_make_region_leave(x_main, 3, "cat", "l3", "m3"),
		x_make_region_leave(x_main, 2, "cat", "l2", "m2"),
		x_make_region_leave(x_main, 1, "cat", "l1", "m1"),

		x_make_atexit(), // Should be last
	}

	tr2, sufficient, _ := load_test_dataset(t, events)
	assert.True(t, sufficient, "have sufficient data")

	assert.Equal(t, len(tr2.process.mainThread.regionStack), 0)
	assert.Equal(t, len(tr2.completedRegions), 3)

	// Items were popped off the stack and appended to the completed
	// set, so {3,cat,l3,m3} should be first.
	r_0 := tr2.completedRegions[0]
	assert.Equal(t, r_0.nestingLevel, int64(3))
	assert.Equal(t, r_0.message, "m3")
	assert.Less(t, r_0.lifetime.startTime, r_0.lifetime.endTime)
	// We don't actually store the original category and label in
	// the region.  We only use it to make a pretty display name.
	assert.Equal(t, r_0.lifetime.displayName, "region(cat,l3)")

	r_1 := tr2.completedRegions[1]
	assert.Equal(t, r_1.nestingLevel, int64(2))
	assert.Equal(t, r_1.message, "m2")
	assert.Less(t, r_1.lifetime.startTime, r_1.lifetime.endTime)
	assert.Equal(t, r_1.lifetime.displayName, "region(cat,l2)")

	r_2 := tr2.completedRegions[2]
	assert.Equal(t, r_2.nestingLevel, int64(1))
	assert.Equal(t, r_2.message, "m1")
	assert.Less(t, r_2.lifetime.startTime, r_2.lifetime.endTime)
	assert.Equal(t, r_2.lifetime.displayName, "region(cat,l1)")

	// r_2 is the {1,cat,l1} top-level region -- it's parent span is the thread
	assert.Equal(t, r_2.lifetime.parentSpanID, tr2.process.mainThread.lifetime.selfSpanID)
	// r_1's parent span is r_2.
	assert.Equal(t, r_1.lifetime.parentSpanID, r_2.lifetime.selfSpanID)
	assert.Equal(t, r_0.lifetime.parentSpanID, r_1.lifetime.selfSpanID)
}

func Test_Dataset_Data_ProcessLevel(t *testing.T) {

	var events []string = []string{
		x_make_version(),
		x_make_start(),

		x_make_data_string(x_main, 1, "cat1", "key11", "val11"),
		x_make_data_intmax(x_main, 1, "cat1", "key12", 12),
		x_make_data_string(x_main, 1, "cat2", "key21", "val21"),
		x_make_data_intmax(x_main, 1, "cat2", "key22", 22),
		x_make_data_json(x_main, 1, "cat3", "key33", `{"x0":"y0","x1":"y1","x2":[1,2,"aaa"]}`),

		x_make_atexit(), // Should be last
	}

	tr2, sufficient, _ := load_test_dataset(t, events)
	assert.True(t, sufficient, "have sufficient data")

	assert.Equal(t, len(tr2.process.dataValues), 3) // [cat1, cat2, cat3]
	assert.NotNil(t, tr2.process.dataValues["cat1"])
	assert.NotNil(t, tr2.process.dataValues["cat2"])
	assert.NotNil(t, tr2.process.dataValues["cat3"])
	assert.Nil(t, tr2.process.dataValues["unk"])

	// All "data" and "data_json" events are combined in a single
	// data[<category>][<key>] map with generic typed values.

	assert.NotNil(t, tr2.process.dataValues["cat1"]["key11"])
	assert.IsType(t, tr2.process.dataValues["cat1"]["key11"], "val11")
	assert.Equal(t, tr2.process.dataValues["cat1"]["key11"], "val11")

	assert.NotNil(t, tr2.process.dataValues["cat1"]["key12"])
	assert.IsType(t, tr2.process.dataValues["cat1"]["key12"], int64(12))
	assert.Equal(t, tr2.process.dataValues["cat1"]["key12"], int64(12))

	assert.NotNil(t, tr2.process.dataValues["cat2"]["key21"])
	assert.IsType(t, tr2.process.dataValues["cat2"]["key21"], "val21")
	assert.Equal(t, tr2.process.dataValues["cat2"]["key21"], "val21")

	assert.NotNil(t, tr2.process.dataValues["cat2"]["key22"])
	assert.IsType(t, tr2.process.dataValues["cat2"]["key22"], int64(22))
	assert.Equal(t, tr2.process.dataValues["cat2"]["key22"], int64(22))

	// The JSON data is dynamically added under the map value, so we can walk into it.

	assert.NotNil(t, tr2.process.dataValues["cat3"]["key33"])
	jv := tr2.process.dataValues["cat3"]["key33"]
	switch jv := jv.(type) {
	case map[string]interface{}:
		assert.NotNil(t, jv["x0"])
		assert.Equal(t, jv["x0"], "y0")

		assert.NotNil(t, jv["x1"])
		assert.Equal(t, jv["x1"], "y1")

		assert.NotNil(t, jv["x2"])
		switch x2 := jv["x2"].(type) { // json array [1, 2] comes back as floats
		case []interface{}:
			assert.Equal(t, len(x2), 3) // [1, 2, "aaa"]

			x2_0 := x2[0]
			switch x2_0 := x2_0.(type) {
			case float64:
				assert.Equal(t, int(x2_0), 1)
			default:
				t.Fatalf("jv.x2[0] is wrong type: '%T'", x2_0)
			}

			x2_2 := x2[2]
			switch x2_2 := x2_2.(type) {
			case string:
				assert.Equal(t, x2_2, "aaa")
			default:
				t.Fatalf("jv.x2[2] is wrong")
			}

		default:
			t.Fatalf("jv.x2 is wrong type: '%T'", x2)
		}
	default:
		t.Fatalf("jv is wrong type: '%T", jv)
	}
}

func Test_Dataset_Timers_Main(t *testing.T) {

	var events []string = []string{
		x_make_version(),
		x_make_start(),

		x_make_timer("cat", "tmr-1", 5, 4.0, 1.0, 2.0),
		x_make_timer("cat", "tmr-2", 8, 8.0, 1.0, 2.0),

		x_make_atexit(), // Should be last
	}

	tr2, sufficient, _ := load_test_dataset(t, events)
	assert.True(t, sufficient, "have sufficient data")

	assert.NotNil(t, tr2.process.timers)
	assert.Equal(t, len(tr2.process.timers), 1) // ["cat"]
	assert.NotNil(t, tr2.process.timers["cat"])

	cv := tr2.process.timers["cat"]
	assert.Equal(t, len(cv), 2) // ["tmr-1", "tmr-2"]
	assert.NotNil(t, cv["tmr-1"])
	assert.NotNil(t, cv["tmr-2"])
	_, ok := cv["unk"]
	assert.False(t, ok)

	sw1, ok := cv["tmr-1"]
	assert.True(t, ok)
	assert.Equal(t, sw1.Intervals, int64(5))
	assert.True(t, float_is_near(sw1.Total_sec, 4.0))
	assert.True(t, float_is_near(sw1.Min_sec, 1.0))
	assert.True(t, float_is_near(sw1.Max_sec, 2.0))
}

func Test_Dataset_Counters_Main(t *testing.T) {

	var events []string = []string{
		x_make_version(),
		x_make_start(),

		x_make_counter("cat", "ctr-1", 5),
		x_make_counter("cat", "ctr-2", 8),

		x_make_atexit(), // Should be last
	}

	tr2, sufficient, _ := load_test_dataset(t, events)
	assert.True(t, sufficient, "have sufficient data")

	assert.NotNil(t, tr2.process.counters)
	assert.Equal(t, len(tr2.process.counters), 1) // ["cat"]
	assert.NotNil(t, tr2.process.counters["cat"])

	cv := tr2.process.counters["cat"]
	assert.Equal(t, len(cv), 2) // ["ctr-1", "ctr-2"]
	assert.NotNil(t, cv["ctr-1"])
	assert.NotNil(t, cv["ctr-2"])
	_, ok := cv["unk"]
	assert.False(t, ok)

	ctr1, ok := cv["ctr-1"]
	assert.True(t, ok)
	assert.Equal(t, ctr1, int64(5))
}

func Test_Dataset_Threads(t *testing.T) {
	var events []string = []string{
		x_make_version(),
		x_make_start(),

		x_make_thread_start("th01"),
		x_make_thread_start("th02"),

		x_make_thread_exit("th01"),
		x_make_thread_exit("th02"),

		x_make_atexit(), // Should be last
	}

	tr2, sufficient, _ := load_test_dataset(t, events)
	assert.True(t, sufficient, "have sufficient data")

	assert.Equal(t, len(tr2.threads), 2)

	th01, ok := tr2.threads["th01"]
	assert.True(t, ok)
	assert.Equal(t, th01.lifetime.displayName, "th01")
	assert.Less(t, th01.lifetime.startTime, th01.lifetime.endTime)
	assert.Equal(t, th01.lifetime.parentSpanID, tr2.process.mainThread.lifetime.selfSpanID)

	th02, ok := tr2.threads["th02"]
	assert.True(t, ok)
	assert.Equal(t, th02.lifetime.displayName, "th02")
	assert.Less(t, th02.lifetime.startTime, th02.lifetime.endTime)
	assert.Equal(t, th02.lifetime.parentSpanID, tr2.process.mainThread.lifetime.selfSpanID)
}

func Test_Dataset_ThreadRegions(t *testing.T) {
	var events []string = []string{
		x_make_version(),
		x_make_start(),

		x_make_thread_start("th01"),
		x_make_thread_start("th02"),

		x_make_region_enter("th01", 1, "cat", "l1", "m1"),
		x_make_region_enter("th01", 2, "cat", "l2", "m2"),
		x_make_region_enter("th01", 3, "cat", "l3", "m3"),

		x_make_region_leave("th01", 3, "cat", "l3", "m3"),
		x_make_region_leave("th01", 2, "cat", "l2", "m2"),
		x_make_region_leave("th01", 1, "cat", "l1", "m1"),

		x_make_thread_exit("th01"),
		x_make_thread_exit("th02"),

		x_make_atexit(), // Should be last
	}

	tr2, sufficient, _ := load_test_dataset(t, events)
	assert.True(t, sufficient, "have sufficient data")

	assert.Equal(t, len(tr2.threads), 2)

	th01, ok := tr2.threads["th01"]
	assert.True(t, ok)
	assert.Equal(t, len(th01.regionStack), 0) // all per-thread regions were popped

	th02, ok := tr2.threads["th02"]
	assert.True(t, ok)
	assert.Equal(t, len(th02.regionStack), 0)

	assert.Equal(t, len(tr2.completedRegions), 3)

	r_2 := tr2.completedRegions[2]
	assert.Equal(t, r_2.nestingLevel, int64(1))
	assert.Equal(t, r_2.message, "m1")

	// r_2 is the {1,cat,l1} top-level region -- it's parent span is the thread
	assert.Equal(t, r_2.lifetime.parentSpanID, th01.lifetime.selfSpanID)
}

// Verify that we saw sufficient event data to generate telemetry.
func Test_Dataset_HaveStart(t *testing.T) {

	var events []string = []string{
		x_make_version(),
		//x_make_start(),
		x_make_error(x_error_1_msg, x_error_1_fmt),
		x_make_cmd_path(),
		x_make_cmd_ancestry(),
		x_make_cmd_name(),
		x_make_cmd_mode(),
		x_make_alias(),

		x_make_def_repo(1, x_repo_1_worktree),
		x_make_def_repo(3, x_repo_3_worktree), // non-contiguous

		x_make_atexit(), // Should be last
	}

	_, sufficient, err := load_test_dataset(t, events)
	assert.False(t, sufficient, "have sufficient data")
	assert.Nil(t, err)
}

// Verify that when cmd_name is "_run_dashed_" that we compose
// "<argv0>:<argv1>" for the command base name.  And verify that
// argv has enough args.
func Test_Dataset_RunDashed_Valid(t *testing.T) {

	var events []string = []string{
		x_make_version(),
		x_make_start_argv3("xx", "yy", "zz"),
		x_make_cmd_name_nh("_run_dashed_", "qq"),
		x_make_cmd_mode(),
		x_make_alias(),

		x_make_def_repo(1, x_repo_1_worktree),
		x_make_def_repo(3, x_repo_3_worktree), // non-contiguous

		x_make_atexit(), // Should be last
	}

	tr2, sufficient, _ := load_test_dataset(t, events)
	assert.True(t, sufficient, "have sufficient data")

	assert.Equal(t, tr2.trace2SID, x_sid)

	assert.Equal(t, tr2.process.qualifiedNames.exe, "xx")
	assert.Equal(t, tr2.process.qualifiedNames.exeVerb, "xx:yy")
	assert.Equal(t, tr2.process.qualifiedNames.exeVerbMode, "xx:yy#x-mode")
}

func Test_Dataset_RunDashed_Invalid(t *testing.T) {

	var events []string = []string{
		x_make_version(),
		x_make_start_argv1("xx"),
		x_make_cmd_name_nh("_run_dashed_", "qq"),
		x_make_cmd_mode(),
		x_make_alias(),

		x_make_def_repo(1, x_repo_1_worktree),
		x_make_def_repo(3, x_repo_3_worktree), // non-contiguous

		x_make_atexit(), // Should be last
	}

	tr2, sufficient, _ := load_test_dataset(t, events)
	assert.True(t, sufficient, "have sufficient data")

	assert.Equal(t, tr2.trace2SID, x_sid)

	assert.Equal(t, tr2.process.qualifiedNames.exe, "xx")
	assert.Equal(t, tr2.process.qualifiedNames.exeVerb, "xx:_run_dashed_")
	assert.Equal(t, tr2.process.qualifiedNames.exeVerbMode, "xx:_run_dashed_#x-mode")
}

func Test_Dataset_RejectClient_FSMonitor(t *testing.T) {

	var events []string = []string{
		x_make_version(),
		x_make_start_argv1("xx"),
		x_make_cmd_name_nh("fsmonitor--daemon", "qq"),
		x_make_cmd_mode(),
		x_make_alias(),

		x_make_def_repo(1, x_repo_1_worktree),
		x_make_def_repo(3, x_repo_3_worktree), // non-contiguous

		x_make_atexit(), // Should be last
	}

	tr2, sufficient, err := load_test_dataset(t, events)
	assert.Nil(t, tr2)
	assert.False(t, sufficient)
	assert.NotNil(t, err)

	rce, ok := err.(*RejectClientError)
	assert.True(t, ok)
	assert.True(t, rce.FSMonitor)
}

func Test_Dataset_Exec(t *testing.T) {

	var events []string = []string{
		x_make_version(),
		x_make_start_argv1("xx"),
		x_make_cmd_name_nh("foo", "qq"),
		x_make_cmd_mode(),
		x_make_alias(),

		x_make_exec(0, "git", "a0", "a1"),

		x_make_atexit(), // should be last
	}

	tr2, sufficient, _ := load_test_dataset(t, events)
	assert.True(t, sufficient, "have sufficient data")
	assert.Equal(t, tr2.trace2SID, x_sid)
	assert.Equal(t, tr2.process.qualifiedNames.exe, "xx")
	assert.Equal(t, tr2.process.qualifiedNames.exeVerb, "xx:foo")
	assert.Equal(t, tr2.process.qualifiedNames.exeVerbMode, "xx:foo#x-mode")

	assert.Equal(t, len(tr2.exec), 1)
	assert.NotNil(t, tr2.exec)
	assert.NotNil(t, tr2.exec[0])

	assert.Equal(t, tr2.exec[0].exe, "git")
	assert.Equal(t, len(tr2.exec[0].argv), 2)
	assert.Equal(t, tr2.exec[0].argv[0], "a0")
	assert.Equal(t, tr2.exec[0].argv[1], "a1")
}

// Given an array of raw Trace2 messages, parse and appy them
// to a newly created dataset.
func load_test_dataset(t *testing.T, events []string) (tr2 *trace2Dataset, sufficient bool, err error) {
	tr2 = NewTrace2Dataset(nil)

	for _, s := range events {
		// Use `parse_json()` rather than `evt_parse()` to avoid
		// all of the `rcvr_Base` setup.  This also bypasses the
		// helper tool.
		evt, err := parse_json([]byte(s))
		if err != nil {
			t.Fatalf("parse of '%s' failed: %s", s, err.Error())
		}
		if evt == nil {
			// An ignored event
			continue
		}

		err = evt_apply(tr2, evt)
		if err != nil {
			if rce, ok := err.(*RejectClientError); ok {
				return nil, false, rce
			}
			t.Fatalf("apply of '%s' failed: %s", s, err.Error())
		}
	}

	sufficient = tr2.prepareDataset()

	return tr2, sufficient, nil
}
