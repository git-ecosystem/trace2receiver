package trace2receiver

import (
	"fmt"
)

// CustomSummarySettings describes the configuration for custom summary
// metrics that should be aggregated from trace2 events and emitted as
// a single JSON object in the OTEL process span.
type CustomSummarySettings struct {
	MessagePatterns []MessagePatternRule `mapstructure:"message_patterns"`
	RegionTimers    []RegionTimerRule    `mapstructure:"region_timers"`
}

// MessagePatternRule defines a rule for counting messages that match
// a specific prefix. When a message (from error, data, or other events)
// starts with the specified prefix, a counter is incremented and emitted
// in the custom summary using the specified field name.
type MessagePatternRule struct {
	// Prefix is the string prefix to match at the beginning of messages
	Prefix string `mapstructure:"prefix"`

	// FieldName is the name of the field in the customSummary object
	// where the count will be stored
	FieldName string `mapstructure:"field_name"`
}

// RegionTimerRule defines a rule for aggregating time spent in regions
// that match a specific (category, label) pair. The count and/or total
// time can be emitted in the custom summary using the specified field names.
type RegionTimerRule struct {
	// Category is the region category to match (exact match)
	Category string `mapstructure:"category"`

	// Label is the region label to match (exact match)
	Label string `mapstructure:"label"`

	// CountField is the optional name of the field in the customSummary
	// object where the count of matching regions will be stored.
	// If empty, count will not be tracked.
	CountField string `mapstructure:"count_field"`

	// TimeField is the optional name of the field in the customSummary
	// object where the total time (in seconds) will be stored.
	// If empty, time will not be tracked.
	TimeField string `mapstructure:"time_field"`
}

// parseCustomSummarySettings parses a custom summary configuration
// from a YML file and validates the configuration.
func parseCustomSummarySettings(path string) (*CustomSummarySettings, error) {
	return parseYmlFile[CustomSummarySettings](path, parseCustomSummarySettingsFromBuffer)
}

// parseCustomSummarySettingsFromBuffer parses and validates custom
// summary settings from a YAML byte buffer.
func parseCustomSummarySettingsFromBuffer(data []byte, path string) (*CustomSummarySettings, error) {
	css, err := parseYmlBuffer[CustomSummarySettings](data, path)
	if err != nil {
		return nil, err
	}

	// Track all field names to detect duplicates
	fieldNames := make(map[string]bool)

	// Validate message pattern rules
	for i, rule := range css.MessagePatterns {
		if len(rule.Prefix) == 0 {
			return nil, fmt.Errorf("message_patterns[%d]: prefix cannot be empty", i)
		}
		if len(rule.FieldName) == 0 {
			return nil, fmt.Errorf("message_patterns[%d]: field_name cannot be empty", i)
		}
		if fieldNames[rule.FieldName] {
			return nil, fmt.Errorf("message_patterns[%d]: duplicate field_name '%s'", i, rule.FieldName)
		}
		fieldNames[rule.FieldName] = true
	}

	// Validate region timer rules
	for i, rule := range css.RegionTimers {
		if len(rule.Category) == 0 {
			return nil, fmt.Errorf("region_timers[%d]: category cannot be empty", i)
		}
		if len(rule.Label) == 0 {
			return nil, fmt.Errorf("region_timers[%d]: label cannot be empty", i)
		}
		if len(rule.CountField) == 0 && len(rule.TimeField) == 0 {
			return nil, fmt.Errorf("region_timers[%d]: at least one of count_field or time_field must be specified", i)
		}

		if len(rule.CountField) > 0 {
			if fieldNames[rule.CountField] {
				return nil, fmt.Errorf("region_timers[%d]: duplicate field_name '%s'", i, rule.CountField)
			}
			fieldNames[rule.CountField] = true
		}

		if len(rule.TimeField) > 0 {
			if fieldNames[rule.TimeField] {
				return nil, fmt.Errorf("region_timers[%d]: duplicate field_name '%s'", i, rule.TimeField)
			}
			fieldNames[rule.TimeField] = true
		}
	}

	return css, nil
}
