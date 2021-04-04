package services

import (
	"bytes"
	"io"

	ot "github.com/opentracing/opentracing-go"
	log "github.com/sirupsen/logrus"

	"github.com/nats-io/nats.go"
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

// NewTraceMsg creates a trace msg from a NATS message's data payload.
func NewTraceMsg(m *nats.Msg) *TraceMsg {
	b := bytes.NewBuffer(m.Data)
	return &TraceMsg{*b}
}

// InitTracing handles the common tracing setup functionality, and keeps
// implementation specific (Jaeger) configuration here.
func InitTracing(service string) (ot.Tracer, io.Closer) {

	log.Info("Antidote uses OpenTracing for detailed analysis of application behavior. Please consult the documentation for how to set up a supported collector")

	// Sample configuration for testing. Use constant sampling to sample every trace
	// and enable LogSpan to log every span via configured Logger.
	cfg := jaegercfg.Configuration{
		ServiceName: service,
		Sampler: &jaegercfg.SamplerConfig{
			Type:  jaeger.SamplerTypeConst,
			Param: 1,
		},
		Reporter: &jaegercfg.ReporterConfig{
			LogSpans: false,
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

// SafePayload is a helper function for ensuring strings meant to be used in Spans are kept to a
// reasonable size.
// The Jaeger UDP client will be unable to send spans that are too large. This value is somewhere
// in the ballpark of 64K bytes. We will limit large strings to a reasonable size, say 50K bytes.
//
// This isn't a silver bullet, however. There's nothing preventing you from using multiple strings
// from this function and still overloading a Span. Generally, strings from this function should
// only be used once - i.e. to describe a payload that contains logs from a failed pod
func SafePayload(payload string) string {
	// return original payload if already safe
	if len([]byte(payload)) <= 50000 {
		return payload
	}
	return string([]byte(payload)[len(payload)-50000:])
}
