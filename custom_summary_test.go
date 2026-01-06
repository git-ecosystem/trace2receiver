package trace2receiver

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Test that valid settings parse correctly
func Test_ValidCustomSummarySettings(t *testing.T) {
	yml := `
message_patterns:
  - prefix: "error:"
    field_name: "error_count"
  - prefix: "warning:"
    field_name: "warning_count"

region_timers:
  - category: "index"
    label: "do_read_index"
    count_field: "index_read_count"
    time_field: "index_read_time"
  - category: "pack"
    label: "prepare"
    count_field: "pack_prepare_count"
`
	css, err := parseCustomSummarySettingsFromBuffer([]byte(yml), "test.yml")
	assert.NoError(t, err)
	assert.NotNil(t, css)
	assert.Equal(t, 2, len(css.MessagePatterns))
	assert.Equal(t, 2, len(css.RegionTimers))
}

// Test that empty prefix is rejected
func Test_EmptyPrefix_Rejected(t *testing.T) {
	yml := `
message_patterns:
  - prefix: ""
    field_name: "error_count"
`
	_, err := parseCustomSummarySettingsFromBuffer([]byte(yml), "test.yml")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "prefix cannot be empty")
}

// Test that empty field name is rejected
func Test_EmptyFieldName_Rejected(t *testing.T) {
	yml := `
message_patterns:
  - prefix: "error:"
    field_name: ""
`
	_, err := parseCustomSummarySettingsFromBuffer([]byte(yml), "test.yml")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "field_name cannot be empty")
}

// Test that duplicate field names are rejected
func Test_DuplicateFieldNames_Rejected(t *testing.T) {
	yml := `
message_patterns:
  - prefix: "error:"
    field_name: "count"
  - prefix: "warning:"
    field_name: "count"
`
	_, err := parseCustomSummarySettingsFromBuffer([]byte(yml), "test.yml")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "duplicate field_name")
}

// Test that empty category is rejected
func Test_EmptyCategory_Rejected(t *testing.T) {
	yml := `
region_timers:
  - category: ""
    label: "test"
    count_field: "count"
`
	_, err := parseCustomSummarySettingsFromBuffer([]byte(yml), "test.yml")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "category cannot be empty")
}

// Test that region timer with neither count nor time field is rejected
func Test_NoCountOrTimeField_Rejected(t *testing.T) {
	yml := `
region_timers:
  - category: "index"
    label: "test"
`
	_, err := parseCustomSummarySettingsFromBuffer([]byte(yml), "test.yml")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "at least one of count_field or time_field must be specified")
}

// Test that duplicate field names across types are rejected
func Test_DuplicateFieldNames_CrossType_Rejected(t *testing.T) {
	yml := `
message_patterns:
  - prefix: "error:"
    field_name: "count"

region_timers:
  - category: "index"
    label: "test"
    count_field: "count"
`
	_, err := parseCustomSummarySettingsFromBuffer([]byte(yml), "test.yml")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "duplicate field_name")
}

// Test toMap with mixed zero and non-zero values
func Test_ToMap_MixedValues(t *testing.T) {
	csa := newCustomSummaryAccumulator()
	csa.incrementMessageCount("msg1")
	csa.incrementMessageCount("msg1")
	csa.incrementMessageCount("msg1")
	csa.incrementMessageCount("msg1")
	csa.incrementMessageCount("msg1")
	csa.messageCounts["msg2"] = 0
	csa.addRegionMetrics("region1", "", 0)
	csa.addRegionMetrics("region1", "", 0)
	csa.addRegionMetrics("region1", "", 0)
	csa.regionCounts["region2"] = 0
	csa.addRegionMetrics("", "time1", 1.5)
	csa.regionTimes["time2"] = 0.0

	result := csa.toMap()

	// Should only include non-zero values
	assert.Equal(t, 3, len(result))
	assert.Equal(t, int64(5), result["msg1"])
	assert.Equal(t, int64(3), result["region1"])
	assert.Equal(t, 1.5, result["time1"])
	_, hasMsg2 := result["msg2"]
	assert.False(t, hasMsg2)
}

// Test toMap with empty accumulator
func Test_ToMap_Empty(t *testing.T) {
	csa := newCustomSummaryAccumulator()
	result := csa.toMap()
	assert.Equal(t, 0, len(result))
}

// Test configuredCustomSummary with empty settings
func Test_ConfiguredCustomSummary_EmptySettings(t *testing.T) {
	settings := &CustomSummarySettings{}
	csa := configuredCustomSummary(settings)

	assert.NotNil(t, csa)
	assert.Equal(t, 0, len(csa.messageCounts))
	assert.Equal(t, 0, len(csa.regionCounts))
	assert.Equal(t, 0, len(csa.regionTimes))
}

// Test configuredCustomSummary with only message patterns
func Test_ConfiguredCustomSummary_MessagePatternsOnly(t *testing.T) {
	settings := &CustomSummarySettings{
		MessagePatterns: []MessagePatternRule{
			{Prefix: "error:", FieldName: "error_count"},
			{Prefix: "warning:", FieldName: "warning_count"},
		},
	}
	csa := configuredCustomSummary(settings)

	assert.NotNil(t, csa)
	assert.Equal(t, 2, len(csa.messageCounts))
	assert.Equal(t, int64(0), csa.messageCounts["error_count"])
	assert.Equal(t, int64(0), csa.messageCounts["warning_count"])
	assert.Equal(t, 0, len(csa.regionCounts))
	assert.Equal(t, 0, len(csa.regionTimes))
}

// Test configuredCustomSummary with region timers having only count field
func Test_ConfiguredCustomSummary_RegionTimers_CountOnly(t *testing.T) {
	settings := &CustomSummarySettings{
		RegionTimers: []RegionTimerRule{
			{Category: "index", Label: "read", CountField: "index_read_count"},
			{Category: "pack", Label: "prepare", CountField: "pack_prepare_count"},
		},
	}
	csa := configuredCustomSummary(settings)

	assert.NotNil(t, csa)
	assert.Equal(t, 0, len(csa.messageCounts))
	assert.Equal(t, 2, len(csa.regionCounts))
	assert.Equal(t, int64(0), csa.regionCounts["index_read_count"])
	assert.Equal(t, int64(0), csa.regionCounts["pack_prepare_count"])
	assert.Equal(t, 0, len(csa.regionTimes))
}

// Test configuredCustomSummary with region timers having only time field
func Test_ConfiguredCustomSummary_RegionTimers_TimeOnly(t *testing.T) {
	settings := &CustomSummarySettings{
		RegionTimers: []RegionTimerRule{
			{Category: "index", Label: "read", TimeField: "index_read_time"},
			{Category: "pack", Label: "prepare", TimeField: "pack_prepare_time"},
		},
	}
	csa := configuredCustomSummary(settings)

	assert.NotNil(t, csa)
	assert.Equal(t, 0, len(csa.messageCounts))
	assert.Equal(t, 0, len(csa.regionCounts))
	assert.Equal(t, 2, len(csa.regionTimes))
	assert.Equal(t, 0.0, csa.regionTimes["index_read_time"])
	assert.Equal(t, 0.0, csa.regionTimes["pack_prepare_time"])
}

// Test configuredCustomSummary with region timers having both count and time fields
func Test_ConfiguredCustomSummary_RegionTimers_BothFields(t *testing.T) {
	settings := &CustomSummarySettings{
		RegionTimers: []RegionTimerRule{
			{
				Category:   "index",
				Label:      "read",
				CountField: "index_read_count",
				TimeField:  "index_read_time",
			},
		},
	}
	csa := configuredCustomSummary(settings)

	assert.NotNil(t, csa)
	assert.Equal(t, 0, len(csa.messageCounts))
	assert.Equal(t, 1, len(csa.regionCounts))
	assert.Equal(t, int64(0), csa.regionCounts["index_read_count"])
	assert.Equal(t, 1, len(csa.regionTimes))
	assert.Equal(t, 0.0, csa.regionTimes["index_read_time"])
}

// Test configuredCustomSummary with empty count/time field strings
func Test_ConfiguredCustomSummary_RegionTimers_EmptyFields(t *testing.T) {
	settings := &CustomSummarySettings{
		RegionTimers: []RegionTimerRule{
			{
				Category:   "index",
				Label:      "read",
				CountField: "",
				TimeField:  "index_read_time",
			},
			{
				Category:   "pack",
				Label:      "prepare",
				CountField: "pack_prepare_count",
				TimeField:  "",
			},
		},
	}
	csa := configuredCustomSummary(settings)

	assert.NotNil(t, csa)
	assert.Equal(t, 0, len(csa.messageCounts))
	assert.Equal(t, 1, len(csa.regionCounts))
	assert.Equal(t, int64(0), csa.regionCounts["pack_prepare_count"])
	assert.Equal(t, 1, len(csa.regionTimes))
	assert.Equal(t, 0.0, csa.regionTimes["index_read_time"])
}

// Test configuredCustomSummary with mixed message patterns and region timers
func Test_ConfiguredCustomSummary_Mixed(t *testing.T) {
	settings := &CustomSummarySettings{
		MessagePatterns: []MessagePatternRule{
			{Prefix: "error:", FieldName: "error_count"},
			{Prefix: "warning:", FieldName: "warning_count"},
		},
		RegionTimers: []RegionTimerRule{
			{
				Category:   "index",
				Label:      "read",
				CountField: "index_read_count",
				TimeField:  "index_read_time",
			},
			{
				Category:  "pack",
				Label:     "prepare",
				TimeField: "pack_prepare_time",
			},
		},
	}
	csa := configuredCustomSummary(settings)

	assert.NotNil(t, csa)
	// Check message counts
	assert.Equal(t, 2, len(csa.messageCounts))
	assert.Equal(t, int64(0), csa.messageCounts["error_count"])
	assert.Equal(t, int64(0), csa.messageCounts["warning_count"])
	// Check region counts
	assert.Equal(t, 1, len(csa.regionCounts))
	assert.Equal(t, int64(0), csa.regionCounts["index_read_count"])
	// Check region times
	assert.Equal(t, 2, len(csa.regionTimes))
	assert.Equal(t, 0.0, csa.regionTimes["index_read_time"])
	assert.Equal(t, 0.0, csa.regionTimes["pack_prepare_time"])
}

// Test JSON marshaling format
func Test_JSONMarshal_Format(t *testing.T) {
	csa := newCustomSummaryAccumulator()
	csa.incrementMessageCount("queuedCount")
	csa.incrementMessageCount("queuedCount")
	csa.addRegionMetrics("prefetchCount", "prefetchTime", 30.4)

	summaryMap := csa.toMap()
	jsonBytes, err := json.Marshal(summaryMap)
	assert.NoError(t, err)

	// Verify JSON structure
	var result map[string]interface{}
	err = json.Unmarshal(jsonBytes, &result)
	assert.NoError(t, err)
	assert.Equal(t, float64(2), result["queuedCount"])
	assert.Equal(t, float64(1), result["prefetchCount"])
	assert.InDelta(t, 30.4, result["prefetchTime"], 0.01)
}
