package config

import (
	"fmt"
	nested "github.com/antonfisher/nested-logrus-formatter"
	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
	"github.com/uber/jaeger-client-go"
	jaegercfg "github.com/uber/jaeger-client-go/config"
	"github.com/uber/jaeger-client-go/zipkin"
	"google.golang.org/grpc/metadata"
	"io"
	"os"
	"strings"
)

var (
	Log              = logrus.New()
	ZipkinPropagator zipkin.Propagator
)

func TraceInit(serviceName string) (opentracing.Tracer, io.Closer) {
	cfg := &jaegercfg.Configuration{
		Sampler: &jaegercfg.SamplerConfig{
			Type:  "const",
			Param: 1,
		},
		ServiceName: serviceName,
		Reporter: &jaegercfg.ReporterConfig{
			LogSpans: true,
			//LocalAgentHostPort: "jaeger-agent.istio-system:6831",
			LocalAgentHostPort: "127.0.0.1:6831",
		},
	}
	//ZipkinPropagator = zipkin.NewZipkinB3HTTPHeaderPropagator()
	//
	//tracer, closer, err := cfg.NewTracer(jaegercfg.Logger(jaeger.StdLogger),
	//	jaegercfg.ZipkinSharedRPCSpan(true),
	//	jaegercfg.Injector(opentracing.HTTPHeaders, ZipkinPropagator),
	//	jaegercfg.Extractor(opentracing.HTTPHeaders, ZipkinPropagator))

	tracer, closer, err := cfg.NewTracer(jaegercfg.Logger(jaeger.StdLogger))
	if err != nil {
		panic(fmt.Sprintf("ERROR: cannot init Jaeger: %v\n", err))
	}
	return tracer, closer
}

type MDReaderWriter struct {
	metadata.MD
}

// ForeachKey 实现opentracing.TextMapReader
func (c MDReaderWriter) ForeachKey(handler func(key, val string) error) error {
	for k, vs := range c.MD {
		for _, v := range vs {
			if err := handler(k, v); err != nil {
				return err
			}
		}
	}
	return nil
}

// Set 实现 opentracing.TextMapWriter 接口
func (c MDReaderWriter) Set(key, val string) {
	key = strings.ToLower(key)
	c.MD[key] = append(c.MD[key], val)
}

func init() {
	Log.SetLevel(logrus.DebugLevel)
	Log.SetOutput(os.Stdout)
	Log.SetFormatter(&nested.Formatter{
		HideKeys:        true,
		TimestampFormat: "2006-01-02 15:04:05",
		FieldsOrder:     []string{"component", "x-request-id"},
	})
}
