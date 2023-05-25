package trace2receiver

import (
	"context"

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
