package trace2receiver

import (
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
