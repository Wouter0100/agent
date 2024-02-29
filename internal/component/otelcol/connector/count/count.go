package count

import (
	"github.com/grafana/agent/internal/component"
	"github.com/grafana/agent/internal/component/otelcol"
	"github.com/grafana/agent/internal/component/otelcol/connector"
	"github.com/grafana/agent/internal/featuregate"
	"github.com/grafana/river"
	"github.com/open-telemetry/opentelemetry-collector-contrib/connector/countconnector"
	otelcomponent "go.opentelemetry.io/collector/component"
)

func init() {
	component.Register(component.Registration{
		Name:      "otelcol.connector.count",
		Stability: featuregate.StabilityExperimental,
		Args:      nil,
		Exports:   nil,
		Build: func(opts component.Options, args component.Arguments) (component.Component, error) {
			fact := countconnector.NewFactory()
			return connector.New(opts, fact, args.(Arguments))
		},
	})
}

// Arguments configures the otelcol.connector.count component.
type Arguments struct {
	Spans      []MetricInfo `river:"span,block,optional"`
	SpanEvents []MetricInfo `river:"spanevent,block,optional"`
	Metrics    []MetricInfo `river:"metric,block,optional"`
	DataPoints []MetricInfo `river:"datapoint,block,optional"`
	Logs       []MetricInfo `river:"log,block,optional"`

	// Output configures where to send processed data. Required.
	Output *otelcol.ConsumerArguments `river:"output,block"`
}

var (
	_ river.Validator     = (*Arguments)(nil)
	_ river.Defaulter     = (*Arguments)(nil)
	_ connector.Arguments = (*Arguments)(nil)
)

// ConnectorType implements connector.Arguments.
func (Arguments) ConnectorType() int {
	return connector.ConnectorAnyToMetrics
}

// Convert implements connector.Arguments.
func (args Arguments) Convert() (otelcomponent.Config, error) {
	return &countconnector.Config{
		Spans:      convertMetricInfo(args.Spans),
		SpanEvents: convertMetricInfo(args.SpanEvents),
		Metrics:    convertMetricInfo(args.Metrics),
		DataPoints: convertMetricInfo(args.DataPoints),
		Logs:       convertMetricInfo(args.Logs),
	}, nil
}

func convertMetricInfo(mi []MetricInfo) map[string]countconnector.MetricInfo {
	ret := make(map[string]countconnector.MetricInfo)
	for _, metricInfo := range mi {
		var attrConfigs []countconnector.AttributeConfig
		for _, ac := range metricInfo.Attributes {
			a := countconnector.AttributeConfig{
				Key:          ac.Key,
				DefaultValue: ac.DefaultValue,
			}
			attrConfigs = append(attrConfigs, a)
		}
		ret[metricInfo.Name] = countconnector.MetricInfo{
			Description: metricInfo.Description,
			Conditions:  metricInfo.Conditions,
			Attributes:  attrConfigs,
		}
	}
	return ret
}

// Exporters implements connector.Arguments.
func (Arguments) Exporters() map[otelcomponent.Type]map[otelcomponent.ID]otelcomponent.Component {
	return nil
}

// Extensions implements connector.Arguments.
func (Arguments) Extensions() map[otelcomponent.ID]otelcomponent.Component {
	return nil
}

// NextConsumers implements connector.Arguments.
func (args Arguments) NextConsumers() *otelcol.ConsumerArguments {
	return args.Output
}

// Validate implements river.Validator.
func (*Arguments) Validate() error {
	// TODO
	return nil
}

// SetToDefault implements river.Defaulter.
func (args *Arguments) SetToDefault() {
	// this component requires user-defined conditions to generate metrics.
	*args = Arguments{}
}
