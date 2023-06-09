package trace2receiver

import "errors"

// There are some clients that we want to reject as soon as we
// learn their identity.  Primarily this is for daemon Git processes
// like `git fsmonitor--daemon run` and `git daemon` (and their dash
// name peers) that run (for days/months) in the background.  Since
// we do not generate the OTLP process span until the client drops
// the connection (ideally after the `atexit` event), we would be
// forced to collect massive state for the background daemon and
// bog down the entire telemetry service.  So let's reject them as
// soon as we identify them.
//
// There may be other background commands (like the new bundle server),
// so we may have to have more than one detection methods.
//
// At this point I'm just going to hard code the rejection.  I don't
// think it is worth adding code to `FilterSettings` to make this
// optional.

type RejectClientError struct {
	Err       error
	FSMonitor bool
}

func (rce *RejectClientError) Error() string {
	return rce.Err.Error()
}

// Is this Git command a `git fsmonitor--daemon` command?
//
// Check in `apply__cmd_name()` since we know FSMonitor sends a valid
// `cmd_name` event.  We really only need to reject `run` commands,
// but unfortunately, it does not send `cmd_mode` events, so we cannot
// distinguish between `run`, `start` and `stop`.
func IsFSMonitorDaemon(verb string) error {
	if verb == "fsmonitor--daemon" {
		return &RejectClientError{
			Err:       errors.New("rejecting telemetry from fsmonitor--daemon"),
			FSMonitor: true,
		}
	}

	return nil
}
