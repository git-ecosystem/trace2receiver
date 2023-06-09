package trace2receiver

import (
	"bytes"
	"encoding/json"
	"fmt"
	"time"

	"go.uber.org/zap"
)

type TrEvent struct {
	// Common/Required fields for all Trace2 JSON Events.

	mf_event  string
	mf_sid    string
	mf_thread string
	mf_time   time.Time
	pmf_repo  *int64 // (aka repo_id) is optional
	//mf_file -- optional
	//mf_line -- optional

	// Variable portion depends on the type of the event.

	pm_version      *TrEventVersion
	pm_start        *TrEventStart
	pm_atexit       *TrEventAtExit // "exit" or "atexit"
	pm_signal       *TrEventSignal
	pm_error        *TrEventError
	pm_cmd_path     *TrEventCmdPath
	pm_cmd_ancestry *TrEventCmdAncestry
	pm_cmd_name     *TrEventCmdName
	pm_cmd_mode     *TrEventCmdMode
	pm_alias        *TrEventAlias
	pm_child_start  *TrEventChildStart
	pm_child_exit   *TrEventChildExit
	pm_child_ready  *TrEventChildReady
	pm_thread_start *TrEventThreadStart
	pm_thread_exit  *TrEventThreadExit
	pm_exec         *TrEventExec
	pm_exec_result  *TrEventExecResult
	pm_def_param    *TrEventDefParam
	pm_def_repo     *TrEventDefRepo
	pm_region_enter *TrEventRegionEnter
	pm_region_leave *TrEventRegionLeave
	pm_generic_data *TrEventGenericData
	pm_timer        *TrEventTimer   // "timer" and "th_timer"
	pm_counter      *TrEventCounter // "counter" and "th_counter"
}

type FnExtractKeys func(evt *TrEvent, jm *jmap) (err error)
type ExtractKeysMap map[string]FnExtractKeys

var ekm *ExtractKeysMap = &ExtractKeysMap{
	"version":        extract_keys__version,
	"start":          extract_keys__start,
	"exit":           extract_keys__atexit,
	"atexit":         extract_keys__atexit,
	"signal":         extract_keys__signal,
	"error":          extract_keys__error,
	"cmd_path":       extract_keys__cmd_path,
	"cmd_ancestry":   extract_keys__cmd_ancestry,
	"cmd_name":       extract_keys__cmd_name,
	"cmd_mode":       extract_keys__cmd_mode,
	"alias":          extract_keys__alias,
	"child_start":    extract_keys__child_start,
	"child_exit":     extract_keys__child_exit,
	"child_ready":    extract_keys__child_ready,
	"thread_start":   extract_keys__thread_start,
	"thread_exit":    extract_keys__thread_exit,
	"exec":           extract_keys__exec,
	"exec_result":    extract_keys__exec_result,
	"def_param":      extract_keys__def_param,
	"def_repo":       extract_keys__def_repo,
	"region_enter":   extract_keys__region_enter,
	"region_leave":   extract_keys__region_leave,
	"data":           extract_keys__data,
	"data_json":      extract_keys__data_json,
	"timer":          extract_keys__timer,
	"th_timer":       extract_keys__timer,
	"counter":        extract_keys__counter,
	"th_counter":     extract_keys__counter,
	"too_many_files": nil, // we don't care about this
}

var CommandControlVerbPrefix []byte = []byte("cc: ")

// Parse the raw line of text from the client and parse it.
//
// If is JSON, parse and validate it as a Trace2 event message.
// If it is a command/control from the helper tool, process it.
// If it is blank/empty or a "#-style" comment line, ignore it.
//
// Returns (nil, err) if we had an error.
// Returns (nil, nil) if we had command/control data.
// Returns (evt, nil) if we had event data.
func evt_parse(rawLine []byte, logger *zap.Logger, allowCommands bool) (*TrEvent, error) {
	trimmed := bytes.TrimSpace(rawLine)

	if len(trimmed) == 0 || trimmed[0] == '#' {
		return nil, nil
	}

	if trimmed[0] == '{' {
		return parse_json(trimmed)
	}

	if bytes.HasPrefix(trimmed, CommandControlVerbPrefix) {
		if allowCommands {
			return nil, do_command_verb(trimmed[len(CommandControlVerbPrefix):], logger)
		} else {
			logger.Debug(fmt.Sprintf("command verbs are disabled: '%s'", trimmed))
			return nil, nil
		}
	}

	logger.Debug(fmt.Sprintf("unrecognized data stream verb: '%s'", trimmed))
	return nil, nil
}

func do_command_verb(cmd []byte, logger *zap.Logger) error {
	logger.Debug(fmt.Sprintf("Command verb: '%s'", cmd))

	// TODO do something with the rest of the line and return.

	logger.Debug(fmt.Sprintf("invalid command verb: '%s'", cmd))
	return nil
}

// Process a raw line of text from the client.  This should contain a single
// line of Trace2 data in JSON format.  But we do allow command and control
// verbs (primarily for test and debug).
func processRawLine(rawLine []byte, tr2 *trace2Dataset, logger *zap.Logger, allowCommands bool) error {

	logger.Debug(fmt.Sprintf("[dsid %06d] saw: %s", tr2.datasetId, rawLine))

	evt, err := evt_parse(rawLine, logger, allowCommands)
	if err != nil {
		logger.Error(err.Error())
		return err
	}

	if evt != nil {
		tr2.sawData = true

		err = evt_apply(tr2, evt)
		if err != nil {
			if rce, ok := err.(*RejectClientError); ok {
				// Silently reject the client without logging an error.
				logger.Debug(rce.Error())
				return rce
			}
			logger.Error(err.Error())
			return err
		}
	}

	return nil
}

func parse_json(line []byte) (*TrEvent, error) {
	var err error
	var jm *jmap = new(jmap)

	if err = json.Unmarshal(line, jm); err != nil {
		return nil, err
	}

	evt := &TrEvent{}

	if err = extract_keys__common(evt, jm); err != nil {
		return evt, err
	}

	ekfn, ok := (*ekm)[evt.mf_event]
	if !ok {
		// Unrecognized event type. Ignore since the Trace2 format
		// is allowed to add new event types in the future.
		//
		// TODO Consider debug level logging this.
		return evt, nil
	}
	if ekfn == nil {
		// Recognized event type, but we either don't care about
		// it or don't have any event-specific fields to extract.
		return evt, nil
	}
	return evt, ekfn(evt, jm)
}

// Parse common key/value pairs found in almost all Trace2 events.
func extract_keys__common(evt *TrEvent, jm *jmap) (err error) {
	if evt.mf_event, err = jm.getRequiredString("event"); err != nil {
		return err
	}
	if evt.mf_sid, err = jm.getRequiredString("sid"); err != nil {
		return err
	}
	if evt.mf_thread, err = jm.getRequiredString("thread"); err != nil {
		return err
	}
	if evt.mf_time, err = jm.getRequiredTime("time"); err != nil {
		// Force a failure if "time" is omitted.
		//
		// We require "time" on all events so that we can set the span
		// duration on bracketed units of work.  However, "time" is not
		// emitted by Git on most events when in "brief" mode.  For now,
		// don't bother trying to support clients in brief mode.
		return err
	}

	if evt.pmf_repo, err = jm.getOptionalInt64("repo"); err != nil {
		return err
	}

	// TODO Do we care about "file" and "line"?

	return nil
}

// Event fields only present in an "event":"version" event
type TrEventVersion struct {
	mf_evt string // 'evt'
	mf_exe string // 'exe'
}

func extract_keys__version(evt *TrEvent, jm *jmap) (err error) {
	evt.pm_version = &TrEventVersion{}

	if evt.pm_version.mf_evt, err = jm.getRequiredString("evt"); err != nil {
		return err
	}
	if evt.pm_version.mf_exe, err = jm.getRequiredString("exe"); err != nil {
		return err
	}

	return nil
}

// Event fields only present in an "event":"start" event
type TrEventStart struct {
	mf_argv []interface{}
	// TODO? t_abs
}

func extract_keys__start(evt *TrEvent, jm *jmap) (err error) {
	evt.pm_start = &TrEventStart{}

	if evt.pm_start.mf_argv, err = jm.getRequiredArray("argv"); err != nil {
		return err
	}

	return nil
}

// Event fields only present in an "event":"exit" or "event":"atexit" event
type TrEventAtExit struct {
	mf_code int64
	// TODO? t_abs
}

func extract_keys__atexit(evt *TrEvent, jm *jmap) (err error) {
	if evt.pm_atexit == nil {
		evt.pm_atexit = &TrEventAtExit{}
	}

	if evt.pm_atexit.mf_code, err = jm.getRequiredInt64("code"); err != nil {
		return err
	}

	return nil
}

// Event fields only present in an "event":"signal" event
type TrEventSignal struct {
	mf_signo int64
	// TODO? t_abs
}

func extract_keys__signal(evt *TrEvent, jm *jmap) (err error) {
	evt.pm_signal = &TrEventSignal{}

	if evt.pm_signal.mf_signo, err = jm.getRequiredInt64("signo"); err != nil {
		return err
	}

	return nil
}

// Event fields only present in an "event":"error" event
type TrEventError struct {
	mf_msg string
	mf_fmt string
}

func extract_keys__error(evt *TrEvent, jm *jmap) (err error) {
	evt.pm_error = &TrEventError{}

	if evt.pm_error.mf_msg, err = jm.getRequiredString("msg"); err != nil {
		return err
	}
	if evt.pm_error.mf_fmt, err = jm.getRequiredString("fmt"); err != nil {
		return err
	}

	return nil
}

// Event fields only present in an "event":"cmd_path" event
type TrEventCmdPath struct {
	mf_path string
}

func extract_keys__cmd_path(evt *TrEvent, jm *jmap) (err error) {
	evt.pm_cmd_path = &TrEventCmdPath{}

	if evt.pm_cmd_path.mf_path, err = jm.getRequiredString("path"); err != nil {
		return err
	}

	return nil
}

// Event fields only present in an "event":"ancestry" event
type TrEventCmdAncestry struct {
	mf_ancestry []interface{}
}

func extract_keys__cmd_ancestry(evt *TrEvent, jm *jmap) (err error) {
	evt.pm_cmd_ancestry = &TrEventCmdAncestry{}

	if evt.pm_cmd_ancestry.mf_ancestry, err = jm.getRequiredArray("ancestry"); err != nil {
		return err
	}

	return nil
}

// Event fields only present in an "event":"cmd_name" event
type TrEventCmdName struct {
	mf_name      string
	mf_hierarchy string
}

func extract_keys__cmd_name(evt *TrEvent, jm *jmap) (err error) {
	evt.pm_cmd_name = &TrEventCmdName{}

	if evt.pm_cmd_name.mf_name, err = jm.getRequiredString("name"); err != nil {
		return err
	}
	if evt.pm_cmd_name.mf_hierarchy, err = jm.getRequiredString("hierarchy"); err != nil {
		return err
	}

	return nil
}

// Event fields only present in an "event":"cmd_mode" event
type TrEventCmdMode struct {
	mf_name string
}

func extract_keys__cmd_mode(evt *TrEvent, jm *jmap) (err error) {
	evt.pm_cmd_mode = &TrEventCmdMode{}

	if evt.pm_cmd_mode.mf_name, err = jm.getRequiredString("name"); err != nil {
		return err
	}

	return nil
}

// Event fields only present in an "event":"alias" event
type TrEventAlias struct {
	mf_alias string
	mf_argv  []interface{}
}

func extract_keys__alias(evt *TrEvent, jm *jmap) (err error) {
	evt.pm_alias = &TrEventAlias{}

	if evt.pm_alias.mf_alias, err = jm.getRequiredString("alias"); err != nil {
		return err
	}
	if evt.pm_alias.mf_argv, err = jm.getRequiredArray("argv"); err != nil {
		return err
	}

	return nil
}

// Event fields only present in an "event":"child_start" event
type TrEventChildStart struct {
	mf_child_id    int64
	mf_child_class string
	mf_use_shell   bool
	mf_argv        []interface{}
	pmf_hook_name  *string // only set when child_class is "hook"
	pmf_cd         *string // optional
}

func extract_keys__child_start(evt *TrEvent, jm *jmap) (err error) {
	evt.pm_child_start = &TrEventChildStart{}

	if evt.pm_child_start.mf_child_id, err = jm.getRequiredInt64("child_id"); err != nil {
		return err
	}
	if evt.pm_child_start.mf_child_class, err = jm.getRequiredString("child_class"); err != nil {
		return err
	}
	if evt.pm_child_start.mf_use_shell, err = jm.getRequiredBool("use_shell"); err != nil {
		return err
	}
	if evt.pm_child_start.mf_argv, err = jm.getRequiredArray("argv"); err != nil {
		return err
	}

	if evt.pm_child_start.mf_child_class == "hook" {
		evt.pm_child_start.pmf_hook_name = new(string)
		if *evt.pm_child_start.pmf_hook_name, err = jm.getRequiredString("hook_name"); err != nil {
			return err
		}
	}

	if evt.pm_child_start.pmf_cd, err = jm.getOptionalString("cd"); err != nil {
		return err
	}

	return nil
}

// Event fields only present in an "event":"child_exit" event
type TrEventChildExit struct {
	mf_child_id int64
	mf_pid      int64
	mf_code     int64
	// TODO? t_rel
}

func extract_keys__child_exit(evt *TrEvent, jm *jmap) (err error) {
	evt.pm_child_exit = &TrEventChildExit{}

	if evt.pm_child_exit.mf_child_id, err = jm.getRequiredInt64("child_id"); err != nil {
		return err
	}
	if evt.pm_child_exit.mf_pid, err = jm.getRequiredInt64("pid"); err != nil {
		return err
	}
	if evt.pm_child_exit.mf_code, err = jm.getRequiredInt64("code"); err != nil {
		return err
	}

	return nil
}

// Event fields only present in an "event":"child_ready" event.
// Note that there is no exit-code, only a ready state hint.
type TrEventChildReady struct {
	mf_child_id int64
	mf_pid      int64
	mf_ready    string
	// TODO? t_rel
}

func extract_keys__child_ready(evt *TrEvent, jm *jmap) (err error) {
	evt.pm_child_ready = &TrEventChildReady{}

	if evt.pm_child_ready.mf_child_id, err = jm.getRequiredInt64("child_id"); err != nil {
		return err
	}
	if evt.pm_child_ready.mf_pid, err = jm.getRequiredInt64("pid"); err != nil {
		return err
	}
	if evt.pm_child_ready.mf_ready, err = jm.getRequiredString("ready"); err != nil {
		return err
	}

	return nil
}

// Event fields only present in an "event":"thread_start" event
type TrEventThreadStart struct {
}

func extract_keys__thread_start(evt *TrEvent, jm *jmap) (err error) {
	evt.pm_thread_start = &TrEventThreadStart{}

	return nil
}

// Event fields only present in an "event":"thread_exit" event
type TrEventThreadExit struct {
	// TODO? t_rel
}

func extract_keys__thread_exit(evt *TrEvent, jm *jmap) (err error) {
	evt.pm_thread_exit = &TrEventThreadExit{}

	return nil
}

// Event fields only present in an "event":"exec" event
type TrEventExec struct {
	mf_exec_id int64
	mf_argv    []interface{}
	pmf_exe    *string // optional
}

func extract_keys__exec(evt *TrEvent, jm *jmap) (err error) {
	evt.pm_exec = &TrEventExec{}

	if evt.pm_exec.mf_exec_id, err = jm.getRequiredInt64("exec_id"); err != nil {
		return err
	}
	if evt.pm_exec.mf_argv, err = jm.getRequiredArray("argv"); err != nil {
		return err
	}

	if evt.pm_exec.pmf_exe, err = jm.getOptionalString("exe"); err != nil {
		return err
	}

	return nil
}

// Event fields only present in an "event":"exec_result" event
type TrEventExecResult struct {
	mf_exec_id int64
	mf_code    int64
}

func extract_keys__exec_result(evt *TrEvent, jm *jmap) (err error) {
	evt.pm_exec_result = &TrEventExecResult{}

	if evt.pm_exec_result.mf_exec_id, err = jm.getRequiredInt64("exec_id"); err != nil {
		return err
	}
	if evt.pm_exec_result.mf_code, err = jm.getRequiredInt64("code"); err != nil {
		return err
	}

	return nil
}

// Event fields only present in an "event":"def_param" event
type TrEventDefParam struct {
	mf_param  string
	mf_value  string
	pmf_scope *string // added to Git in 2022/08, so do not require it yet
}

func extract_keys__def_param(evt *TrEvent, jm *jmap) (err error) {
	evt.pm_def_param = &TrEventDefParam{}

	if evt.pm_def_param.mf_param, err = jm.getRequiredString("param"); err != nil {
		return err
	}
	if evt.pm_def_param.mf_value, err = jm.getRequiredString("value"); err != nil {
		return err
	}

	if evt.pm_def_param.pmf_scope, err = jm.getOptionalString("scope"); err != nil {
		return err
	}

	return nil
}

// Event fields only present in an "event":"def_repo" event
type TrEventDefRepo struct {
	mf_worktree string
}

func extract_keys__def_repo(evt *TrEvent, jm *jmap) (err error) {
	evt.pm_def_repo = &TrEventDefRepo{}

	// "repo" (aka the `repo_id`) was handled in the common fields
	// section, but it was considered optional.  Force it for this
	// event.
	if evt.pmf_repo == nil {
		return fmt.Errorf("key 'repo' is not present in Trace2 event")
	}

	if evt.pm_def_repo.mf_worktree, err = jm.getRequiredString("worktree"); err != nil {
		return err
	}

	return nil
}

// Event fields only present in an "event":"region_enter" event
type TrEventRegionEnter struct {
	mf_nesting   int64
	pmf_category *string // optional category
	pmf_label    *string // optional label
	pmf_msg      *string // optional message
}

func extract_keys__region_enter(evt *TrEvent, jm *jmap) (err error) {
	evt.pm_region_enter = &TrEventRegionEnter{}

	if evt.pm_region_enter.mf_nesting, err = jm.getRequiredInt64("nesting"); err != nil {
		return err
	}

	// "repo" (aka the `repo_id`) was handled in the common fields section

	if evt.pm_region_enter.pmf_category, err = jm.getOptionalString("category"); err != nil {
		return err
	}
	if evt.pm_region_enter.pmf_label, err = jm.getOptionalString("label"); err != nil {
		return err
	}
	if evt.pm_region_enter.pmf_msg, err = jm.getOptionalString("msg"); err != nil {
		return err
	}

	return nil
}

// Event fields only present in an "event":"region_leave" event
type TrEventRegionLeave struct {
	mf_nesting   int64
	pmf_category *string // optional category
	pmf_label    *string // optional label
	pmf_msg      *string // optional message
	// TODO? t_rel
}

func extract_keys__region_leave(evt *TrEvent, jm *jmap) (err error) {
	evt.pm_region_leave = &TrEventRegionLeave{}

	if evt.pm_region_leave.mf_nesting, err = jm.getRequiredInt64("nesting"); err != nil {
		return err
	}

	// "repo" (aka the `repo_id`) was handled in the common fields section

	if evt.pm_region_leave.pmf_category, err = jm.getOptionalString("category"); err != nil {
		return err
	}
	if evt.pm_region_leave.pmf_label, err = jm.getOptionalString("label"); err != nil {
		return err
	}
	if evt.pm_region_leave.pmf_msg, err = jm.getOptionalString("msg"); err != nil {
		return err
	}

	return nil
}

// Event fields only present in a "data" or "data_json" event.
// The value type can be string or int64 for the former.
type TrEventGenericData struct {
	mf_nesting       int64
	mf_category      string
	mf_key           string
	mf_generic_value interface{}
	// TODO? t_rel
	// TODO? t_abs
}

func extract_keys__data(evt *TrEvent, jm *jmap) (err error) {
	evt.pm_generic_data = &TrEventGenericData{}

	// "repo" (aka the `repo_id`) was handled in the common fields section

	if evt.pm_generic_data.mf_nesting, err = jm.getRequiredInt64("nesting"); err != nil {
		return err
	}
	if evt.pm_generic_data.mf_category, err = jm.getRequiredString("category"); err != nil {
		return err
	}
	if evt.pm_generic_data.mf_key, err = jm.getRequiredString("key"); err != nil {
		return err
	}
	if evt.pm_generic_data.mf_generic_value, err = jm.getRequiredStringOrInt64("value"); err != nil {
		return err
	}

	return nil
}

func extract_keys__data_json(evt *TrEvent, jm *jmap) (err error) {
	evt.pm_generic_data = &TrEventGenericData{}

	// "repo" (aka the `repo_id`) was handled in the common fields section

	if evt.pm_generic_data.mf_nesting, err = jm.getRequiredInt64("nesting"); err != nil {
		return err
	}
	if evt.pm_generic_data.mf_category, err = jm.getRequiredString("category"); err != nil {
		return err
	}
	if evt.pm_generic_data.mf_key, err = jm.getRequiredString("key"); err != nil {
		return err
	}
	if evt.pm_generic_data.mf_generic_value, err = jm.getRequired("value"); err != nil {
		return err
	}

	return nil
}

// Event fields only present in an "event":"timer" or "event":"th_timer" events
type TrEventTimer struct {
	mf_category  string
	mf_name      string
	mf_intervals int64
	mf_t_total   float64 // seconds
	mf_t_min     float64 // seconds
	mf_t_max     float64 // seconds
}

func extract_keys__timer(evt *TrEvent, jm *jmap) (err error) {
	evt.pm_timer = &TrEventTimer{}

	if evt.pm_timer.mf_category, err = jm.getRequiredString("category"); err != nil {
		return err
	}
	if evt.pm_timer.mf_name, err = jm.getRequiredString("name"); err != nil {
		return err
	}
	if evt.pm_timer.mf_intervals, err = jm.getRequiredInt64("intervals"); err != nil {
		return err
	}
	if evt.pm_timer.mf_t_total, err = jm.getRequiredFloat64("t_total"); err != nil {
		return err
	}
	if evt.pm_timer.mf_t_min, err = jm.getRequiredFloat64("t_min"); err != nil {
		return err
	}
	if evt.pm_timer.mf_t_max, err = jm.getRequiredFloat64("t_max"); err != nil {
		return err
	}

	return nil
}

// Event fields only present in "event":"counter" and "event":"th_counter" events
type TrEventCounter struct {
	mf_category string
	mf_name     string
	mf_count    int64
}

func extract_keys__counter(evt *TrEvent, jm *jmap) (err error) {
	evt.pm_counter = &TrEventCounter{}

	if evt.pm_counter.mf_category, err = jm.getRequiredString("category"); err != nil {
		return err
	}
	if evt.pm_counter.mf_name, err = jm.getRequiredString("name"); err != nil {
		return err
	}
	if evt.pm_counter.mf_count, err = jm.getRequiredInt64("count"); err != nil {
		return err
	}

	return nil
}
