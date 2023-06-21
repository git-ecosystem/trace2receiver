//go:build windows
// +build windows

package trace2receiver

import (
	"context"
	"os"
	"os/user"

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
			Logger:         logger,
			TracesConsumer: consumer,
			RcvrConfig:     trace2Cfg,
		},
		NamedPipePath: trace2Cfg.NamedPipePath,
	}
	return rcvr, nil
}

// Gather up any requested PII from the machine or
// possibly the connection from the client process.
// Add any requested PII data to `tr2.pii[]`.
func (tr2 *trace2Dataset) pii_gather(cfg *Config) {
	if cfg.PiiSettings != nil && cfg.PiiSettings.Include.Hostname {
		if h, err := os.Hostname(); err == nil {
			tr2.pii[string(Trace2PiiHostname)] = h
		}
	}

	if cfg.PiiSettings != nil && cfg.PiiSettings.Include.Username {
		// TODO For now, just lookup the current user.  This may
		// or may not be valid when the service is officially
		// installed.  Ideally we should get the user-id of the
		// client process.  Or, since most Windows systems are
		// single login, get the name of the owner of the console
		// and assume it doesn't change.

		if u, err := user.Current(); err == nil {
			tr2.pii[string(Trace2PiiUsername)] = u.Username
		}
	}
}
