package trace2receiver

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Test important_events rule matching with basic rules
func Test_ImportantEvents_Match_Basic(t *testing.T) {
	fs := &FilterSettings{
		ImportantEvents: []ImportantEventRule{
			{Category: "gvfs-helper", KeyPrefix: "error/", FieldName: "gvfs_helper_errors"},
		},
	}

	tr2 := &trace2Dataset{
		process: TrProcess{
			importantEvents: make(map[string][]interface{}),
		},
	}
	tr2.rcvr_base = &Rcvr_Base{
		RcvrConfig: &Config{
			filterSettings: fs,
		},
	}

	apply__important_events(tr2, "gvfs-helper", "error/curl", "(curl:35) SSL connect error [hard_fail]")
	apply__important_events(tr2, "gvfs-helper", "error/http", "(http:503) Service Unavailable")

	values, ok := tr2.process.importantEvents["gvfs_helper_errors"]
	assert.True(t, ok)
	assert.Equal(t, 2, len(values))
	assert.Equal(t, "(curl:35) SSL connect error [hard_fail]", values[0])
	assert.Equal(t, "(http:503) Service Unavailable", values[1])
}

// Test important_events rule matching with no match
func Test_ImportantEvents_Match_NoMatch(t *testing.T) {
	fs := &FilterSettings{
		ImportantEvents: []ImportantEventRule{
			{Category: "gvfs-helper", KeyPrefix: "error/", FieldName: "gvfs_helper_errors"},
		},
	}

	tr2 := &trace2Dataset{
		process: TrProcess{
			importantEvents: make(map[string][]interface{}),
		},
	}
	tr2.rcvr_base = &Rcvr_Base{
		RcvrConfig: &Config{
			filterSettings: fs,
		},
	}

	// Wrong category
	apply__important_events(tr2, "other-helper", "error/curl", "some error")
	// Right category, wrong key prefix
	apply__important_events(tr2, "gvfs-helper", "status/ok", "all good")

	assert.Equal(t, 0, len(tr2.process.importantEvents))
}

// Test important_events rule matching when no rules are configured
func Test_ImportantEvents_Match_NoConfig(t *testing.T) {
	tr2 := &trace2Dataset{
		process: TrProcess{
			importantEvents: nil,
		},
	}
	tr2.rcvr_base = &Rcvr_Base{
		RcvrConfig: &Config{
			filterSettings: nil,
		},
	}

	// Should not crash
	apply__important_events(tr2, "gvfs-helper", "error/curl", "some error")
}

// Test important_events values appear in their own attribute at dl:summary
func Test_ImportantEvents_EmittedAtSummaryLevel(t *testing.T) {
	cfg := &Config{
		filterSettings: &FilterSettings{
			ImportantEvents: []ImportantEventRule{
				{Category: "gvfs-helper", KeyPrefix: "error/", FieldName: "gvfs_helper_errors"},
			},
		},
	}

	rcvr := &Rcvr_Base{
		RcvrConfig: cfg,
	}

	tr2 := NewTrace2Dataset(rcvr)

	if tr2.process.importantEvents == nil {
		t.Fatal("importantEvents should be initialized")
	}

	tr2.process.importantEvents["gvfs_helper_errors"] = append(
		tr2.process.importantEvents["gvfs_helper_errors"],
		"(curl:35) SSL connect error [hard_fail]")

	tr2.process.paramSetValues = make(map[string]string)
	tr2.process.mainThread.lifetime.startTime = mustParseTime(t, "2024-01-01T10:00:00Z")
	tr2.process.mainThread.lifetime.endTime = mustParseTime(t, "2024-01-01T10:00:10Z")

	traces := tr2.ToTraces(DetailLevelSummary, FilterKeynames{})

	assert.Equal(t, 1, traces.ResourceSpans().Len())
	rs := traces.ResourceSpans().At(0)
	ss := rs.ScopeSpans().At(0)
	assert.Greater(t, ss.Spans().Len(), 0)

	processSpan := ss.Spans().At(0)
	attrs := processSpan.Attributes()

	val, found := attrs.Get("trace2.process.important_events")
	assert.True(t, found, "important_events should be present at dl:summary level")

	jsonStr := val.Str()
	assert.Contains(t, jsonStr, "gvfs_helper_errors")
	assert.Contains(t, jsonStr, "SSL connect error")
}

// Test important_events with an integer value (data events can carry int64)
func Test_ImportantEvents_Match_IntValue(t *testing.T) {
	fs := &FilterSettings{
		ImportantEvents: []ImportantEventRule{
			{Category: "perf", KeyPrefix: "count/", FieldName: "perf_counts"},
		},
	}

	tr2 := &trace2Dataset{
		process: TrProcess{
			importantEvents: make(map[string][]interface{}),
		},
	}
	tr2.rcvr_base = &Rcvr_Base{
		RcvrConfig: &Config{
			filterSettings: fs,
		},
	}

	apply__important_events(tr2, "perf", "count/objects", int64(42))

	values := tr2.process.importantEvents["perf_counts"]
	assert.Equal(t, 1, len(values))
	assert.Equal(t, int64(42), values[0])
}

// Test end-to-end: parse a raw data event JSON and verify the value
// is captured into important_events, even when the region stack is empty
// (nesting > 1 with no matching region).
func Test_ImportantEvents_EndToEnd_NestedEvent(t *testing.T) {
	cfg := &Config{
		filterSettings: &FilterSettings{
			ImportantEvents: []ImportantEventRule{
				{Category: "gvfs-helper", KeyPrefix: "error/", FieldName: "gvfs_helper_errors"},
			},
		},
	}
	rcvr := &Rcvr_Base{
		RcvrConfig: cfg,
	}
	tr2 := NewTrace2Dataset(rcvr)

	// Feed the exact JSON from trace.json through parse+apply.
	// This event has nesting=2 but no region stack, which previously
	// would have caused the value to be silently dropped.
	raw := `{"event":"data","sid":"20260420T161123.706538Z-H27d9ce02-P0000f630","thread":"main","time":"2026-04-20T16:11:29.201996Z","file":"gvfs-helper.c","line":1079,"t_abs":5.306816,"t_rel":0.148534,"nesting":2,"category":"gvfs-helper","key":"error/curl","value":"(curl:35) SSL connect error [hard_fail]"}`
	evt, err := parse_json([]byte(raw))
	assert.NoError(t, err)
	assert.NotNil(t, evt)

	err = evt_apply(tr2, evt)
	assert.NoError(t, err)

	assert.NotNil(t, tr2.process.importantEvents)
	values, ok := tr2.process.importantEvents["gvfs_helper_errors"]
	assert.True(t, ok, "gvfs_helper_errors should be in importantEvents")
	if ok {
		assert.Equal(t, 1, len(values))
		assert.Equal(t, "(curl:35) SSL connect error [hard_fail]", values[0])
	}
}

// ================================================================
// Full pipeline E2E tests: raw trace2 JSON events -> parse -> apply
// -> prepareDataset -> ToTraces -> extract OTLP span attributes ->
// verify important_events JSON content.
// ================================================================

// load_test_dataset_with_config is like load_test_dataset in
// evt_apply_test.go but accepts a Config so that important_events rules
// are active during event processing.
func load_test_dataset_with_config(t *testing.T, cfg *Config, events []string) (tr2 *trace2Dataset, sufficient bool) {
	t.Helper()

	rcvr := &Rcvr_Base{
		RcvrConfig: cfg,
	}
	tr2 = NewTrace2Dataset(rcvr)

	for _, s := range events {
		evt, err := parse_json([]byte(s))
		if err != nil {
			t.Fatalf("parse of '%s' failed: %s", s, err.Error())
		}
		if evt == nil {
			continue
		}
		err = evt_apply(tr2, evt)
		if err != nil {
			if _, ok := err.(*RejectClientError); ok {
				t.Fatalf("rejected: %s", err.Error())
			}
			t.Fatalf("apply of '%s' failed: %s", s, err.Error())
		}
	}

	sufficient = tr2.prepareDataset()
	return tr2, sufficient
}

// extractSummaryJSON runs the full OTLP conversion at a given detail
// level and returns the parsed summary JSON from the process span.
// Returns nil if the summary attribute is absent.
func extractSummaryJSON(t *testing.T, tr2 *trace2Dataset, dl FilterDetailLevel) map[string]interface{} {
	t.Helper()

	traces := tr2.ToTraces(dl, FilterKeynames{})
	if traces.ResourceSpans().Len() == 0 {
		t.Fatal("no resource spans")
	}
	ss := traces.ResourceSpans().At(0).ScopeSpans().At(0)
	if ss.Spans().Len() == 0 {
		t.Fatal("no spans")
	}

	processSpan := ss.Spans().At(0)
	attrs := processSpan.Attributes()

	summaryVal, found := attrs.Get(string(Trace2ProcessSummary))
	if !found {
		return nil
	}

	var result map[string]interface{}
	err := json.Unmarshal([]byte(summaryVal.Str()), &result)
	if err != nil {
		t.Fatalf("failed to parse summary JSON: %s", err.Error())
	}
	return result
}

// extractImportantEventsJSON runs the full OTLP conversion at a given
// detail level and returns the parsed important_events JSON from the
// process span. Returns nil if the attribute is absent.
func extractImportantEventsJSON(t *testing.T, tr2 *trace2Dataset, dl FilterDetailLevel) map[string]interface{} {
	t.Helper()

	traces := tr2.ToTraces(dl, FilterKeynames{})
	if traces.ResourceSpans().Len() == 0 {
		t.Fatal("no resource spans")
	}
	ss := traces.ResourceSpans().At(0).ScopeSpans().At(0)
	if ss.Spans().Len() == 0 {
		t.Fatal("no spans")
	}

	processSpan := ss.Spans().At(0)
	attrs := processSpan.Attributes()

	val, found := attrs.Get(string(Trace2ProcessImportantEvents))
	if !found {
		return nil
	}

	var result map[string]interface{}
	err := json.Unmarshal([]byte(val.Str()), &result)
	if err != nil {
		t.Fatalf("failed to parse important_events JSON: %s", err.Error())
	}
	return result
}

// Test: full pipeline with a process-level data event (nesting=1)
// captured and visible in the OTLP span at dl:summary.
func Test_E2E_ImportantEvents_ProcessLevel_AtSummaryDetailLevel(t *testing.T) {
	cfg := &Config{
		filterSettings: &FilterSettings{
			ImportantEvents: []ImportantEventRule{
				{Category: "gvfs-helper", KeyPrefix: "error/", FieldName: "gvfs_helper_errors"},
			},
		},
	}

	events := []string{
		x_make_version(),
		x_make_start(),
		x_make_cmd_name(),
		x_make_data_string(x_main, 1, "gvfs-helper", "error/curl", "(curl:35) SSL connect error"),
		x_make_atexit(),
	}

	tr2, sufficient := load_test_dataset_with_config(t, cfg, events)
	assert.True(t, sufficient)

	ie := extractImportantEventsJSON(t, tr2, DetailLevelSummary)
	assert.NotNil(t, ie, "important_events should be present at dl:summary")

	raw, ok := ie["gvfs_helper_errors"]
	assert.True(t, ok, "gvfs_helper_errors should be in important_events")
	arr := raw.([]interface{})
	assert.Equal(t, 1, len(arr))
	assert.Equal(t, "(curl:35) SSL connect error", arr[0])
}

// Test: data event inside a region (nesting=2) is captured even when
// the region stack is properly set up. The value should appear in the
// important_events AND the region should have it in its own data.
func Test_E2E_ImportantEvents_InsideRegion(t *testing.T) {
	cfg := &Config{
		filterSettings: &FilterSettings{
			ImportantEvents: []ImportantEventRule{
				{Category: "gvfs-helper", KeyPrefix: "error/", FieldName: "gvfs_helper_errors"},
			},
		},
	}

	events := []string{
		x_make_version(),
		x_make_start(),
		x_make_cmd_name(),
		x_make_region_enter(x_main, 1, "gvfs-helper", "fetch", "fetching"),
		x_make_data_string(x_main, 2, "gvfs-helper", "error/curl", "(curl:35) SSL connect error"),
		x_make_region_leave(x_main, 1, "gvfs-helper", "fetch", "fetching"),
		x_make_atexit(),
	}

	tr2, sufficient := load_test_dataset_with_config(t, cfg, events)
	assert.True(t, sufficient)

	// important_events should have the captured value
	ie := extractImportantEventsJSON(t, tr2, DetailLevelSummary)
	assert.NotNil(t, ie)
	raw, ok := ie["gvfs_helper_errors"]
	assert.True(t, ok, "captured value should be in important_events")
	arr := raw.([]interface{})
	assert.Equal(t, "(curl:35) SSL connect error", arr[0])

	// The region should also have it in its own data
	assert.Equal(t, 1, len(tr2.completedRegions))
	r := tr2.completedRegions[0]
	assert.NotNil(t, r.dataValues)
	assert.Equal(t, "(curl:35) SSL connect error", r.dataValues["gvfs-helper"]["error/curl"])
}

// Test: data event at nesting=2 with NO region on the stack (orphaned).
// Value should still be captured; region attachment fails silently.
func Test_E2E_ImportantEvents_OrphanedNesting(t *testing.T) {
	cfg := &Config{
		filterSettings: &FilterSettings{
			ImportantEvents: []ImportantEventRule{
				{Category: "gvfs-helper", KeyPrefix: "error/", FieldName: "gvfs_helper_errors"},
			},
		},
	}

	events := []string{
		x_make_version(),
		x_make_start(),
		x_make_cmd_name(),
		// No region_enter, so the region stack is empty
		x_make_data_string(x_main, 2, "gvfs-helper", "error/curl", "(curl:35) SSL connect error"),
		x_make_atexit(),
	}

	tr2, sufficient := load_test_dataset_with_config(t, cfg, events)
	assert.True(t, sufficient)

	ie := extractImportantEventsJSON(t, tr2, DetailLevelSummary)
	assert.NotNil(t, ie)
	raw, ok := ie["gvfs_helper_errors"]
	assert.True(t, ok, "captured value should be in important_events even without region")
	arr := raw.([]interface{})
	assert.Equal(t, 1, len(arr))
	assert.Equal(t, "(curl:35) SSL connect error", arr[0])
}

// Test: multiple data events matching the same rule accumulate all values.
func Test_E2E_ImportantEvents_MultipleValues(t *testing.T) {
	cfg := &Config{
		filterSettings: &FilterSettings{
			ImportantEvents: []ImportantEventRule{
				{Category: "gvfs-helper", KeyPrefix: "error/", FieldName: "gvfs_helper_errors"},
			},
		},
	}

	events := []string{
		x_make_version(),
		x_make_start(),
		x_make_cmd_name(),
		x_make_data_string(x_main, 1, "gvfs-helper", "error/curl", "first error"),
		x_make_data_string(x_main, 1, "gvfs-helper", "error/http", "second error"),
		x_make_data_string(x_main, 1, "gvfs-helper", "error/tls", "third error"),
		x_make_atexit(),
	}

	tr2, sufficient := load_test_dataset_with_config(t, cfg, events)
	assert.True(t, sufficient)

	ie := extractImportantEventsJSON(t, tr2, DetailLevelSummary)
	assert.NotNil(t, ie)
	arr := ie["gvfs_helper_errors"].([]interface{})
	assert.Equal(t, 3, len(arr))
	assert.Equal(t, "first error", arr[0])
	assert.Equal(t, "second error", arr[1])
	assert.Equal(t, "third error", arr[2])
}

// Test: data events that do NOT match the pattern should NOT appear
// in the important_events.
func Test_E2E_ImportantEvents_NonMatchingEventsExcluded(t *testing.T) {
	cfg := &Config{
		filterSettings: &FilterSettings{
			ImportantEvents: []ImportantEventRule{
				{Category: "gvfs-helper", KeyPrefix: "error/", FieldName: "gvfs_helper_errors"},
			},
		},
	}

	events := []string{
		x_make_version(),
		x_make_start(),
		x_make_cmd_name(),
		// Wrong category
		x_make_data_string(x_main, 1, "other-helper", "error/curl", "wrong category"),
		// Right category, wrong key prefix
		x_make_data_string(x_main, 1, "gvfs-helper", "status/ok", "wrong prefix"),
		// Right category, key doesn't start with prefix
		x_make_data_string(x_main, 1, "gvfs-helper", "warn/timeout", "wrong prefix too"),
		x_make_atexit(),
	}

	tr2, sufficient := load_test_dataset_with_config(t, cfg, events)
	assert.True(t, sufficient)

	ie := extractImportantEventsJSON(t, tr2, DetailLevelSummary)
	assert.Nil(t, ie, "non-matching events should not produce important_events attribute")
}

// Test: integer values (data events can carry int64).
func Test_E2E_ImportantEvents_IntegerValue(t *testing.T) {
	cfg := &Config{
		filterSettings: &FilterSettings{
			ImportantEvents: []ImportantEventRule{
				{Category: "perf", KeyPrefix: "count/", FieldName: "perf_counts"},
			},
		},
	}

	events := []string{
		x_make_version(),
		x_make_start(),
		x_make_cmd_name(),
		x_make_data_intmax(x_main, 1, "perf", "count/objects", 42),
		x_make_atexit(),
	}

	tr2, sufficient := load_test_dataset_with_config(t, cfg, events)
	assert.True(t, sufficient)

	ie := extractImportantEventsJSON(t, tr2, DetailLevelSummary)
	assert.NotNil(t, ie)
	arr := ie["perf_counts"].([]interface{})
	assert.Equal(t, 1, len(arr))
	// JSON numbers unmarshal as float64
	assert.Equal(t, float64(42), arr[0])
}

// Test: important_events coexists with message_patterns and region_timers
// in the same output without interference.
func Test_E2E_ImportantEvents_CoexistsWithOtherRuleTypes(t *testing.T) {
	cfg := &Config{
		summary: &SummarySettings{
			MessagePatterns: []MessagePatternRule{
				{Prefix: "error:", FieldName: "error_msg_count"},
			},
			RegionTimers: []RegionTimerRule{
				{Category: "gvfs-helper", Label: "fetch", CountField: "fetch_count"},
			},
		},
		filterSettings: &FilterSettings{
			ImportantEvents: []ImportantEventRule{
				{Category: "gvfs-helper", KeyPrefix: "error/", FieldName: "gvfs_helper_errors"},
			},
		},
	}

	events := []string{
		x_make_version(),
		x_make_start(),
		x_make_cmd_name(),
		// Trigger message pattern
		x_make_error("error: something broke", "error: %s"),
		// Trigger region timer
		x_make_region_enter(x_main, 1, "gvfs-helper", "fetch", "fetching"),
		// Trigger important_events inside region
		x_make_data_string(x_main, 2, "gvfs-helper", "error/curl", "(curl:35) fail"),
		x_make_region_leave(x_main, 1, "gvfs-helper", "fetch", "fetching"),
		x_make_atexit(),
	}

	tr2, sufficient := load_test_dataset_with_config(t, cfg, events)
	assert.True(t, sufficient)

	summary := extractSummaryJSON(t, tr2, DetailLevelSummary)
	assert.NotNil(t, summary)

	// Message pattern count
	assert.Equal(t, float64(1), summary["error_msg_count"])
	// Region timer count
	assert.Equal(t, float64(1), summary["fetch_count"])

	// important_events value appears in its separate attribute
	ie := extractImportantEventsJSON(t, tr2, DetailLevelSummary)
	assert.NotNil(t, ie)
	arr := ie["gvfs_helper_errors"].([]interface{})
	assert.Equal(t, 1, len(arr))
	assert.Equal(t, "(curl:35) fail", arr[0])
}

// Test: captured values appear at ALL detail levels, not just verbose.
func Test_E2E_ImportantEvents_AllDetailLevels(t *testing.T) {
	cfg := &Config{
		filterSettings: &FilterSettings{
			ImportantEvents: []ImportantEventRule{
				{Category: "gvfs-helper", KeyPrefix: "error/", FieldName: "gvfs_helper_errors"},
			},
		},
	}

	events := []string{
		x_make_version(),
		x_make_start(),
		x_make_cmd_name(),
		x_make_data_string(x_main, 1, "gvfs-helper", "error/curl", "(curl:35) fail"),
		x_make_atexit(),
	}

	for _, dl := range []FilterDetailLevel{DetailLevelSummary, DetailLevelProcess, DetailLevelVerbose} {
		tr2, sufficient := load_test_dataset_with_config(t, cfg, events)
		assert.True(t, sufficient)

		ie := extractImportantEventsJSON(t, tr2, dl)
		assert.NotNil(t, ie, "important_events should be present at detail level %d", dl)
		arr := ie["gvfs_helper_errors"].([]interface{})
		assert.Equal(t, 1, len(arr))
		assert.Equal(t, "(curl:35) fail", arr[0])
	}
}

// Test: when no important_events are configured, data events do NOT
// create the attribute (no spurious empty arrays).
func Test_E2E_ImportantEvents_NoPatternsConfigured(t *testing.T) {
	cfg := &Config{
		summary: &SummarySettings{
			MessagePatterns: []MessagePatternRule{
				{Prefix: "error:", FieldName: "error_count"},
			},
		},
		filterSettings: &FilterSettings{},
	}

	events := []string{
		x_make_version(),
		x_make_start(),
		x_make_cmd_name(),
		x_make_data_string(x_main, 1, "gvfs-helper", "error/curl", "some error"),
		x_make_atexit(),
	}

	tr2, sufficient := load_test_dataset_with_config(t, cfg, events)
	assert.True(t, sufficient)

	ie := extractImportantEventsJSON(t, tr2, DetailLevelSummary)
	assert.Nil(t, ie, "should not have important_events when no rules configured")
}

// Test: with no summary config at all (summary is nil), data events
// are processed without crashing.
func Test_E2E_ImportantEvents_NoSummaryConfig(t *testing.T) {
	cfg := &Config{
		summary:        nil,
		filterSettings: &FilterSettings{},
	}

	events := []string{
		x_make_version(),
		x_make_start(),
		x_make_cmd_name(),
		x_make_data_string(x_main, 1, "gvfs-helper", "error/curl", "some error"),
		x_make_data_string(x_main, 2, "gvfs-helper", "error/http", "nested error"),
		x_make_atexit(),
	}

	tr2, sufficient := load_test_dataset_with_config(t, cfg, events)
	assert.True(t, sufficient)

	summary := extractSummaryJSON(t, tr2, DetailLevelSummary)
	assert.Nil(t, summary, "no summary when not configured")
	ie := extractImportantEventsJSON(t, tr2, DetailLevelSummary)
	assert.Nil(t, ie, "no important_events when no rules configured")
}

// Test: data events on a non-main thread are still captured.
func Test_E2E_ImportantEvents_NonMainThread(t *testing.T) {
	cfg := &Config{
		filterSettings: &FilterSettings{
			ImportantEvents: []ImportantEventRule{
				{Category: "gvfs-helper", KeyPrefix: "error/", FieldName: "gvfs_helper_errors"},
			},
		},
	}

	events := []string{
		x_make_version(),
		x_make_start(),
		x_make_cmd_name(),
		x_make_thread_start("worker01"),
		x_make_region_enter("worker01", 1, "gvfs-helper", "fetch", "fetching"),
		x_make_data_string("worker01", 2, "gvfs-helper", "error/curl", "thread error"),
		x_make_region_leave("worker01", 1, "gvfs-helper", "fetch", "fetching"),
		x_make_thread_exit("worker01"),
		x_make_atexit(),
	}

	tr2, sufficient := load_test_dataset_with_config(t, cfg, events)
	assert.True(t, sufficient)

	ie := extractImportantEventsJSON(t, tr2, DetailLevelSummary)
	assert.NotNil(t, ie)
	arr := ie["gvfs_helper_errors"].([]interface{})
	assert.Equal(t, 1, len(arr))
	assert.Equal(t, "thread error", arr[0])
}

// Test: deeply nested data event (nesting=5) with only partial region
// stack. The value should still be captured even though region attachment fails.
func Test_E2E_ImportantEvents_DeepNesting_PartialRegionStack(t *testing.T) {
	cfg := &Config{
		filterSettings: &FilterSettings{
			ImportantEvents: []ImportantEventRule{
				{Category: "gvfs-helper", KeyPrefix: "error/", FieldName: "gvfs_helper_errors"},
			},
		},
	}

	events := []string{
		x_make_version(),
		x_make_start(),
		x_make_cmd_name(),
		// Only push one region, but data claims nesting=5
		x_make_region_enter(x_main, 1, "gvfs-helper", "fetch", "fetching"),
		x_make_data_string(x_main, 5, "gvfs-helper", "error/curl", "deep nested error"),
		x_make_region_leave(x_main, 1, "gvfs-helper", "fetch", "fetching"),
		x_make_atexit(),
	}

	tr2, sufficient := load_test_dataset_with_config(t, cfg, events)
	assert.True(t, sufficient)

	ie := extractImportantEventsJSON(t, tr2, DetailLevelSummary)
	assert.NotNil(t, ie)
	arr := ie["gvfs_helper_errors"].([]interface{})
	assert.Equal(t, 1, len(arr))
	assert.Equal(t, "deep nested error", arr[0])
}

// Test: multiple rules matching different categories in the same
// event stream produce independent fields.
func Test_E2E_ImportantEvents_MultipleRules(t *testing.T) {
	cfg := &Config{
		filterSettings: &FilterSettings{
			ImportantEvents: []ImportantEventRule{
				{Category: "gvfs-helper", KeyPrefix: "error/", FieldName: "gvfs_errors"},
				{Category: "network", KeyPrefix: "timeout/", FieldName: "network_timeouts"},
			},
		},
	}

	events := []string{
		x_make_version(),
		x_make_start(),
		x_make_cmd_name(),
		x_make_data_string(x_main, 1, "gvfs-helper", "error/curl", "curl error"),
		x_make_data_string(x_main, 1, "network", "timeout/dns", "DNS timeout"),
		x_make_data_string(x_main, 1, "network", "timeout/connect", "connect timeout"),
		// This matches neither rule
		x_make_data_string(x_main, 1, "unrelated", "error/foo", "should not appear"),
		x_make_atexit(),
	}

	tr2, sufficient := load_test_dataset_with_config(t, cfg, events)
	assert.True(t, sufficient)

	ie := extractImportantEventsJSON(t, tr2, DetailLevelSummary)
	assert.NotNil(t, ie)

	gvfs := ie["gvfs_errors"].([]interface{})
	assert.Equal(t, 1, len(gvfs))
	assert.Equal(t, "curl error", gvfs[0])

	net := ie["network_timeouts"].([]interface{})
	assert.Equal(t, 2, len(net))
	assert.Equal(t, "DNS timeout", net[0])
	assert.Equal(t, "connect timeout", net[1])

	_, ok := ie["unrelated"]
	assert.False(t, ok, "unrelated category should not appear")
}
