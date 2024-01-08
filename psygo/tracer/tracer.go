package tracer

import (
	"github.com/opentracing/opentracing-go"
	"github.com/uber/jaeger-client-go/config"
	"io"
)

func CreateTracer(serviceName string, samplerConfig *config.SamplerConfig, reporter *config.ReporterConfig, options ...config.Option) (opentracing.Tracer, io.Closer, error) {
	var cfg = config.Configuration{
		ServiceName: serviceName,
		Sampler:     samplerConfig,
		Reporter:    reporter,
	}
	tracer, closer, err := cfg.NewTracer(options...)
	return tracer, closer, err
}
