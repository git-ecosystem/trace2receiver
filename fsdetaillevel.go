package trace2receiver

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

	FSDetailLevelDefaultName string        = FSDetailLevelSummaryName
	FSDetailLevelDefault     FSDetailLevel = FSDetailLevelSummary
)

// Convert a detail level name or ruleset name into a detail level id.
func getDetailLevel(rs string) (FSDetailLevel, bool) {
	switch rs {
	case FSDetailLevelDropName:
		return FSDetailLevelDrop, true
	case FSDetailLevelSummaryName:
		return FSDetailLevelSummary, true
	case FSDetailLevelProcessName:
		return FSDetailLevelProcess, true
	case FSDetailLevelVerboseName:
		return FSDetailLevelVerbose, true
	default:
		return FSDetailLevelUnset, false
	}
}

// Convert a detail level id back into a detail level name.
func getDetailLevelName(dl FSDetailLevel) (string, bool) {
	switch dl {
	case FSDetailLevelDrop:
		return FSDetailLevelDropName, true
	case FSDetailLevelSummary:
		return FSDetailLevelSummaryName, true
	case FSDetailLevelProcess:
		return FSDetailLevelProcessName, true
	case FSDetailLevelVerbose:
		return FSDetailLevelVerboseName, true
	default:
		return "", false
	}
}
