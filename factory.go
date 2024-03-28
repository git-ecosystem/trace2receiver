package trace2receiver

import (
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/receiver"
)

var (
	typeStr = component.MustNewType("trace2receiver")
)

const (
	stability = component.StabilityLevelStable
)

func createDefaultConfig() component.Config {
	return &Config{
		NamedPipePath:            "",
		UnixSocketPath:           "",
		AllowCommandControlVerbs: false,
		PiiSettingsPath:          "",
		piiSettings:              nil,
		FilterSettingsPath:       "",
		filterSettings:           nil,
	}
}

//func createMetrics(_ context.Context, params receiver.CreateSettings, baseCfg component.Config, consumer consumer.Metrics) (receiver.Metrics, error) {
//	return nil, nil
//}

//func createLogs(_ context.Context, params receiver.CreateSettings, baseCfg component.Config, consumer consumer.Logs) (receiver.Logs, error) {
//	return nil, nil
//}

// NewFactory creates a factory for trace2 receiver.
func NewFactory() receiver.Factory {
	return receiver.NewFactory(
		typeStr,
		createDefaultConfig,
		receiver.WithTraces(createTraces, stability),
	//receiver.WithMetrics(createMetrics, stability),
	//receiver.WithLogs(createLogs, stability),
	)
}
