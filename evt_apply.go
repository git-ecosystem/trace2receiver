package trace2receiver

import (
	"fmt"
	"path/filepath"
	"strings"
)

func evt_apply(tr2 *trace2Dataset, evt *TrEvent) error {
	if evt == nil {
		return nil
	}

	afn, ok := (*applymap)[evt.mf_event]
	if !ok {
		// Unrecognized event type. Ignore since the Trace2 format
		// is allowed to add new event types in the future.
		//
		// TODO Consider debug level logging this.
		return nil
	}
	if afn == nil {
		// Recognized event type, but we want to ignore it since
		// we don't have an apply function for it.
		return nil
	}

	return afn(tr2, evt)
}

type FnApply func(tr2 *trace2Dataset, evt *TrEvent) (err error)
type ApplyMap map[string]FnApply

var applymap *ApplyMap = &ApplyMap{
	"version":      apply__version,
	"start":        apply__start,
	"exit":         apply__atexit, // treat as if "atexit"
	"atexit":       apply__atexit,
	"signal":       apply__signal,
	"error":        apply__error,
	"cmd_path":     apply__cmd_path,
	"cmd_ancestry": apply__cmd_ancestry,
	"cmd_name":     apply__cmd_name,
	"cmd_mode":     apply__cmd_mode,
	"alias":        apply__alias,
	"child_start":  apply__child_start,
	"child_exit":   apply__child_exit,
	"child_ready":  apply__child_ready,
	"thread_start": apply__thread_start,
	"thread_exit":  apply__thread_exit,
	"exec":         apply__exec,
	"exec_result":  apply__exec_result,
	"def_param":    apply__def_param,
	"def_repo":     apply__def_repo,
	"region_enter": apply__region_enter,
	"region_leave": apply__region_leave,
	"data":         apply__data_generic, // "data" can have "string" or "intmax". combine
	"data_json":    apply__data_generic, // with "data_json" using generic interface{}
	"timer":        apply__timer,
	"th_timer":     apply__th_timer,
	"counter":      apply__counter,
	"th_counter":   apply__th_counter,
	"printf":       apply__printf,
	// "too_many_files": nil, // we don't care about this
}

func apply__version(tr2 *trace2Dataset, evt *TrEvent) (err error) {
	tr2.trace2SID = evt.mf_sid

	tr2.process.exeVersion = evt.pm_version.mf_exe
	tr2.process.evtVersion = evt.pm_version.mf_evt

	// For now, set the displayName of the process-level span to
	// "main" (aka the thread name of the main thread).  Later, we'll
	// overwrite it with the normalized/qualified name built from the
	// {command,verb,mode} field values that will arrive later in the
	// Trace2 protocol.
	tr2.process.mainThread.lifetime.displayName = evt.mf_thread

	tr2.process.mainThread.lifetime.startTime = evt.mf_time

	tr2.otelTraceID,
		tr2.process.mainThread.lifetime.selfSpanID,
		tr2.process.mainThread.lifetime.parentSpanID =
		extractIDsfromSID(tr2.trace2SID)

	return nil
}

func apply__start(tr2 *trace2Dataset, evt *TrEvent) (err error) {
	tr2.process.cmdArgv = evt.pm_start.mf_argv

	return nil
}

func apply__atexit(tr2 *trace2Dataset, evt *TrEvent) (err error) {
	// Code shared by "exit" and "atexit". Let the last one win.
	//
	// Defer popping the region stack until EOF.

	tr2.process.mainThread.lifetime.endTime = evt.mf_time
	tr2.process.exeExitCode = evt.pm_atexit.mf_code

	return nil
}

func apply__signal(tr2 *trace2Dataset, evt *TrEvent) (err error) {
	// Signals are highly platform-specific, so we may or may not
	// actually see them in practice.  And we might only get them
	// for some signals.  Trace2 currently only registers for SIGPIPE,
	// for example.  If we do get a signal event, we won't get an
	// exit or atexit event, so some we have to synthesize some data
	// here.
	//
	// Defer poping the region stack until EOF.

	signo := evt.pm_signal.mf_signo

	tr2.process.mainThread.lifetime.endTime = evt.mf_time
	tr2.process.exeExitCode = 128 + signo // Match what the shell does

	return nil
}

func apply__error(tr2 *trace2Dataset, evt *TrEvent) (err error) {
	// The "error" event contains a "msg" string with the actual error
	// message that the user would see on the console.  It also
	// contains a "fmt" string with the sprintf-style format string.
	// The latter is good for grouping similar errors, since it won't
	// have any user-specific strings embedded within it.  The latter
	// is also better for GDPR purposes, since it is less likely to
	// have PII data.
	//
	// Technically, we could see more than one Trace2 error message
	// from the process, but logging an array of strings is problematic,
	// so just remember the first one.

	if len(tr2.process.exeErrorFmt) == 0 {
		tr2.process.exeErrorFmt = evt.pm_error.mf_fmt
		tr2.process.exeErrorMsg = evt.pm_error.mf_msg
	}

	// Check for summary message pattern matches
	apply__summary_message(tr2, evt.pm_error.mf_msg)

	return nil
}

func apply__printf(tr2 *trace2Dataset, evt *TrEvent) (err error) {
	// The "printf" event contains a "msg" string with the actual error
	// message that the user would see on the console.

	// Check for summary message pattern matches
	apply__summary_message(tr2, evt.pm_printf.mf_msg)

	return nil
}

func apply__cmd_path(tr2 *trace2Dataset, evt *TrEvent) (err error) {
	// "cmd_path" is only present in certain circumstances where Git needs
	// to reconstruct the path to currently running EXE by querying the
	// system (see `git_get_exec_path()`).  It uses this, for example, to
	// find the correct version of `libexec/git-core` for subordinate
	// commands.
	//
	// We will ignore this message because it is not always present and
	// we do not need the data.

	return nil
}

func apply__cmd_ancestry(tr2 *trace2Dataset, evt *TrEvent) (err error) {
	// The "cmd_ancestry" event is relatively new and not yet supported
	// on all platforms.
	//
	// (There is an older platform-specific one (for Windows) in a
	// "data_json" event that we might use to fake this.)

	tr2.process.cmdAncestry = evt.pm_cmd_ancestry.mf_ancestry

	return nil
}

func apply__cmd_name(tr2 *trace2Dataset, evt *TrEvent) (err error) {

	// There are some very long-running Git commands that we probably
	// never want to collect telemetry for, such as a background
	// `git fsmonitor--daemon run`.  This is likely to be started
	// automatically without the user knowing it.  It will connect
	// and send telemetry and consume resources in our service
	// until it is stopped, since we cannot send the process span
	// until the process exits.  Meanwhile, we may collect days-worth
	// of region and thread data for the fsmonitor daemon.  This data
	// would be associated with (probably) the first `git status` that
	// (ran days ago and) caused the fsmonitor daemon to be started in
	// the background.  This is just not useful data and will bog
	// down the telemetry service.  So throw an error here and cause
	// the socket to be dropped (safely causing the fsmonitor daemon
	// to silently close the client side and stop trying to send data)
	// and discard the preliminary telemetry for this process.
	//
	// This may be a bit of a hammer because it will also drop
	// telemetry for `git fsmonitor--daemon start` and `... stop`
	// foreground commands, but I'm OK with that.  (Fsmonitor only
	// sends `cmd_name` (aka verb) events but not `cmd_mode` events.
	// We could fix that upstream or pick thru the argv, but it's not
	// worth the effort right now.)
	//
	// TODO There are other long-running services that we should
	// reject here, such as `git daemon` or a future bundle server,
	// but these are not likely to be automatically started and be
	// running on a client machine, so I'll leave that for the
	// future.

	if err := IsFSMonitorDaemon(evt.pm_cmd_name.mf_name); err != nil {
		return err
	}

	tr2.process.cmdVerb = evt.pm_cmd_name.mf_name
	tr2.process.cmdHierarchy = evt.pm_cmd_name.mf_hierarchy

	return nil
}

func apply__cmd_mode(tr2 *trace2Dataset, evt *TrEvent) (err error) {
	tr2.process.cmdMode = evt.pm_cmd_mode.mf_name

	return nil
}

func apply__alias(tr2 *trace2Dataset, evt *TrEvent) (err error) {
	tr2.process.cmdAliasKey = evt.pm_alias.mf_alias
	tr2.process.cmdAliasValue = evt.pm_alias.mf_argv

	return nil
}

// Create a `TrChild` to capture the lifetime of a child process.
//
// (If the child is a Git command, it will independently generate Trace2
// data for itself, but that is not our concern here.)  Here, we
// want to account for the time in the parent process between the
// `fork()`+`exec()` / `CreateProcess()` and the `wait3()` calls.
// This is the "outer" time for the child process.
//
// Note that this will probably be the only data that we will get
// for non-Git processes, such as hook shell scripts, since they
// won't emit telemetry data.
//
// Child-start events contain the name of the thread calling that
// started it and we can infer the current state of the region stack
// for that thread, but we don't know if the child is sync or async.
// That is, will Git block that thread (or the whole process) on
// that child (such as a hook process) or will Git let the child run
// as a mini service, such as LFS, and shut it down later in a
// different region possibly on a different thread?
//
// So we DO NOT want to attach the child span to the thread object
// or the thread's region stack. This also helps when filtering is
// set to `dl:process` and we need to report child spans, but not
// thread/region spans. It also helps because the child process
// itself will inherit the SID of the current process and won't know
// anything about the thread/region spans (nor our child span), so
// the child's process span will appear as a sibling of our child
// span.
func apply__child_start(tr2 *trace2Dataset, evt *TrEvent) (err error) {
	_, ok := tr2.children[evt.pm_child_start.mf_child_id]
	if ok {
		// Duplicate child-start event for this child-id.  This should
		// not happen because Git uses a unique child-id for each child.
		// Ignore this child because we may already have open data for
		// this child-id.
		//
		// TODO log debug warning.
		return nil
	}

	child := &TrChild{
		lifetime: TrSpanEssentials{
			selfSpanID:   tr2.NewSpanID(), // children get a random SpanID
			parentSpanID: tr2.process.mainThread.lifetime.selfSpanID,
			startTime:    evt.mf_time,
			displayName:  evt.pm_child_start.makeChildDisplayName(),
		},
		argv:     evt.pm_child_start.mf_argv,
		pid:      -1,
		exitcode: -1,
	}

	child.class = evt.pm_child_start.mf_child_class
	if child.class == "hook" {
		if evt.pm_child_start.pmf_hook_name != nil {
			child.hookname = *evt.pm_child_start.pmf_hook_name
		} else {
			child.hookname = "??"
		}
	}

	// TODO Do we care about "use_shell" and "cd"?

	tr2.children[evt.pm_child_start.mf_child_id] = child

	return nil
}

// Construct a pretty name for a "child_start" event.
//
// There are several different types of child processes created by
// Git and we can use that to create a custom display name for the
// child span.
func (evt_cs *TrEventChildStart) makeChildDisplayName() string {
	switch evt_cs.mf_child_class {
	case "editor", "pager":
		// We don't care which tools they use, only that the overall
		// performance of the current command will be affected because
		// it is interactive.  (For example, `git commit` is slower
		// than `git commit -m` because the former waits for the user
		// to edit the commit message.)
		return fmt.Sprintf("child(class:%s)", evt_cs.mf_child_class)
	case "hook":
		// An external hook, such as `pre-commit`.  This could do
		// anything and Git has to wait for it to finish, so it could
		// seriously affect (and incorrectly be billed to) the Git
		// command.  For example, `git commit` may be seen to be slow
		// because of an expensive `pre-commit` hook.
		//
		// We do no know if the hook script will be interactive (another
		// source of slow performance), so we just note that a hook was
		// used.
		return fmt.Sprintf("child(hook:%s)", *evt_cs.pmf_hook_name)
	case "git_alias":
		// Alias expansion works by creating a new command line by
		// substituting the alias keyword with the alias value and
		// running a child process with the new command line and
		// waiting for it to finish.
		return "child(alias:git)"
	case "shell_alias":
		// Alias expansion works by creating a new command line by
		// substituting the alias keyword with the alias value and
		// running a child process with the new command line and
		// waiting for it to finish.
		return "child(alias:shell)"
	case "dashed":
		// Some commands have a "space form" and a "dashed form".
		// For example, a `git remote-https ...` command repacks the
		// args and invokes `git-remote-https ...` and waits for it to
		// do the work.
		return fmt.Sprintf("child(dashed:%s)", evt_cs.mf_argv[0].(string))
	case "cred":
		// The child is a credential manager.
		//
		// Unfortunately, the child-start message for the credential
		// manager is a single string rather than a true argv[], so we
		// can't safely extract the "get" or "store" and have to work
		// for it a bit.
		if len(evt_cs.mf_argv) > 1 {
			return fmt.Sprintf("child(cred:%s)", evt_cs.mf_argv[1].(string))
		}
		child_argv0 := evt_cs.mf_argv[0].(string)
		switch {
		case strings.HasSuffix(child_argv0, "get"):
			return "child(cred:get)"
		case strings.HasSuffix(child_argv0, "store"):
			return "child(cred:store)"
		case strings.HasSuffix(child_argv0, "erase"):
			return "child(cred:erase)"
		default:
			return "child(cred:unknown)"
		}
	case "?":
		// Some child processes have not yet been classified in the
		// Git source.  These get a "?" classification.
		return "child(class:unknown)"
	default:
		// There are a variety of other classifications, such as
		// "transport/ssh", "remote-https", "background", "subprocess",
		// and etc.
		//
		// TODO Lets just pass it thru as is (until we decide whether
		// we want to customize it).
		return fmt.Sprintf("child(class:%s)", evt_cs.mf_child_class)
	}
}

func apply__child_exit(tr2 *trace2Dataset, evt *TrEvent) (err error) {
	child, ok := tr2.children[evt.pm_child_exit.mf_child_id]
	if !ok {
		// We saw a "child_exit", but not the corresponding "child_start".
		// Ignore it.
		//
		// TODO log debug warning.
		return nil
	}

	child.lifetime.endTime = evt.mf_time

	child.pid = evt.pm_child_exit.mf_pid
	child.exitcode = evt.pm_child_exit.mf_code

	return nil
}

func apply__child_ready(tr2 *trace2Dataset, evt *TrEvent) (err error) {
	child, ok := tr2.children[evt.pm_child_ready.mf_child_id]
	if !ok {
		// We saw a "child_ready", but not the corresponding "child_start".
		// Ignore it.
		//
		// TODO log debug warning.
		return nil
	}

	child.lifetime.endTime = evt.mf_time

	child.pid = evt.pm_child_ready.mf_pid
	// The child process was pushed into the background by the foreground
	// Git process.  The child's exit-code is unknown, since the child
	// may outlive the parent (and the parent doesn't do any type of `wait()`
	// for it).  Substitute -1 as a placeholder.
	child.exitcode = -1
	child.readystate = evt.pm_child_ready.mf_ready

	return nil
}

// Create a new thread object so that Trace2 regions can be associated
// their actual thread.  Add the thread to the dictionary (using the
// thread name as the key) so that region events can be mapped back to
// the proper thread.  Each thread maintains a region-stack of open
// regions.
//
// Note that we must not add the "main" thread to the thread dictionary
// because we assume it is the owner of all the other threads (and we
// have a pseudo-thread/process for it elsewhere).
func apply__thread_start(tr2 *trace2Dataset, evt *TrEvent) (err error) {

	// Assert tr2.threads[evt.mf_thread] does not already exist.
	_, ok := tr2.threads[evt.mf_thread]
	if ok {
		// Duplicate thread-start event.  This should not happen because
		// Git uses a unique thread-id in the construction of the
		// thread-name.  Ignore this event, since we may already have an
		// open stack for this thread-name.
		//
		// TODO log debug warning.
		return nil
	}

	var th *TrThread = new(TrThread)

	// Thread-start events contain the name of the new thread since
	// the logging call must be called *inside* the thread-proc so
	// that thread-local-storage is properly setup inside of Git.
	//
	// Because of this we do not know which already-running thread
	// that actually started the thread, so we assume "main".
	//
	// So we use the main thread's root SpanID (aka the SpanID
	// associated with the entire process) as the new thread's parent
	// SpanID.
	//
	// Also, we ignore the main thread's open region-stack because we
	// don't know whether the thread will outlive the region; that is,
	// is it a:
	//   {region-start, create threads, do work, join all, region-end}
	// situation or is Git creating a long-running thread to do
	// some background task that might outlast the current tip of the
	// region-stack.

	th.lifetime.selfSpanID = tr2.NewSpanID()
	th.lifetime.parentSpanID = tr2.process.mainThread.lifetime.selfSpanID
	th.lifetime.startTime = evt.mf_time
	th.lifetime.displayName = evt.mf_thread

	tr2.threads[evt.mf_thread] = th

	return nil
}

func apply__thread_exit(tr2 *trace2Dataset, evt *TrEvent) (err error) {
	th, ok := tr2.threads[evt.mf_thread]
	if !ok {
		// We saw a "thread_exit" but not the corresponding "thread_start".
		// Ignore it.
		//
		// TODO log debug warning.
		return nil
	}

	// Git should have closed all open regions on the region-stack for
	// this thread, but force close any unclosed ones.  Set their end
	// times to that of the thead.
	tr2.popAllRegionStack(th, evt.mf_time)

	th.lifetime.endTime = evt.mf_time

	return nil
}

// The Git process called one of the `exec()` variants to replace its process
// with a new one.  This doesn't happen very often, but there's not much for
// us to do.  On Unix, in theory, we won't see an exit/atexit event for the
// current process and it should abruptly drop the connection, so we can let
// the EOF handling take care of the details.
//
// If the replacement command is a Git command, it may start a fresh telemetry
// stream.
//
// On Windows, everything is different, where exec() behaves like a
// blocking child process, so we might get exit/atexit events.
//
// Use the same span-parenting rules as we do for child_start.
func apply__exec(tr2 *trace2Dataset, evt *TrEvent) (err error) {
	_, ok := tr2.exec[evt.pm_exec.mf_exec_id]
	if ok {
		// Duplicate "exec" event for this exec-id.  This should
		// not happen because Git uses a unique exec-id for each exec.
		// (And normally, exec() doesn't return.)
		//
		// Ignore this event because we may already have open data for
		// this exec-id.
		//
		// TODO log debug warning.
		return nil
	}

	exec := &TrExec{
		lifetime: TrSpanEssentials{
			selfSpanID:   tr2.NewSpanID(), // children get a random SpanID
			parentSpanID: tr2.process.mainThread.lifetime.selfSpanID,
			startTime:    evt.mf_time,
			displayName:  evt.pm_exec.makeExecDisplayName(),
		},
		argv:     evt.pm_exec.mf_argv,
		exitcode: -1,
	}

	if evt.pm_exec.pmf_exe != nil {
		exec.exe = *evt.pm_exec.pmf_exe
	}

	tr2.exec[evt.pm_exec.mf_exec_id] = exec

	return nil
}

// Construct a pretty name for an "exec" event.
func (evt_ex *TrEventExec) makeExecDisplayName() string {
	if evt_ex.pmf_exe != nil {
		basename := filepath.Base(*evt_ex.pmf_exe)
		// TODO verify or fixup weird edge cases
		return fmt.Sprintf("exec(%s)", basename)
	}

	if len(evt_ex.mf_argv) > 0 {
		basename := filepath.Base(evt_ex.mf_argv[0].(string))
		// TODO verify or fixup weird edge cases
		return fmt.Sprintf("exec(%s)", basename)
	}

	return "exec(?)"
}

// We only get an "exec_result" event if the `exec()` failed.
// (O)
func apply__exec_result(tr2 *trace2Dataset, evt *TrEvent) (err error) {
	exec, ok := tr2.exec[evt.pm_exec_result.mf_exec_id]
	if !ok {
		// We saw a "exec_result", but not the corresponding "exec"
		// start event.  Ignore it.
		//
		// TODO log debug warning.
		return nil
	}

	exec.lifetime.endTime = evt.mf_time
	exec.exitcode = evt.pm_exec_result.mf_code

	return nil
}

// Add this key/value pair to the set of parameters.
//
// Note: A recent (2022/08) change to Git causes it to enumerate over
// all scopes for each matching config setting, rather than just the
// active one (and it is unclear if we get duplicate keys in scope-order),
// so we decode the scope and remember the one with the highest priority.
//
// We get values for both Git config settings and special environment
// variables.  The latter don't have a scope.
func apply__def_param(tr2 *trace2Dataset, evt *TrEvent) (err error) {
	key := evt.pm_def_param.mf_param
	valNew := evt.pm_def_param.mf_value
	priNew := get_scope_priority(evt.pm_def_param.pmf_scope)

	_, havePrevVal := tr2.process.paramSetValues[key]
	priCur, havePrevPri := tr2.process.paramSetPriorities[key]

	if havePrevVal && havePrevPri && priNew < priCur {
		// We already have a value for this key with a higher
		// priority, so ignore this value.
		// Note: When priorities are equal, we accept the new value
		// to match Git's "last one wins" behavior.
		return nil
	}

	tr2.process.paramSetValues[key] = valNew
	tr2.process.paramSetPriorities[key] = priNew

	// We DO NOT try to lookup the filtering keys at this point
	// because we don't know if this is final (highest priority)
	// param, so we cannot try to short circut for "dl:drop" yet.

	return nil
}

func get_scope_priority(scope *string) int {
	if scope == nil {
		// EnvVars don't have a scope.  Also, the scope field is optional.
		// So assume the last one wins.
		return 100
	}

	switch *scope {
	case "system":
		return 1
	case "global":
		return 2
	case "local":
		return 3
	case "worktree":
		return 4
	case "command":
		return 5
	case "submodule":
		return 6
	case "unknown":
		return 7
	default:
		// This should not happen.  We could assert or warn on this.
		return 99
	}
}

func apply__def_repo(tr2 *trace2Dataset, evt *TrEvent) (err error) {
	repoId := *evt.pmf_repo
	tr2.process.repoSet[repoId] = evt.pm_def_repo.mf_worktree

	return nil
}

// Open a region and push it onto the per-thread region-stack.
func apply__region_enter(tr2 *trace2Dataset, evt *TrEvent) (err error) {
	th, ok := tr2.lookupThread(evt.mf_thread)
	if !ok || th == nil {
		// We did not get the thread-start event for the current thread.
		// Since regions are stored on a thread-specific stack, we don't
		// have a region-stack to push this region.
		//
		// TODO For now, ignore this region-enter (and the corresponding
		// region-leave).  We might want to try synthesizing a thread
		// using this thread-name if we see this in the wild.
		//
		// TODO log debug warning.
		return nil
	}

	// Note that the "category" and "label" fields are optional (and
	// are not required to match the values in the corresponding
	// "region_leave" event.  The Trace2 format gives the illusion of
	// being balanced with named events, but in reality we just have
	// an per-thread stack of open regions.  Region "nesting" levels
	// are 1-based values and to help identify the region's depth.
	// (And it helps _PERF format print the ".." prefix.)
	//
	// Therefore, a region with nesting level k when pushed onto the
	// top of the stack, should be at position regionStack[k-1].
	if int64(len(th.regionStack)) != evt.pm_region_enter.mf_nesting-1 {
		// Ignore the region if this doesn't match up properly.
		//
		// TODO log debug warning.
		return nil
	}

	r := &TrRegion{
		lifetime: TrSpanEssentials{
			selfSpanID:   tr2.NewSpanID(), // regions get a random SpanID
			parentSpanID: th.lookupTopParentSpanID(),
			startTime:    evt.mf_time,
			displayName:  evt.pm_region_enter.makeRegionDisplayName(),
		},
	}

	r.nestingLevel = evt.pm_region_enter.mf_nesting
	if evt.pm_region_enter.pmf_msg != nil {
		r.message = *evt.pm_region_enter.pmf_msg
	}

	// Store category and label for summary matching
	if evt.pm_region_enter.pmf_category != nil {
		r.category = *evt.pm_region_enter.pmf_category
	}
	if evt.pm_region_enter.pmf_label != nil {
		r.label = *evt.pm_region_enter.pmf_label
	}

	// Regions are associated with an optional repo-id that defines the
	// worktree.
	if evt.pmf_repo == nil {
		// Currently, Git does not support multiple in-proc repositories, so
		// the repo-id is (or should always be) 1.  Let's assume the primary
		// repository for any regions that didn't explicitly set it.
		r.repoId = 1
	} else {
		r.repoId = *evt.pmf_repo
	}

	th.regionStack = append(th.regionStack, r)

	return nil
}

// Create a display name for the region.
func (evt_re *TrEventRegionEnter) makeRegionDisplayName() string {
	var c string
	var l string

	// Technically, the category and label fields are optional,
	// but are rarely ever omitted.

	if evt_re.pmf_category != nil {
		c = normalizeForRegionDisplayName(*evt_re.pmf_category)
	} else {
		c = "C"
	}

	if evt_re.pmf_label != nil {
		l = normalizeForRegionDisplayName(*evt_re.pmf_label)
	} else {
		l = "L"
	}

	return fmt.Sprintf("region(%s,%s)", c, l)
}

// Trace2 region-enter events contain a "category" and "label"
// field.  Both are somewhat free form.  (That wasn't the intent,
// that is how they have evolved.)  Scrub them a little to help
// make a display name that is easy to search for in the database.
func normalizeForRegionDisplayName(value string) string {
	value = strings.Replace(value, " ", "_", -1)
	value = strings.Replace(value, "-", "_", -1)
	value = strings.Replace(value, ".", "_", -1)
	value = strings.Replace(value, ",", "_", -1)
	value = strings.Replace(value, ":", "_", -1)
	value = strings.Replace(value, "(", "_", -1)
	value = strings.Replace(value, ")", "_", -1)
	value = strings.ToLower(value)

	return value
}

// Close the open region, pop it from the region-stack for the
// current thread, and move it to the vector of completed regions.
func apply__region_leave(tr2 *trace2Dataset, evt *TrEvent) (err error) {
	th, ok := tr2.lookupThread(evt.mf_thread)
	if !ok || th == nil {
		// We did not get the thread-start event for the current thread,
		// Since regions are stored on a thread-specific region stack, we
		// have to assume that the corresponding region-start was ignored.
		//
		// TODO log debug warning.
		return nil
	}

	rCount := len(th.regionStack)
	if rCount == 0 {
		// The per-thread region-stack is empty, so we either missed a
		// region-start or received too many region-leaves.  Ignore it.
		//
		// TODO log debug warning.
		return nil
	}

	r := th.regionStack[rCount-1]

	// Because the "category" and "label" filds are optional (and there is
	// no requirement that the spelling here match that of the corresponding
	// "region_enter"), we cannot verify that this event is properly matched
	// up.
	//
	// The only thing we can do is verify that the nesting level makes sense.
	// The open region that we are about to pop off of the stack should have
	// the same nesting level as the current event.
	if r.nestingLevel != evt.pm_region_leave.mf_nesting {
		// TODO log debug warning.
		return nil
	}

	r.lifetime.endTime = evt.mf_time

	// Apply summary region rules
	apply__summary_region(tr2, r)

	// TODO The region-leave event has optional category and label fields.
	// These almost always match the values on the region-enter, but they
	// don't have to.  Consider overriding them or somehow picking the
	// "better" pair to keep in the region object.

	// TODO Likewise, the region-leave has an optional message field.
	// If set, should it override any message from the region-start?

	// I'm not going to set r.repoID here.  Let's assume that the repo-id
	// on this "region_leave" event matches the value that we saw on the
	// "region_enter" event.

	tr2.completedRegions = append(tr2.completedRegions, r)
	th.regionStack = th.regionStack[:rCount-1]

	return nil
}

func apply__data_generic(tr2 *trace2Dataset, evt *TrEvent) (err error) {
	// Nesting levels are 1-based values.
	//
	// Data events with nesting level 1 refer to the process rather
	// than a thread-specific region.
	//
	// Data events with nesting level n are logically contained with
	// the region with level n-1 (in the current thread).  And because
	// of the 1-based value in the region itself, the region with
	// nesting level n-1 is stored at regionStack[n-2] (assuming the
	// Git process properly sets things up).

	if evt.pm_generic_data.mf_nesting <= 1 {
		tr2.process.setGenericDataValue(evt.pm_generic_data.mf_category,
			evt.pm_generic_data.mf_key, evt.pm_generic_data.mf_generic_value)
		return nil
	}

	// Find the associated thread and region for this data event.
	// Ignore the event if we can't find where to attach it.
	th, ok := tr2.lookupThread(evt.mf_thread)
	if !ok || th == nil {
		// TODO log debug warning.
		return nil
	}
	rWant := evt.pm_generic_data.mf_nesting - 2
	if int64(len(th.regionStack)) < rWant {
		// TODO log debug warning.
		return nil
	}
	r := th.regionStack[rWant]
	if r.nestingLevel != evt.pm_generic_data.mf_nesting-1 {
		// TODO log debug warning.
		return nil
	}

	r.setGenericDataValue(evt.pm_generic_data.mf_category,
		evt.pm_generic_data.mf_key, evt.pm_generic_data.mf_generic_value)

	return nil
}

// Set data[<category>][<key>] = <value>
func (p *TrProcess) setGenericDataValue(category string, key string, value interface{}) {
	if p.dataValues == nil {
		p.dataValues = make(map[string]map[string]interface{})
	}
	kmap, ok := p.dataValues[category]
	if !ok {
		kmap = make(map[string]interface{})
		p.dataValues[category] = kmap
	}
	kmap[key] = value
}

// Set data[<category>][<key>] = <value>
func (r *TrRegion) setGenericDataValue(category string, key string, value interface{}) {
	if r.dataValues == nil {
		r.dataValues = make(map[string]map[string]interface{})
	}
	kmap, ok := r.dataValues[category]
	if !ok {
		kmap = make(map[string]interface{})
		r.dataValues[category] = kmap
	}
	kmap[key] = value
}

func apply__timer(tr2 *trace2Dataset, evt *TrEvent) (err error) {
	if tr2.process.timers == nil {
		tr2.process.timers = make(map[string]map[string]TrStopwatchTimer)
	}
	nmap, ok := tr2.process.timers[evt.pm_timer.mf_category]
	if !ok {
		nmap = make(map[string]TrStopwatchTimer)
		tr2.process.timers[evt.pm_timer.mf_category] = nmap
	}
	nmap[evt.pm_timer.mf_name] = TrStopwatchTimer{
		Intervals: evt.pm_timer.mf_intervals,
		Total_sec: evt.pm_timer.mf_t_total,
		Min_sec:   evt.pm_timer.mf_t_min,
		Max_sec:   evt.pm_timer.mf_t_max,
	}

	return nil
}

func apply__th_timer(tr2 *trace2Dataset, evt *TrEvent) (err error) {
	th, ok := tr2.lookupThread(evt.mf_thread)
	if !ok || th == nil {
		// TODO log debug warning.
		return nil
	}

	if th.timers == nil {
		th.timers = make(map[string]map[string]TrStopwatchTimer)
	}
	nmap, ok := th.timers[evt.pm_timer.mf_category]
	if !ok {
		nmap = make(map[string]TrStopwatchTimer)
		th.timers[evt.pm_timer.mf_category] = nmap
	}
	nmap[evt.pm_timer.mf_name] = TrStopwatchTimer{
		Intervals: evt.pm_timer.mf_intervals,
		Total_sec: evt.pm_timer.mf_t_total,
		Min_sec:   evt.pm_timer.mf_t_min,
		Max_sec:   evt.pm_timer.mf_t_max,
	}

	return nil
}

func apply__counter(tr2 *trace2Dataset, evt *TrEvent) (err error) {
	if tr2.process.counters == nil {
		tr2.process.counters = make(map[string]map[string]int64)
	}
	nmap, ok := tr2.process.counters[evt.pm_counter.mf_category]
	if !ok {
		nmap = make(map[string]int64)
		tr2.process.counters[evt.pm_counter.mf_category] = nmap
	}

	nmap[evt.pm_counter.mf_name] = evt.pm_counter.mf_count

	return nil
}
func apply__th_counter(tr2 *trace2Dataset, evt *TrEvent) (err error) {
	th, ok := tr2.lookupThread(evt.mf_thread)
	if !ok || th == nil {
		// TODO log debug warning.
		return nil
	}

	if th.counters == nil {
		th.counters = make(map[string]map[string]int64)
	}
	nmap, ok := th.counters[evt.pm_counter.mf_category]
	if !ok {
		nmap = make(map[string]int64)
		th.counters[evt.pm_counter.mf_category] = nmap
	}

	nmap[evt.pm_counter.mf_name] = evt.pm_counter.mf_count

	return nil
}
