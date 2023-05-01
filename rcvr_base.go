package trace2receiver

import (
	"bytes"
	"context"
	"fmt"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.uber.org/zap"
)

type Rcvr_Base struct {
	// These fields should be set in ctor() in factory.go:createTraces()
	Logger                   *zap.Logger
	TracesConsumer           consumer.Traces
	MetricsConsumer          consumer.Metrics
	LogsConsumer             consumer.Logs
	AllowCommandControlVerbs bool

	// Component properties set in Start()
	ctx    context.Context
	host   component.Host
	cancel context.CancelFunc

	// Did we see at least one Trace2 event from the client?
	sawData bool
}

// `Start()` handles base-class portions of receiver initialization.
func (rcvr_base *Rcvr_Base) Start(unused_ctx context.Context, host component.Host) error {
	rcvr_base.host = host
	rcvr_base.ctx = context.Background()
	rcvr_base.ctx, rcvr_base.cancel = context.WithCancel(rcvr_base.ctx)

	if rcvr_base.AllowCommandControlVerbs {
		rcvr_base.Logger.Info("Command verbs are enabled")
	}

	return nil
}

var CommandControlVerbPrefix []byte = []byte("cc: ")

// Parse the raw line of text from the client and parse it.
//
// If is JSON, parse and validate it as a Trace2 event message.
// If it is a command/control from the helper tool, process it.
// If it is blank/empty or a "#-style" comment line, ignore it.
//
// Returns (nil, err) if we had an error.
// Returns (nil, nil) if we had command/control data.
// Returns (evt, nil) if we had event data.
func (rcvr_base *Rcvr_Base) evt_parse(rawLine []byte) (*TrEvent, error) {
	trimmed := bytes.TrimSpace(rawLine)

	if len(trimmed) == 0 || trimmed[0] == '#' {
		return nil, nil
	}

	if trimmed[0] == '{' {
		return parse_json(trimmed)
	}

	if bytes.HasPrefix(trimmed, CommandControlVerbPrefix) {
		if rcvr_base.AllowCommandControlVerbs {
			return nil, rcvr_base.do_command_verb(trimmed[len(CommandControlVerbPrefix):])
		} else {
			rcvr_base.Logger.Debug(fmt.Sprintf("command verbs are disabled: '%s'", trimmed))
			return nil, nil
		}
	}

	rcvr_base.Logger.Debug(fmt.Sprintf("unrecognized data stream verb: '%s'", trimmed))
	return nil, nil
}

func (rcvr_base *Rcvr_Base) do_command_verb(cmd []byte) error {
	rcvr_base.Logger.Debug(fmt.Sprintf("Command verb: '%s'", cmd))

	// TODO do something with the rest of the line and return.

	rcvr_base.Logger.Debug(fmt.Sprintf("invalid command verb: '%s'", cmd))
	return nil
}

// Process a raw line of text from the client.  This should contain a single
// line of Trace2 data in JSON format.  But we do allow command and control
// verbs (primarily for test and debug).
func (rcvr_base *Rcvr_Base) processRawLine(rawLine []byte, tr2 *trace2Dataset) error {

	rcvr_base.Logger.Debug(fmt.Sprintf("[dsid %06d] saw: %s", tr2.datasetId, rawLine))

	evt, err := rcvr_base.evt_parse(rawLine)
	if err != nil {
		return err
	}

	if evt != nil {
		rcvr_base.sawData = true

		err = evt_apply(tr2, evt)
		if err != nil {
			return err
		}
	}

	return nil
}

func (rcvr_base *Rcvr_Base) exportTraces(tr2 *trace2Dataset) error {
	if !rcvr_base.sawData {
		return nil
	}

	if !tr2.prepareDataset() {
		return nil
	}

	return rcvr_base.TracesConsumer.ConsumeTraces(rcvr_base.ctx, tr2.ToTraces())
}
