package main

import (
	"context"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"github.com/uber/jaeger-client-go"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"io"
	"log"
	"net/http"
	"opentracing-sample/config"
	. "opentracing-sample/config"
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

func ClientInterceptor(tracer opentracing.Tracer, spanContext opentracing.SpanContext) grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string,
		req, reply interface{}, cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {

		span := opentracing.StartSpan(
			"call gRPC",
			opentracing.ChildOf(spanContext),
			opentracing.Tag{Key: string(ext.Component), Value: "gRPC"},
			ext.SpanKindRPCClient,
		)

		defer span.Finish()

		md, ok := metadata.FromOutgoingContext(ctx)
		if !ok {
			md = metadata.New(nil)
		} else {
			md = md.Copy()
		}

		// 在客户端拦截器中把 span 注入进去`
		err := tracer.Inject(span.Context(), opentracing.TextMap, MDReaderWriter{md})
		if err != nil {
			panic(err)
		}

		newCtx := metadata.NewOutgoingContext(ctx, md)
		err = invoker(newCtx, method, req, reply, cc, opts...)
		if err != nil {
			panic(err)
		}
		return err
	}
}

func ConnectgRPCServer(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*500)
	defer cancel()

	parentSpanContext, _ := c.Get("parentSpanCtx")
	Conn, err = grpc.DialContext(ctx, address, grpc.WithInsecure(), grpc.WithBlock(),
		grpc.WithUnaryInterceptor(ClientInterceptor(opentracing.GlobalTracer(), parentSpanContext.(opentracing.SpanContext))))
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}

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

	c.Set("parentSpanCtx", sp.Context())
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

	checkToken(c)

	psc, _ := c.Get("ctx")
	ctx := psc.(context.Context)

	doSomething1(ctx)
	doSomething2(ctx)
}

func getProduceDetails(c *gin.Context) {
	psc, _ := c.Get("ctx")
	ctx := psc.(context.Context)
	reqSpan, _ := opentracing.StartSpanFromContext(ctx, "getProduceDetails")
	defer reqSpan.Finish()

	spanContext := reqSpan.Context().(jaeger.SpanContext)
	log.Println(spanContext.TraceID())
	log.Println(spanContext.SpanID())
	checkToken(c)
}

func doSomething1(ctx context.Context) {
	span, _ := opentracing.StartSpanFromContext(ctx, "doSomething1 (进程内)")
	defer span.Finish()
	time.Sleep(time.Second)
	fmt.Println("pong")
}

func doSomething2(ctx context.Context) {
	span, _ := opentracing.StartSpanFromContext(ctx, "doSomething2 (进程内)")
	defer span.Finish()
	time.Sleep(time.Second)
	fmt.Println("pong")
}

func checkToken(c *gin.Context) context.Context {
	ConnectgRPCServer(c)
	client := service.NewGreeterClient(Conn)

	name := defaultName
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	r, err := client.SayHello(ctx, &service.HelloRequest{Name: name})
	if err != nil {
		log.Fatalf("could not greet: %v", err)
	}
	log.Printf("Greeting: %s", r.GetMessage())

	return ctx
}

func main() {

	defer Conn.Close()

	var closer io.Closer
	tracer, closer := config.TraceInit("gin-sample-tracing")
	defer closer.Close()
	opentracing.SetGlobalTracer(tracer)

	r := httpServer()
	r.Run()
}
