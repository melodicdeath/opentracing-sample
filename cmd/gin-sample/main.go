package main

import (
	"context"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"github.com/uber/jaeger-client-go"
	jaegercfg "github.com/uber/jaeger-client-go/config"
	"google.golang.org/grpc"
	"io"
	"log"
	"net/http"
	"opentracing-sample/service"
	"time"
)

const (
	address     = "localhost:50051"
	defaultName = "world"
)

var (
	Conn *grpc.ClientConn
	err  error
)

func TraceInit(serviceName string) (opentracing.Tracer, io.Closer) {
	cfg := &jaegercfg.Configuration{
		Sampler: &jaegercfg.SamplerConfig{
			Type:  "const",
			Param: 1,
		},
		Reporter: &jaegercfg.ReporterConfig{
			LogSpans:           true,
			LocalAgentHostPort: "127.0.0.1:6831",
		},
	}
	tracer, closer, err := cfg.New(serviceName, jaegercfg.Logger(jaeger.StdLogger))
	if err != nil {
		panic(fmt.Sprintf("ERROR: cannot init Jaeger: %v\n", err))
	}
	return tracer, closer
}

func ConnectgRPCServer() {
	Conn, err = grpc.Dial(address, grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}

	c := service.NewGreeterClient(Conn)

	// Contact the server and print out its response.
	name := defaultName
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	r, err := c.SayHello(ctx, &service.HelloRequest{Name: name})
	if err != nil {
		log.Fatalf("could not greet: %v", err)
	}
	log.Printf("Greeting: %s", r.GetMessage())
}

func TracerWrapper(c *gin.Context) {
	//md := make(map[string]string)
	spanCtx, _ := opentracing.GlobalTracer().Extract(opentracing.HTTPHeaders, opentracing.HTTPHeadersCarrier(c.Request.Header))
	sp := opentracing.GlobalTracer().StartSpan(c.Request.URL.Path, opentracing.ChildOf(spanCtx))
	defer sp.Finish()

	//if err := opentracing.GlobalTracer().Inject(sp.Context(),
	//	opentracing.TextMap,
	//	opentracing.TextMapCarrier(md)); err != nil {
	//	zap.Error(err)
	//}
	if err := opentracing.GlobalTracer().Inject(
		sp.Context(),
		opentracing.HTTPHeaders,
		opentracing.HTTPHeadersCarrier(c.Request.Header)); err != nil {
		panic(err)
	}

	statusCode := c.Writer.Status()
	ext.HTTPStatusCode.Set(sp, uint16(statusCode))
	ext.HTTPMethod.Set(sp, c.Request.Method)
	ext.HTTPUrl.Set(sp, c.Request.URL.EscapedPath())
	if statusCode >= http.StatusInternalServerError {
		ext.Error.Set(sp, true)
	}

	//ctx := context.TODO()
	//ctx = opentracing.ContextWithSpan(ctx, sp)
	//ctx = metadata.NewContext(ctx, md)
	//c.Set(contextTracerKey, ctx)

	c.Set("ctx", opentracing.ContextWithSpan(context.Background(), sp))
	c.Next()
}

func httpServer() *gin.Engine {

	r := gin.Default()
	r.Use(TracerWrapper)
	//r.Use(ginzap.Ginzap(zap.L(), time.RFC3339, true))
	//r.Use(ginzap.RecoveryWithZap(zap.L(), true))
	r.GET("/api/product", getProduceDetails)
	r.GET("/api/reviews", getProductReviews)
	return r
}

func getProductReviews(c *gin.Context) {
	psc, _ := c.Get("ctx")
	ctx := psc.(context.Context)
	doPing1(ctx)
	doPing2(ctx)
}

func getProduceDetails(c *gin.Context) {
	psc, _ := c.Get("ctx")
	ctx := psc.(context.Context)
	reqSpan, _ := opentracing.StartSpanFromContext(ctx, "getProduceDetails")
	defer reqSpan.Finish()
}

func doPing1(ctx context.Context) {
	span, _ := opentracing.StartSpanFromContext(ctx, "doPing1")
	defer span.Finish()
	time.Sleep(time.Second)
	fmt.Println("pong")
}

func doPing2(ctx context.Context) {
	span, _ := opentracing.StartSpanFromContext(ctx, "doPing2")
	defer span.Finish()
	time.Sleep(time.Second)
	fmt.Println("pong")
}

func main() {
	ConnectgRPCServer()
	defer Conn.Close()

	var closer io.Closer
	tracer, closer := TraceInit("gin-sample-tracing")
	defer closer.Close()
	opentracing.SetGlobalTracer(tracer)

	r := httpServer()
	r.Run()
}
