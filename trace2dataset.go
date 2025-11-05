package trace2receiver

import (
	crand "crypto/rand"
	"encoding/binary"
	"math/rand"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// A dataset captures all of the Trace2 event data from a single
// process.
//
// We assume that all Trace2 events are from the same process
// and therefore have the same Trace2 SID.  (We do not support
// processing a multi-process trace file currently; that should
// be done at a higher level probably.)
//
// At this layer we do not know when the command has finished.
// It is usually when the "atexit" event is seen, but the process
// may terminate earlier (e.g. if it gets killed or crashes).
// The socket/named pipe reader will know if/when it gets an EOF
// from the client process, so we wait for it to tell us that the
// process is really finished.
type trace2Dataset struct {
	rcvr_base *Rcvr_Base

	// Unique dataset id for this dataset.  We'll use this in our
	// debug logging to disambiguate messages (and associate them
	// back to the worker thread).
	datasetId uint64

	// Did we see at least one Trace2 event from the client?
	sawData bool

	randSource *rand.Rand

	otelTraceID [16]byte

	// The Trace2 SID for the command.  Technically, this should be
	// just attached to the process span, since it is a process-level
	// concept, but it is useful to have it on all of the spans that
	// we generate for database queries.  This has slightly different
	// scope than the OTEL TraceID (such as when we're not the top-level
	// Git command).
	trace2SID string

	// Application-layer data for the main process and thread.  Span
	// data for the main thread is not present in the `threads[]` map.
	process TrProcess

	// Map of thread data for non-main threads.
	threads map[string]*TrThread

	// The set of child processes spawned by the current process.
	children map[int64]*TrChild

	// The set of exec()-style replacement processes spawned.
	exec map[int64]*TrExec

	// The set of completed regions (across any thread).
	completedRegions []*TrRegion

	// Dictionary of optional PII data that we want to include in
	// the process data.  This is only used when bits are enabled
	// in the `receivers.trace2receiver.pii.*` are set in config.yml.
	// These fields maybe GDPR-restricted, so use this at your own risk.
	// Map from the SemConv keys to the data value.
	pii map[string]string
}

// Data associated with the entire process.
type TrProcess struct {
	mainThread TrThread

	// The version string from the Git command
	exeVersion string
	// The Trace2 file format version
	evtVersion string

	// The Argv passed to the command from the system
	cmdArgv []interface{}

	// The command name (aka verb), such as `checkout` or `fetch`
	// extracted by Git from somewhere within Argv.
	cmdVerb string
	// The concise verb hierarchy.
	cmdHierarchy string
	// The command mode (set for commands like `checkout` that
	// have multiple uses, like branch switching to single file
	// restore).
	cmdMode string

	// When set, the alias name
	cmdAliasKey string
	// When set, the alias expansion
	cmdAliasValue []interface{}

	// When set, the array of parent processes extraced from /proc
	cmdAncestry []interface{}

	// The exit code for the main process
	exeExitCode int64
	// Arbitrarily pick one error messages from the process
	exeErrorMsg string
	exeErrorFmt string

	// Map repo-ids to worktree from `def_repo` events.
	// We use a map rather than an array because we are
	// not guaranteed the order of the events.
	repoSet map[int64]string

	// The collapsed set of advertised parameters from the
	// `def_param` events.
	paramSetValues     map[string]string
	paramSetPriorities map[string]int

	// Collect the values of all process-level "data" and "data_json"
	// events using a "data[<category>][<key>] = <value>" model.
	// We assume that Git does not repeat (category,key) pairs, or
	// rather, we just remember the last value.
	dataValues map[string]map[string]interface{}

	// Process-level stopwatch timers
	timers map[string]map[string]TrStopwatchTimer

	// Process-level global counters
	counters map[string]map[string]int64

	qualifiedNames QualifiedNames
}

type QualifiedNames struct {
	exe         string
	exeVerb     string
	exeVerbMode string
}

// The `TrThread` structure captures the lifetime of a
// thread.
//
// Each thread (including the "main" thread) contains a
// `TrSpanEssentials` to document the life of the thread
// or process) and a "region stack" to
// capture in-progress Trace2 Regions as they are being
// reported by the client.
//
// Yes, each thread needs its own region stack because
// regions are per-thread.
//
// When we start a region-start event, we push a new frame
// on to the region stack.  When we see the (hopefully,
// corresponding) region-leave event, we "complete" the
// region, pop it off of the region stack, and move it to
// the "completed regions" array for later reporting.
type TrThread struct {
	// Describes the lifetime of the thread.
	lifetime TrSpanEssentials

	// Stack of open regions on this thread.
	regionStack []*TrRegion

	// Per-thread timers[<category>][<name>]
	timers map[string]map[string]TrStopwatchTimer

	// Per-thread counters[<category>][<name>]
	counters map[string]map[string]int64
}

// The `TrChild` structure captures the lifetime of a
// child process spawned by the current Git process.
// This is the "outer" time from the exec() to wait3()
// as observed by the invoking Git process.
//
// This is independent of any telemetry that the child
// process itself may emit.
type TrChild struct {
	lifetime TrSpanEssentials

	argv       []interface{}
	pid        int64
	exitcode   int64
	readystate string
	class      string
	hookname   string
}

type TrExec struct {
	lifetime TrSpanEssentials

	argv     []interface{}
	exe      string
	exitcode int64
}

type TrRegion struct {
	lifetime TrSpanEssentials

	repoId       int64
	nestingLevel int64
	message      string

	// Collect the values of all region-level "data" and "data_json"
	// events using a "data[<category>][<key>] = <value>" model.
	// We assume that Git does not repeat (category,key) pairs, or
	// rather, we just remember the last value.
	dataValues map[string]map[string]interface{}
}

type TrStopwatchTimer struct {
	Intervals int64   `json:"intervals"`
	Total_sec float64 `json:"total_sec"`
	Min_sec   float64 `json:"min_sec"`
	Max_sec   float64 `json:"max_sec"`
}

// `TrSpanEssentials` is a generic term to describe a chunk of
// time doing something.  It may refer to the lifetime of
// the whole process, the lifetime of a thread, a Trace2
// region, the lifetime of a child process, and etc.
type TrSpanEssentials struct {
	selfSpanID   [8]byte
	parentSpanID [8]byte
	startTime    time.Time
	endTime      time.Time
	displayName  string
}

var mux sync.Mutex
var datasetId uint64

func makeDatasetId() uint64 {
	mux.Lock()
	dsid := datasetId
	datasetId++
	mux.Unlock()

	return dsid
}

func NewTrace2Dataset(rcvr_base *Rcvr_Base) *trace2Dataset {
	var tr2 *trace2Dataset = new(trace2Dataset)

	tr2.rcvr_base = rcvr_base
	tr2.datasetId = makeDatasetId()

	var rngSeed int64
	_ = binary.Read(crand.Reader, binary.LittleEndian, &rngSeed)
	tr2.randSource = rand.New(rand.NewSource(rngSeed))

	tr2.threads = make(map[string]*TrThread)
	tr2.children = make(map[int64]*TrChild)

	tr2.process.repoSet = make(map[int64]string)
	tr2.process.paramSetValues = make(map[string]string)
	tr2.process.paramSetPriorities = make(map[string]int)

	tr2.pii = make(map[string]string)
	tr2.exec = make(map[int64]*TrExec)

	return tr2
}

func (tr2 *trace2Dataset) NewSpanID() [8]byte {
	var spid [8]byte

	tr2.randSource.Read(spid[:])

	return spid
}

func (tr2 *trace2Dataset) popRegionStack(th *TrThread, t time.Time) {
	rCount := len(th.regionStack)
	r := th.regionStack[rCount-1]

	r.lifetime.endTime = t

	tr2.completedRegions = append(tr2.completedRegions, r)
	th.regionStack = th.regionStack[:rCount-1]
}

func (tr2 *trace2Dataset) popAllRegionStack(th *TrThread, t time.Time) {
	for len(th.regionStack) > 0 {
		tr2.popRegionStack(th, t)
	}
}

func (tr2 *trace2Dataset) lookupThread(threadName string) (*TrThread, bool) {
	if threadName == "main" {
		return &tr2.process.mainThread, true
	} else {
		th, ok := tr2.threads[threadName]
		return th, ok
	}
}

// Return the SpanID of the top of the region stack for this
// thread or the SpanID of the thread itself.
func (th *TrThread) lookupTopParentSpanID() (parent [8]byte) {
	if len(th.regionStack) == 0 {
		copy(parent[:], th.lifetime.selfSpanID[:])
	} else {
		copy(parent[:], th.regionStack[len(th.regionStack)-1].lifetime.selfSpanID[:])
	}

	return parent
}

// Fixup any incomplete work units and set the spelling of
// the various qualified names for the EXE.
//
// We should only have incomplete work units if Git dies, crashes
// or received a signal and did not get a chance to pop all of
// the region stack frames in all threads before emitting the
// "atexit" event.
//
// Part of this is just to get closure on any in-progress
// work.  Part of this is to not generate ill-formed OTEL
// spans (with negative durations) that might cause problems
// downstream.
//
// Return false if we did not receive sufficient information
// from the Git client to emit telemetry for this dataset.
func (tr2 *trace2Dataset) prepareDataset() bool {

	// If no command line, we didn't see the "start" event, so we
	// don't know anything about the command, so ignore it.
	if len(tr2.process.cmdArgv) == 0 {
		return false
	}

	now := time.Now()

	for _, child := range tr2.children {
		if child.lifetime.isIncomplete() {
			child.lifetime.endTime = now
			child.pid = -1
			child.exitcode = -1
		}
	}

	for _, th := range tr2.threads {
		if th.lifetime.isIncomplete() {
			tr2.popAllRegionStack(th, now)
			th.lifetime.endTime = now
		}
	}

	// The main thread is special, both because it is not in the thread
	// vector and because we normally expect "exit" and "atexit" events
	// and we deferred the region stack cleanup.
	tr2.popAllRegionStack(&tr2.process.mainThread, now)

	if tr2.process.mainThread.lifetime.isIncomplete() {
		tr2.process.mainThread.lifetime.endTime = now
		tr2.process.exeExitCode = -1
	}

	// Compute normalized <exe>, <exe>[:<verb>], and <exe>[:<verb>][#<mode>]
	tr2.setQualifiedExeName()
	tr2.setQualifiedExeVerbName()
	tr2.setQualifiedExeVerbModeName()

	// Update the display name of the process-level work unit to be
	// this normalized/qualified name so that the process-level span
	// will be more useful than just the name of the "main" thread.
	tr2.process.mainThread.lifetime.displayName = tr2.process.qualifiedNames.exeVerbMode

	return true
}

// A span (region, thread, etc.) is said to be "incomplete"
// (meaning unclosed) if the end time is still zero.  This is
// possible if the corresponding `endRegion()` or `endThread()`
// or whatever has not been called.  This can happen if the process
// dies or crashes and the data stream is prematurely terminated,
// for example.
func (se *TrSpanEssentials) isIncomplete() bool {
	return se.endTime.IsZero()
}

// Set the "qualified exe base name" from Argv.
//
// Omit all platform-specific pathname quirks, like Windows
// drive letters, forward and back slashes, and `.exe` extensions.
//
// The expected result is `git` or `git-remote-https`, for example.
func (tr2 *trace2Dataset) setQualifiedExeName() {
	var argv_0 string = tr2.process.cmdArgv[0].(string)
	var exeName string = filepath.Base(argv_0)
	var ext string = filepath.Ext(exeName)
	if len(ext) > 0 {
		switch strings.ToLower(ext) {
		case ".exe":
			exeName = strings.TrimSuffix(exeName, ext)
		default:
			// Don't strip unknown suffixes.
		}
	}

	tr2.process.qualifiedNames.exe = exeName
}

// Set the "qualified exe + verb name"
//
// The `git` executable assumes a top-level command (aka verb),
// for example `git checkout` or `git fetch`.  Both use the same
// executable, but do completely different things.
//
// This is incontrast to specialized executables, such as
// `git-remote-https` that do not have a "verb" argument.
//
// Format this as "<exe>[:<verb>]" to disambiguate.
func (tr2 *trace2Dataset) setQualifiedExeVerbName() {
	tr2.process.qualifiedNames.exeVerb = tr2.process.qualifiedNames.exe

	if len(tr2.process.cmdVerb) == 0 {
		return
	}

	tr2.process.qualifiedNames.exeVerb += ":"

	switch tr2.process.cmdVerb {
	case "_run_dashed_":
		// We expect something like ["git", "remote-https", "origin", ...]
		// where Git parses the command line and thinks that it should
		// invoke a "dashed form" as a sub-process and just wait for it to
		// do the work.
		//
		// This command line manipulation gets a little muddy when command
		// aliases are involved (where Git will try to "dash run" the alias
		// name, fail, and then apply alias value, and try to invoke that).
		if len(tr2.process.cmdArgv) > 1 {
			tr2.process.qualifiedNames.exeVerb += tr2.process.cmdArgv[1].(string)
		} else {
			// Quietly fail if argv is not long enough.  This should not happen
			// in real life, since Git uses ["git", "remote-http"] to compose
			// ["git-remote-https"].  We guard it here for the test suite.
			tr2.process.qualifiedNames.exeVerb += "_run_dashed_"
		}
	case "_run_git_alias_":
		// The current Git command is trying to expand an alias and invoke
		// it as a child process.  We cannot predict what command that will
		// eventually be, so keep the pseudo-verb marker.
		//
		// At some point we could just omit this process from the trace, but
		// it is a member of the SID vector, so it would leave a hole in our
		// parent/child process graph in the trace/span.
		tr2.process.qualifiedNames.exeVerb += tr2.process.cmdVerb
	case "_query_":
		// The current Git command only needs to lookup a config setting
		// or something.  There are several commands, such as
		// `git --exec-path` and `git --html-path`, that just print a
		// constant and exit.  These `--value` arguments take the place of the
		// normal (non-dashed) verb.  (It is not safe to assume Argv[1] is the
		// name of the specific value, for example `git -C . --exe-path`, so
		// just keep the pseudo-verb.)
		tr2.process.qualifiedNames.exeVerb += tr2.process.cmdVerb
	case "_run_shell_alias_":
		// The current Git command wants to run a non-builtin shell command.
		// And like the other pseudo-verbs, Git will invoke it and just wait
		// for it to exit.
		tr2.process.qualifiedNames.exeVerb += tr2.process.cmdVerb
	default:
		// We have a non-pseudo verb, like `git checkout`.  (We cannot assume
		// Argv[1] because the actual command might be something like
		// `git -C . checkout`.)
		tr2.process.qualifiedNames.exeVerb += tr2.process.cmdVerb
	}
}

// Set the qualified "name + verb + mode".
//
// Some Git verbs have multiple meanings, such as `git checkout <branch>`
// vs `git checkout <pathname>`.  One switches branches and one refreshes
// a single file.  It is not meaningful to compare perf times between
// two different modes.
//
// Format this as "<exe>[:<verb>][#<mode>]" to further disambiguate it
// from commands without modes.
func (tr2 *trace2Dataset) setQualifiedExeVerbModeName() {
	tr2.process.qualifiedNames.exeVerbMode = tr2.process.qualifiedNames.exeVerb

	if len(tr2.process.cmdMode) == 0 {
		return
	}

	tr2.process.qualifiedNames.exeVerbMode += "#" + tr2.process.cmdMode
}

func (tr2 *trace2Dataset) exportTraces() {
	if !tr2.sawData {
		return
	}

	if !tr2.prepareDataset() {
		return
	}

	dl, dl_debug := computeDetailLevel(
		tr2.rcvr_base.RcvrConfig.filterSettings,
		tr2.process.paramSetValues,
		tr2.process.qualifiedNames)

	tr2.rcvr_base.Logger.Debug(dl_debug)

	if dl == DetailLevelDrop {
		return
	}

	traces := tr2.ToTraces(dl, tr2.rcvr_base.RcvrConfig.filterSettings.Keynames)

	err := tr2.rcvr_base.TracesConsumer.ConsumeTraces(tr2.rcvr_base.ctx, traces)
	if err != nil {
		tr2.rcvr_base.Logger.Error(err.Error())
	}
}
