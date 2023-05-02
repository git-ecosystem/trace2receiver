//go:build windows
// +build windows

package trace2receiver

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/receiver"
)

func createTraces(_ context.Context,
	params receiver.CreateSettings,
	baseCfg component.Config,
	consumer consumer.Traces) (receiver.Traces, error) {

	if consumer == nil {
		return nil, component.ErrNilNextConsumer
	}

	logger := params.Logger
	trace2Cfg := baseCfg.(*Config)

	rcvr := &Rcvr_NamedPipe{
		Base: &Rcvr_Base{
			Logger:                   logger,
			TracesConsumer:           consumer,
			AllowCommandControlVerbs: trace2Cfg.AllowCommandControlVerbs,
		},
		NamedPipePath: trace2Cfg.NamedPipePath,
	}
	return rcvr, nil
}
