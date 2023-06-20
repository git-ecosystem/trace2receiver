package trace2receiver

import (
	"errors"
)

// FilterDetailLevel describes the amount of detail in the output
// OTLP that we will generate for a Git command.
type FilterDetailLevel int

const (
	DetailLevelUnset FilterDetailLevel = iota
	DetailLevelDrop
	DetailLevelSummary
	DetailLevelProcess
	DetailLevelVerbose
)

// All detail level names have leading "dl:" to help avoid
// cycles when resolving a custom ruleset name.
const (
	DetailLevelDropName    string = "dl:drop"
	DetailLevelSummaryName string = "dl:summary"
	DetailLevelProcessName string = "dl:process"
	DetailLevelVerboseName string = "dl:verbose"

	DetailLevelDefaultName string = DetailLevelSummaryName
)

// Convert a detail level name into a detail level id.
func getDetailLevel(dl_name string) (FilterDetailLevel, error) {
	switch dl_name {
	case DetailLevelDropName:
		return DetailLevelDrop, nil
	case DetailLevelSummaryName:
		return DetailLevelSummary, nil
	case DetailLevelProcessName:
		return DetailLevelProcess, nil
	case DetailLevelVerboseName:
		return DetailLevelVerbose, nil
	default:
		return DetailLevelUnset, errors.New("invalid detail level")
	}
}
