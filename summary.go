package trace2receiver

import "strings"

// SummaryAccumulator stores aggregated metric values
// during trace2 event processing. These values are accumulated as
// events arrive and then emitted as a single JSON object in the
// process span.
type SummaryAccumulator struct {
	// messageCounts maps field names to message count values
	messageCounts map[string]int64

	// regionCounts maps field names to region occurrence counts
	regionCounts map[string]int64

	// regionTimes maps field names to total time in seconds
	regionTimes map[string]float64
}

// newSummaryAccumulator creates a new accumulator with
// initialized maps.
func newSummaryAccumulator() *SummaryAccumulator {
	return &SummaryAccumulator{
		messageCounts: make(map[string]int64),
		regionCounts:  make(map[string]int64),
		regionTimes:   make(map[string]float64),
	}
}

// configuredSummary creates an accumulator initialized with
// field names from the settings, all set to zero values.
func configuredSummary(settings *SummarySettings) *SummaryAccumulator {
	summary := newSummaryAccumulator()

	// Initialize messageCounts with field names from MessagePatterns
	for _, rule := range settings.MessagePatterns {
		summary.messageCounts[rule.FieldName] = 0
	}

	// Initialize regionCounts and regionTimes with field names from RegionTimers
	for _, rule := range settings.RegionTimers {
		if len(rule.CountField) > 0 {
			summary.regionCounts[rule.CountField] = 0
		}
		if len(rule.TimeField) > 0 {
			summary.regionTimes[rule.TimeField] = 0.0
		}
	}

	return summary
}

// incrementMessageCount increments the count for a specific field name
// by 1. This is called when a message matches a configured prefix pattern.
func (csa *SummaryAccumulator) incrementMessageCount(fieldName string) {
	csa.messageCounts[fieldName]++
}

// addRegionMetrics adds metrics for a matching region. If countField
// is non-empty, increments the count. If timeField is non-empty, adds
// the duration to the total time.
func (csa *SummaryAccumulator) addRegionMetrics(countField string, timeField string, duration float64) {
	if len(countField) > 0 {
		csa.regionCounts[countField]++
	}
	if len(timeField) > 0 {
		csa.regionTimes[timeField] += duration
	}
}

// toMap converts the accumulated metrics into a single map suitable
// for JSON marshaling. The map contains all non-zero values across
// all metric types (message counts, region counts, region times).
func (csa *SummaryAccumulator) toMap() map[string]interface{} {
	result := make(map[string]interface{})

	for fieldName, count := range csa.messageCounts {
		if count > 0 {
			result[fieldName] = count
		}
	}

	for fieldName, count := range csa.regionCounts {
		if count > 0 {
			result[fieldName] = count
		}
	}

	for fieldName, time := range csa.regionTimes {
		if time > 0 {
			result[fieldName] = time
		}
	}

	return result
}

// apply__summary_message checks if a message matches any
// configured message pattern rules and increments the appropriate
// counters if matches are found.
func apply__summary_message(tr2 *trace2Dataset, message string) {
	// Check if summary is enabled
	if tr2.process.summary == nil {
		return
	}

	if tr2.rcvr_base == nil || tr2.rcvr_base.RcvrConfig == nil {
		return
	}

	css := tr2.rcvr_base.RcvrConfig.summary
	if css == nil {
		return
	}

	// Check message against all configured patterns
	for _, rule := range css.MessagePatterns {
		if strings.HasPrefix(message, rule.Prefix) {
			tr2.process.summary.incrementMessageCount(rule.FieldName)
		}
	}
}

// apply__summary_region checks if a region matches any
// configured region timer rules and aggregates the count and/or
// time if matches are found.
func apply__summary_region(tr2 *trace2Dataset, region *TrRegion) {
	// Check if summary is enabled
	if tr2.process.summary == nil {
		return
	}

	if tr2.rcvr_base == nil || tr2.rcvr_base.RcvrConfig == nil {
		return
	}

	css := tr2.rcvr_base.RcvrConfig.summary
	if css == nil {
		return
	}

	// Calculate region duration in seconds
	duration := region.lifetime.endTime.Sub(region.lifetime.startTime).Seconds()

	// Check region against all configured rules
	for _, rule := range css.RegionTimers {
		if region.category == rule.Category && region.label == rule.Label {
			tr2.process.summary.addRegionMetrics(
				rule.CountField,
				rule.TimeField,
				duration,
			)
		}
	}
}
