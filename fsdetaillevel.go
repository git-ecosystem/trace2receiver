package trace2receiver

import (
	"errors"
	"fmt"
)

// Filter Settings Detail Level describes the amount of detail
// in the output OTLP that we will generate for a Git command.
type FSDetailLevel int

const (
	FSDetailLevelUnset FSDetailLevel = iota
	FSDetailLevelDrop
	FSDetailLevelSummary
	FSDetailLevelProcess
	FSDetailLevelVerbose
)

// All detail level names have leading "dl:" to help avoid
// cycles when resolving a custom ruleset name.
const (
	FSDetailLevelDropName    string = "dl:drop"
	FSDetailLevelSummaryName string = "dl:summary"
	FSDetailLevelProcessName string = "dl:process"
	FSDetailLevelVerboseName string = "dl:verbose"

	FSDetailLevelDefaultName string = FSDetailLevelSummaryName
)

// Convert a detail level name or ruleset name into a detail level id.
func getDetailLevel(dl_name string) (FSDetailLevel, error) {
	switch dl_name {
	case FSDetailLevelDropName:
		return FSDetailLevelDrop, nil
	case FSDetailLevelSummaryName:
		return FSDetailLevelSummary, nil
	case FSDetailLevelProcessName:
		return FSDetailLevelProcess, nil
	case FSDetailLevelVerboseName:
		return FSDetailLevelVerbose, nil
	default:
		return FSDetailLevelUnset, errors.New(fmt.Sprintf("invalid detail level '%s'", dl_name))
	}
}
