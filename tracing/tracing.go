package tracing

import (
	"bytes"
	"io"
	"log"

	"github.com/opentracing/opentracing-go"
	"github.com/uber/jaeger-lib/metrics"

	jaeger "github.com/uber/jaeger-client-go"
	jaegercfg "github.com/uber/jaeger-client-go/config"
	jaegerlog "github.com/uber/jaeger-client-go/log"
)

// TraceMsg will be used as an io.Writer and io.Reader for the span's context and
// the payload. The span will have to be written first and read first.
type TraceMsg struct {
	bytes.Buffer
}

// InitTracing handles the common tracing setup functionality, and keeps
// implementation specific (Jaeger) configuration here.
func InitTracing() (opentracing.Tracer, io.Closer) {
	// Sample configuration for testing. Use constant sampling to sample every trace
	// and enable LogSpan to log every span via configured Logger.
	cfg := jaegercfg.Configuration{
		ServiceName: "Antidote",
		Sampler: &jaegercfg.SamplerConfig{
			Type:  jaeger.SamplerTypeConst,
			Param: 1,
		},
		Reporter: &jaegercfg.ReporterConfig{
			LogSpans: true,
			// CollectorEndpoint: "jaeger",
		},
	}

	// Example logger and metrics factory. Use github.com/uber/jaeger-client-go/log
	// and github.com/uber/jaeger-lib/metrics respectively to bind to real logging and metrics
	// frameworks.
	jLogger := jaegerlog.StdLogger
	jMetricsFactory := metrics.NullFactory

	// Initialize tracer with a logger and a metrics factory
	tracer, closer, err := cfg.NewTracer(
		jaegercfg.Logger(jLogger),
		jaegercfg.Metrics(jMetricsFactory),
	)
	if err != nil {
		log.Fatalf("couldn't setup tracing: %v", err)
	}
	return tracer, closer
}
