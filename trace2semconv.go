package trace2receiver

import "go.opentelemetry.io/otel/attribute"

// This file contains semantic conventions for Trace2 reporting.

const (
	// Value of the `service.namespace` key that we inject into
	// all resourceAttributes.
	Trace2ServiceNamespace = "trace2"

	// Value of the `instrumentation.name` or `instrumentationlibrary.name`
	// key that we inject into the resourceAttributes.  (The actual spelling
	// of this key varies it seems.)
	Trace2InstrumentationName = "trace2receiver"
)

// TODO Compare this with the stock `semconv` package for some of
// process keys.

const (
	// The Trace2 SID of the process.  This is the complete SID with
	// zero or more slashes describing the SIDs of any Trace2-aware
	// parent processes.
	Trace2CmdSid = attribute.Key("trace2.cmd.sid")

	// The complete command line args of the process.
	Trace2CmdArgv = attribute.Key("trace2.cmd.argv")

	// The version string of the process executable as reported in the
	// Trace2 "version" event.
	Trace2CmdVersion = attribute.Key("trace2.cmd.version")

	// The command's exit code.  Zero if it completed without error.
	// If this process was signalled, this should be 128+signo.
	Trace2CmdExitCode = attribute.Key("trace2.cmd.exit_code")

	// The base filename of the process executable (with the pathname and
	// `.exe` suffix stripped off), for example `git` or `git-remote-https`.
	Trace2CmdName = attribute.Key("trace2.cmd.name")

	// The executable name and verb isolated from the command line
	// with normalized formatting.  For example `git checkout` should
	// be reported as `git:checkout`.  For commands that do not have
	// a verb, this should just be the name of the executable.
	Trace2CmdNameVerb = attribute.Key("trace2.cmd.name_verb")

	// The executable name, verb and command mode combined with
	// normalized formatting.  For example, `git checkout -- <pathname>`
	// is different from `git checkout <branchname>`.  These should be
	// reported as `git:checkout#path` or `git:checkout#branch`.  For
	// commands that do not have a mode, this should just be the verb.
	Trace2CmdNameVerbMode = attribute.Key("trace2.cmd.name_verb_mode")

	// The verb hierarchy for the command as reported by Git itself.
	// For example when `git index-pack` is launched by `git fetch`,
	// the child process will report a verb of `index-pack` and a
	// hierarchy of `fetch/index-pack`.
	Trace2CmdHierarchy = attribute.Key("trace2.cmd.hierarchy")

	// The format string of one error message from the command.
	Trace2CmdErrFmt = attribute.Key("trace2.cmd.error.format")
	Trace2CmdErrMsg = attribute.Key("trace2.cmd.error.message")

	Trace2CmdAliasKey   = attribute.Key("trace2.cmd.alias.key")
	Trace2CmdAliasValue = attribute.Key("trace2.cmd.alias.value")

	// Optional process hierarchy that invoked this Git command.
	// Usually contains things like "bash" and "sshd".  This data
	// is read from "/proc" on Linux, for example.  It may be
	// truncated, but contain enough entries to give some crude
	// context.
	//
	// Type: array of string
	Trace2CmdAncestry = attribute.Key("trace2.cmd.ancestry")

	// Trace2 classification of the span.  For example: "process",
	// "thread", "child", or "region".
	//
	// Type: string
	Trace2SpanType = attribute.Key("trace2.span.type")

	Trace2ChildPid        = attribute.Key("trace2.child.pid")
	Trace2ChildExitCode   = attribute.Key("trace2.child.exitcode")
	Trace2ChildArgv       = attribute.Key("trace2.child.argv")
	Trace2ChildClass      = attribute.Key("trace2.child.class")
	Trace2ChildHookName   = attribute.Key("trace2.child.hook")
	Trace2ChildReadyState = attribute.Key("trace2.child.ready")

	Trace2RegionMessage = attribute.Key("trace2.region.message")
	Trace2RegionNesting = attribute.Key("trace2.region.nesting")
	Trace2RegionRepoId  = attribute.Key("trace2.region.repoid")
	Trace2RegionData    = attribute.Key("trace2.region.data")

	Trace2ExecExe      = attribute.Key("trace2.exec.exe")
	Trace2ExecArgv     = attribute.Key("trace2.exec.argv")
	Trace2ExecExitCode = attribute.Key("trace2.exec.exitcode")

	Trace2RepoSet  = attribute.Key("trace2.repo.set")
	Trace2ParamSet = attribute.Key("trace2.param.set")

	Trace2RepoNickname = attribute.Key("trace2.repo.nickname")

	Trace2ProcessData     = attribute.Key("trace2.process.data")
	Trace2ProcessTimers   = attribute.Key("trace2.process.timers")
	Trace2ProcessCounters = attribute.Key("trace2.process.counters")
	Trace2ProcessCustom   = attribute.Key("trace2.process.summary")

	Trace2ThreadTimers   = attribute.Key("trace2.thread.timers")
	Trace2ThreadCounters = attribute.Key("trace2.thread.counters")

	Trace2GoArch = attribute.Key("trace2.machine.arch")
	Trace2GoOS   = attribute.Key("trace2.machine.os")

	Trace2PiiHostname = attribute.Key("trace2.pii.hostname")
	Trace2PiiUsername = attribute.Key("trace2.pii.username")
)
