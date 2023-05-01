package trace2receiver

// Tests in this file are concerned with whether the Trace2 JSON
// event message contains all of the required fields, depending
// on the event type.
//
// We do not worry about whether values are of the correct type
// (such as confirming that an "argv" is an array), since value
// type checking was already handled in the jmap_get layer.

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

func Test_parseJsonEvent_Invalid_InvalidJson(t *testing.T) {
	s := `{"thread":"ma`

	_, err := parse_json([]byte(s))
	if err == nil {
		t.Fatalf("parse: failed to detect invalid JSON")
	}
}

// Verify that a missing, but required key/value pair was detected.
func verify_missing_required(err error, key string, t *testing.T) {
	if err == nil {
		t.Fatalf("parse: failed to detect missing field '%s'", key)
	}

	if !strings.Contains(err.Error(), fmt.Sprintf("key '%s' not present", key)) {
		t.Fatalf("parse: failed to detect missing field '%s': '%s'", key, err.Error())
	}
}

func Test_parseJsonEvent_Invalid_NoEventType(t *testing.T) {
	s := `{"thread":"main","time":"2023-01-10T14:57:42.956467Z"}`

	_, err := parse_json([]byte(s))
	verify_missing_required(err, "event", t)
}

func Test_parseJsonEvent_Common_NoSid(t *testing.T) {
	s := `{"event":"UNUSED","thread":"main","time":"2023-01-10T14:57:42.956467Z","file":"common-main.c","line":49,"evt":"3","exe":"2.38.1"}`

	_, err := parse_json([]byte(s))
	verify_missing_required(err, "sid", t)
}
func Test_parseJsonEvent_Common_NoThread(t *testing.T) {
	s := `{"event":"UNUSED","sid":"20230110T145742.956295Z-H0f5a2227-P00002b44","time":"2023-01-10T14:57:42.956467Z","file":"common-main.c","line":49,"evt":"3","exe":"2.38.1"}`

	_, err := parse_json([]byte(s))
	verify_missing_required(err, "thread", t)
}
func Test_parseJsonEvent_Common_NoTime(t *testing.T) {
	s := `{"event":"UNUSED","sid":"20230110T145742.956295Z-H0f5a2227-P00002b44","thread":"main","file":"common-main.c","line":49,"evt":"3","exe":"2.38.1"}`

	_, err := parse_json([]byte(s))
	verify_missing_required(err, "time", t)
}

func verify_common_field_values(s_json string, event_name string, t *testing.T) (evt *TrEvent) {
	evt, err := parse_json([]byte(s_json))
	if err != nil {
		t.Fatalf("parse: failed to parse valid JSON")
	}

	if evt.mf_event != event_name {
		t.Fatalf("parse: incorrect event name field")
	}

	// Values here must match those in `s_common`.
	if evt.mf_sid != "20230110T145742.956295Z-H0f5a2227-P00002b44" ||
		evt.mf_thread != "main" ||
		evt.mf_time.Year() != 2023 || evt.mf_time.Month() != 1 || evt.mf_time.Day() != 10 {
		t.Fatalf("parse: incorrectly parsed common fields")
	}

	return evt
}

// Common fields that we need on every test.  This is a fragment of JSON.
var s_common string = `"sid":"20230110T145742.956295Z-H0f5a2227-P00002b44","thread":"main","time":"2023-01-10T14:57:42.956467Z","file":"common-main.c","line":49`

func Test_parseJsonEvent_Common_Valid(t *testing.T) {
	s := fmt.Sprintf(`{"event":"UNUSED",%s}`, s_common)

	verify_common_field_values(s, "UNUSED", t)
}

func fail_nil_substructure(t *testing.T, n string) {
	t.Fatalf("parse: failed to create 'evt.pm_%s' substructure", n)
}
func fail_wrong(t *testing.T, n string) {
	t.Fatalf("parse: incorrectly parsed '%s' specific fields", n)
}

func Test_parseJsonEvent_Version_Valid(t *testing.T) {
	n := "version"
	s := fmt.Sprintf(`{%s,"event":"%s","evt":"3","exe":"2.38.1"}`, s_common, n)

	evt := verify_common_field_values(s, n, t)

	if evt.pm_version == nil {
		fail_nil_substructure(t, n)
	}
	if evt.pm_version.mf_evt != "3" ||
		evt.pm_version.mf_exe != "2.38.1" {
		fail_wrong(t, n)
	}
}
func Test_parseJsonEvent_Version_MissingEvt(t *testing.T) {
	n := "version"
	s := fmt.Sprintf(`{%s,"event":"%s","exe":"2.38.1"}`, s_common, n)

	_, err := parse_json([]byte(s))
	verify_missing_required(err, "evt", t)
}
func Test_parseJsonEvent_Version_MissingExe(t *testing.T) {
	n := "version"
	s := fmt.Sprintf(`{%s,"event":"%s","evt":"3"}`, s_common, n)

	_, err := parse_json([]byte(s))
	verify_missing_required(err, "exe", t)
}

func Test_parseJsonEvent_Start_Valid(t *testing.T) {
	n := "start"
	s := fmt.Sprintf(`{%s,"event":"%s","argv":["git","version"]}`, s_common, n)

	evt := verify_common_field_values(s, n, t)

	if evt.pm_start == nil {
		fail_nil_substructure(t, n)
	}

	if len(evt.pm_start.mf_argv) != 2 ||
		evt.pm_start.mf_argv[0] != "git" ||
		evt.pm_start.mf_argv[1] != "version" {
		fail_wrong(t, n)
	}
}
func Test_parseJsonEvent_Start_MissingArgv(t *testing.T) {
	n := "start"
	s := fmt.Sprintf(`{%s,"event":"%s"}`, s_common, n)

	_, err := parse_json([]byte(s))
	verify_missing_required(err, "argv", t)
}

func shared_verify_exit(t *testing.T, n string) {
	s := fmt.Sprintf(`{%s,"event":"%s","code":%d}`, s_common, n, 42)

	evt := verify_common_field_values(s, n, t)

	if evt.pm_atexit == nil {
		fail_nil_substructure(t, n)
	}

	if evt.pm_atexit.mf_code != 42 {
		fail_wrong(t, n)
	}
}
func Test_parseJsonEvent_AtExit_Valid(t *testing.T) {
	shared_verify_exit(t, "atexit")
}
func Test_parseJsonEvent_Exit_Valid(t *testing.T) {
	shared_verify_exit(t, "exit")
}

func Test_parseJsonEvent_Signal_Valid(t *testing.T) {
	n := "signal"
	s := fmt.Sprintf(`{%s,"event":"%s","signo":%d}`, s_common, n, 13)

	evt := verify_common_field_values(s, n, t)

	if evt.pm_signal == nil {
		fail_nil_substructure(t, n)
	}

	if evt.pm_signal.mf_signo != 13 {
		fail_wrong(t, n)
	}
}

func Test_parseJsonEvent_Error_Valid(t *testing.T) {
	n := "error"
	s := fmt.Sprintf(`{%s,"event":"%s","msg":"%s","fmt":"%s"}`,
		s_common, n, "My Error Message", "My Format String")

	evt := verify_common_field_values(s, n, t)

	if evt.pm_error == nil {
		fail_nil_substructure(t, n)
	}

	if evt.pm_error.mf_msg != "My Error Message" ||
		evt.pm_error.mf_fmt != "My Format String" {
		fail_wrong(t, n)
	}
}

func Test_parseJsonEvent_CmdPath_Valid(t *testing.T) {
	n := "cmd_path"
	s := fmt.Sprintf(`{%s,"event":"%s","path":"/a/b/c"}`, s_common, n)

	evt := verify_common_field_values(s, n, t)

	if evt.pm_cmd_path == nil {
		fail_nil_substructure(t, n)
	}

	if evt.pm_cmd_path.mf_path != "/a/b/c" {
		fail_wrong(t, n)
	}
}

func Test_parseJsonEvent_CmdAncestry_Valid(t *testing.T) {
	n := "cmd_ancestry"
	s := fmt.Sprintf(`{%s,"event":"%s","ancestry":["a","b","c"]}`, s_common, n)

	evt := verify_common_field_values(s, n, t)

	if evt.pm_cmd_ancestry == nil {
		fail_nil_substructure(t, n)
	}

	if len(evt.pm_cmd_ancestry.mf_ancestry) != 3 ||
		evt.pm_cmd_ancestry.mf_ancestry[0] != "a" ||
		evt.pm_cmd_ancestry.mf_ancestry[1] != "b" ||
		evt.pm_cmd_ancestry.mf_ancestry[2] != "c" {
		fail_wrong(t, n)
	}
}

func Test_parseJsonEvent_CmdName_Valid(t *testing.T) {
	n := "cmd_name"
	s := fmt.Sprintf(`{%s,"event":"%s","name":"foo","hierarchy":"abc/def/ghi"}`, s_common, n)

	evt := verify_common_field_values(s, n, t)

	if evt.pm_cmd_name == nil {
		fail_nil_substructure(t, n)
	}

	if evt.pm_cmd_name.mf_name != "foo" ||
		evt.pm_cmd_name.mf_hierarchy != "abc/def/ghi" {
		fail_wrong(t, n)
	}
}

func Test_parseJsonEvent_CmdMode_Valid(t *testing.T) {
	n := "cmd_mode"
	s := fmt.Sprintf(`{%s,"event":"%s","name":"branch"}`, s_common, n)

	evt := verify_common_field_values(s, n, t)

	if evt.pm_cmd_mode == nil {
		fail_nil_substructure(t, n)
	}

	if evt.pm_cmd_mode.mf_name != "branch" {
		fail_wrong(t, n)
	}
}

func Test_parseJsonEvent_Alias_Valid(t *testing.T) {
	n := "alias"
	s := fmt.Sprintf(`{%s,"event":"%s","alias":"foo","argv":["a","b","c"]}`, s_common, n)

	evt := verify_common_field_values(s, n, t)

	if evt.pm_alias == nil {
		fail_nil_substructure(t, n)
	}

	if evt.pm_alias.mf_alias != "foo" ||
		len(evt.pm_alias.mf_argv) != 3 ||
		evt.pm_alias.mf_argv[0] != "a" ||
		evt.pm_alias.mf_argv[1] != "b" ||
		evt.pm_alias.mf_argv[2] != "c" {
		fail_wrong(t, n)
	}
}

func Test_parseJsonEvent_ChildStart_Valid(t *testing.T) {
	n := "child_start"
	s := fmt.Sprintf(`{%s,"event":"%s","child_id":0,"child_class":"xyz","use_shell":false,"argv":["a","b","c"],"cd":"/tmp"}`, s_common, n)

	evt := verify_common_field_values(s, n, t)

	if evt.pm_child_start == nil {
		fail_nil_substructure(t, n)
	}

	if evt.pm_child_start.mf_child_id != 0 ||
		evt.pm_child_start.mf_child_class != "xyz" ||
		evt.pm_child_start.mf_use_shell != false ||
		len(evt.pm_child_start.mf_argv) != 3 ||
		evt.pm_child_start.mf_argv[0] != "a" ||
		evt.pm_child_start.mf_argv[1] != "b" ||
		evt.pm_child_start.mf_argv[2] != "c" {
		fail_wrong(t, n)
	}

	if evt.pm_child_start.pmf_cd == nil ||
		*evt.pm_child_start.pmf_cd != "/tmp" {
		fail_wrong(t, n)
	}
}
func Test_parseJsonEvent_ChildStart_ValidHook(t *testing.T) {
	n := "child_start"
	s := fmt.Sprintf(`{%s,"event":"%s","child_id":0,"child_class":"hook","use_shell":false,"argv":["a","b","c"],"hook_name":"prefetch"}`, s_common, n)

	evt := verify_common_field_values(s, n, t)

	if evt.pm_child_start == nil {
		fail_nil_substructure(t, n)
	}

	if evt.pm_child_start.mf_child_id != 0 ||
		evt.pm_child_start.mf_child_class != "hook" ||
		evt.pm_child_start.mf_use_shell != false ||
		len(evt.pm_child_start.mf_argv) != 3 ||
		evt.pm_child_start.mf_argv[0] != "a" ||
		evt.pm_child_start.mf_argv[1] != "b" ||
		evt.pm_child_start.mf_argv[2] != "c" {
		fail_wrong(t, n)
	}

	if evt.pm_child_start.pmf_hook_name == nil ||
		*evt.pm_child_start.pmf_hook_name != "prefetch" {
		fail_wrong(t, n)
	}
}

func Test_parseJsonEvent_ChildExit_Valid(t *testing.T) {
	n := "child_exit"
	s := fmt.Sprintf(`{%s,"event":"%s","child_id":0,"pid":1234,"code":0}`, s_common, n)

	evt := verify_common_field_values(s, n, t)

	if evt.pm_child_exit == nil {
		fail_nil_substructure(t, n)
	}

	if evt.pm_child_exit.mf_child_id != 0 ||
		evt.pm_child_exit.mf_pid != 1234 ||
		evt.pm_child_exit.mf_code != 0 {
		fail_wrong(t, n)
	}
}

func Test_parseJsonEvent_ChildReady_Valid(t *testing.T) {
	n := "child_ready"
	s := fmt.Sprintf(`{%s,"event":"%s","child_id":0,"pid":1234,"ready":"timeout"}`, s_common, n)

	evt := verify_common_field_values(s, n, t)

	if evt.pm_child_ready == nil {
		fail_nil_substructure(t, n)
	}

	if evt.pm_child_ready.mf_child_id != 0 ||
		evt.pm_child_ready.mf_pid != 1234 ||
		evt.pm_child_ready.mf_ready != "timeout" {
		fail_wrong(t, n)
	}
}

func Test_parseJsonEvent_ThreadStart_Valid(t *testing.T) {
	n := "thread_start"
	s := fmt.Sprintf(`{%s,"event":"%s"}`, s_common, n)

	evt := verify_common_field_values(s, n, t)

	if evt.pm_thread_start == nil {
		fail_nil_substructure(t, n)
	}
}

func Test_parseJsonEvent_ThreadExit_Valid(t *testing.T) {
	n := "thread_exit"
	s := fmt.Sprintf(`{%s,"event":"%s"}`, s_common, n)

	evt := verify_common_field_values(s, n, t)

	if evt.pm_thread_exit == nil {
		fail_nil_substructure(t, n)
	}
}

func Test_parseJsonEvent_Exec_Valid(t *testing.T) {
	n := "exec"
	s := fmt.Sprintf(`{%s,"event":"%s","exec_id":0,"exe":"git","argv":["a","b","c"]}`, s_common, n)

	evt := verify_common_field_values(s, n, t)

	if evt.pm_exec == nil {
		fail_nil_substructure(t, n)
	}

	if evt.pm_exec.mf_exec_id != 0 ||
		len(evt.pm_exec.mf_argv) != 3 ||
		evt.pm_exec.mf_argv[0] != "a" ||
		evt.pm_exec.mf_argv[1] != "b" ||
		evt.pm_exec.mf_argv[2] != "c" {
		fail_wrong(t, n)
	}

	if evt.pm_exec.pmf_exe == nil ||
		*evt.pm_exec.pmf_exe != "git" {
		fail_wrong(t, n)
	}
}

func Test_parseJsonEvent_ExecResult_Valid(t *testing.T) {
	n := "exec_result"
	s := fmt.Sprintf(`{%s,"event":"%s","exec_id":0,"code":0}`, s_common, n)

	evt := verify_common_field_values(s, n, t)

	if evt.pm_exec_result == nil {
		fail_nil_substructure(t, n)
	}

	if evt.pm_exec_result.mf_exec_id != 0 ||
		evt.pm_exec_result.mf_code != 0 {
		fail_wrong(t, n)
	}
}

func Test_parseJsonEvent_DefParam_Valid(t *testing.T) {
	n := "def_param"
	s := fmt.Sprintf(`{%s,"event":"%s","param":"core.abbrev","value":"7","scope":"global"}`, s_common, n)

	evt := verify_common_field_values(s, n, t)

	if evt.pm_def_param == nil {
		fail_nil_substructure(t, n)
	}

	if evt.pm_def_param.mf_param != "core.abbrev" ||
		evt.pm_def_param.mf_value != "7" {
		fail_wrong(t, n)
	}

	if evt.pm_def_param.pmf_scope == nil ||
		*evt.pm_def_param.pmf_scope != "global" {
		fail_wrong(t, n)
	}
}

func Test_parseJsonEvent_DefRepo_Valid(t *testing.T) {
	n := "def_repo"
	s := fmt.Sprintf(`{%s,"event":"%s","repo":3,"worktree":"/a/b/c"}`, s_common, n)

	evt := verify_common_field_values(s, n, t)

	if evt.pm_def_repo == nil {
		fail_nil_substructure(t, n)
	}

	if evt.pm_def_repo.mf_worktree != "/a/b/c" {
		fail_wrong(t, n)
	}

	// "repo" is an optional common field
	if evt.pmf_repo == nil ||
		*evt.pmf_repo != 3 {
		fail_wrong(t, n)
	}
}

func Test_parseJsonEvent_RegionEnter_Valid(t *testing.T) {
	n := "region_enter"
	s := fmt.Sprintf(`{%s,"event":"%s","repo":3,"nesting":7,"category":"c","label":"l","msg":"m"}`, s_common, n)

	evt := verify_common_field_values(s, n, t)

	if evt.pm_region_enter == nil {
		fail_nil_substructure(t, n)
	}

	if evt.pm_region_enter.mf_nesting != 7 {
		fail_wrong(t, n)
	}

	if evt.pm_region_enter.pmf_category == nil ||
		*evt.pm_region_enter.pmf_category != "c" {
		fail_wrong(t, n)
	}
	if evt.pm_region_enter.pmf_label == nil ||
		*evt.pm_region_enter.pmf_label != "l" {
		fail_wrong(t, n)
	}
	if evt.pm_region_enter.pmf_msg == nil ||
		*evt.pm_region_enter.pmf_msg != "m" {
		fail_wrong(t, n)
	}

	// "repo" is an optional common field
	if evt.pmf_repo == nil ||
		*evt.pmf_repo != 3 {
		fail_wrong(t, n)
	}
}

func Test_parseJsonEvent_RegionLeave_Valid(t *testing.T) {
	n := "region_leave"
	s := fmt.Sprintf(`{%s,"event":"%s","repo":3,"nesting":7,"category":"c","label":"l","msg":"m"}`, s_common, n)

	evt := verify_common_field_values(s, n, t)

	if evt.pm_region_leave == nil {
		fail_nil_substructure(t, n)
	}

	if evt.pm_region_leave.mf_nesting != 7 {
		fail_wrong(t, n)
	}

	if evt.pm_region_leave.pmf_category == nil ||
		*evt.pm_region_leave.pmf_category != "c" {
		fail_wrong(t, n)
	}
	if evt.pm_region_leave.pmf_label == nil ||
		*evt.pm_region_leave.pmf_label != "l" {
		fail_wrong(t, n)
	}
	if evt.pm_region_leave.pmf_msg == nil ||
		*evt.pm_region_leave.pmf_msg != "m" {
		fail_wrong(t, n)
	}

	// "repo" is an optional common field
	if evt.pmf_repo == nil ||
		*evt.pmf_repo != 3 {
		fail_wrong(t, n)
	}
}

func Test_parseJsonEvent_DataString_Valid(t *testing.T) {
	n := "data"
	s := fmt.Sprintf(`{%s,"event":"%s","repo":3,"nesting":7,"category":"c","key":"k","value":"v"}`, s_common, n)

	evt := verify_common_field_values(s, n, t)

	if evt.pm_generic_data == nil {
		fail_nil_substructure(t, n)
	}

	if evt.pm_generic_data.mf_nesting != 7 ||
		evt.pm_generic_data.mf_category != "c" ||
		evt.pm_generic_data.mf_key != "k" {
		fail_wrong(t, n)
	}

	switch s := evt.pm_generic_data.mf_generic_value.(type) {
	case string:
		if s != "v" {
			fail_wrong(t, n)
		}
	default:
		fail_wrong(t, n)
	}

	// "repo" is an optional common field
	if evt.pmf_repo == nil ||
		*evt.pmf_repo != 3 {
		fail_wrong(t, n)
	}
}

func Test_parseJsonEvent_DataInt64_Valid(t *testing.T) {
	n := "data"
	s := fmt.Sprintf(`{%s,"event":"%s","repo":3,"nesting":7,"category":"c","key":"k","value":42}`, s_common, n)

	evt := verify_common_field_values(s, n, t)

	if evt.pm_generic_data == nil {
		fail_nil_substructure(t, n)
	}

	if evt.pm_generic_data.mf_nesting != 7 ||
		evt.pm_generic_data.mf_category != "c" ||
		evt.pm_generic_data.mf_key != "k" {
		fail_wrong(t, n)
	}

	switch i := evt.pm_generic_data.mf_generic_value.(type) {
	case int64:
		if i != 42 {
			fail_wrong(t, n)
		}
	default:
		fail_wrong(t, n)
	}

	// "repo" is an optional common field
	if evt.pmf_repo == nil ||
		*evt.pmf_repo != 3 {
		fail_wrong(t, n)
	}
}

func Test_parseJsonEvent_DataJson_Valid(t *testing.T) {
	n := "data_json"
	s := fmt.Sprintf(`{%s,"event":"%s","repo":3,"nesting":7,"category":"c","key":"k","value":{"a":1,"b":"v"}}`, s_common, n)

	evt := verify_common_field_values(s, n, t)

	if evt.pm_generic_data == nil {
		fail_nil_substructure(t, n)
	}

	if evt.pm_generic_data.mf_nesting != 7 ||
		evt.pm_generic_data.mf_category != "c" ||
		evt.pm_generic_data.mf_key != "k" {
		fail_wrong(t, n)
	}

	byteValue, err := json.Marshal(evt.pm_generic_data.mf_generic_value)
	if err != nil {
		fail_wrong(t, n)
	}
	v := string(byteValue)

	if !strings.Contains(v, "\"a\":1") ||
		!strings.Contains(v, "\"b\":\"v\"") {
		fail_wrong(t, n)
	}

	// "repo" is an optional common field
	if evt.pmf_repo == nil ||
		*evt.pmf_repo != 3 {
		fail_wrong(t, n)
	}
}

func shared_verify_timer(t *testing.T, n string) {
	s := fmt.Sprintf(`{%s,"event":"%s","category":"c","name":"n","intervals":3,"t_total":1.0,"t_min":2.0,"t_max":3.0}`, s_common, n)

	evt := verify_common_field_values(s, n, t)

	if evt.pm_timer == nil {
		fail_nil_substructure(t, n)
	}

	if evt.pm_timer.mf_category != "c" ||
		evt.pm_timer.mf_name != "n" ||
		evt.pm_timer.mf_intervals != 3 ||
		!float_is_near(evt.pm_timer.mf_t_total, 1.0) ||
		!float_is_near(evt.pm_timer.mf_t_min, 2.0) ||
		!float_is_near(evt.pm_timer.mf_t_max, 3.0) {
		fail_wrong(t, n)
	}
}

func Test_parseJsonEvent_Timer_Valid(t *testing.T) {
	shared_verify_timer(t, "timer")
}
func Test_parseJsonEvent_ThTimer_Valid(t *testing.T) {
	shared_verify_timer(t, "th_timer")
}

func shared_verify_counter(t *testing.T, n string) {
	s := fmt.Sprintf(`{%s,"event":"%s","category":"c","name":"n","count":5}`, s_common, n)

	evt := verify_common_field_values(s, n, t)

	if evt.pm_counter == nil {
		fail_nil_substructure(t, n)
	}

	if evt.pm_counter.mf_category != "c" ||
		evt.pm_counter.mf_name != "n" ||
		evt.pm_counter.mf_count != 5 {
		fail_wrong(t, n)
	}
}

func Test_parseJsonEvent_Counter_Valid(t *testing.T) {
	shared_verify_counter(t, "counter")
}
func Test_parseJsonEvent_ThCounter_Valid(t *testing.T) {
	shared_verify_counter(t, "th_counter")
}
