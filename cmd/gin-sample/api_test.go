package main

import (
	"github.com/gavv/httpexpect"
	"github.com/opentracing/opentracing-go"
	"io"
	"net/http"
	"testing"
)

func getHttpExpect(t *testing.T) *httpexpect.Expect {

	return httpexpect.WithConfig(httpexpect.Config{
		Client: &http.Client{
			Transport: httpexpect.NewBinder(httpServer()),
			Jar:       httpexpect.NewJar(),
		},
		Reporter: httpexpect.NewAssertReporter(t),
		Printers: []httpexpect.Printer{
			httpexpect.NewDebugPrinter(t, true),
		},
	})
}

func TestProduct(t *testing.T) {
	var closer io.Closer
	tracer, closer := TraceInit("gin-sample-tracing")
	defer closer.Close()
	opentracing.SetGlobalTracer(tracer)

	e := getHttpExpect(t)

	e.GET("/api/product").Expect().Status(200)
	e.GET("/api/reviews").Expect().Status(200)
}
