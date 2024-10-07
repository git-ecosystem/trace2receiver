package trace2receiver

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/receiver"
	"go.uber.org/zap"
)

type Rcvr_Base struct {
	// These fields should be set in ctor() in platform_*.go:createTraces()
	// when it is called from factory.go:NewFactory().
	Settings        receiver.Settings
	Logger          *zap.Logger
	TracesConsumer  consumer.Traces
	MetricsConsumer consumer.Metrics
	LogsConsumer    consumer.Logs
	RcvrConfig      *Config

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

	if rcvr_base.RcvrConfig.AllowCommandControlVerbs {
		rcvr_base.Logger.Info("Command verbs are enabled")
	}

	if rcvr_base.RcvrConfig.piiSettings != nil {
		if rcvr_base.RcvrConfig.piiSettings.Include.Hostname {
			rcvr_base.Logger.Info("PII: Hostname logging is enabled")
		}
		if rcvr_base.RcvrConfig.piiSettings.Include.Username {
			rcvr_base.Logger.Info("PII: Username logging is enabled")
		}
	}
	return nil
}
