package main

import (
	"context"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"io"
	"log"
	"opentracing-sample/config"
	. "opentracing-sample/config"
	"opentracing-sample/service"
	"time"
)

const (
	address     = "localhost:50051"
	defaultName = "world"
	//address     = "grpc-server:50051"
)

var (
	Conn *grpc.ClientConn
	err  error
)

func ClientInterceptor(c *gin.Context, tracer opentracing.Tracer, spanContext opentracing.SpanContext) grpc.UnaryClientInterceptor {
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
		grpc.WithUnaryInterceptor(ClientInterceptor(c, opentracing.GlobalTracer(), parentSpanContext.(opentracing.SpanContext))))
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
}

func TracerWrapper(c *gin.Context) {
	spanCtx, _ := opentracing.GlobalTracer().Extract(opentracing.HTTPHeaders, opentracing.HTTPHeadersCarrier(c.Request.Header))
	sp := opentracing.GlobalTracer().StartSpan(c.Request.URL.Path, opentracing.ChildOf(spanCtx))

	defer sp.Finish()

	//head:map[Accept:[text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.9] Accept-Encoding:[gzip, deflate] Accept-Language:[zh-CN,zh;q=0.9,en;q=0.8,zh-TW;q=0.7,ja;q=0.6] Content-Length:[0] Cookie:[sidebar_collapsed=false; screenResolution=1536x864; _gitlab_session=e67f65e588be2730a3006cdae744e8a1] Dnt:[1] Upgrade-Insecure-Requests:[1] User-Agent:[Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/87.0.4280.88 Safari/537.36] X-B3-Sampled:[1] X-B3-Spanid:[3c5aa47b98941f3e] X-B3-Traceid:[2f4b419adf0f50953c5aa47b98941f3e] X-Envoy-Decorator-Operation:[gin-sample-tracing.istio-sample.svc.cluster.local:80/api/product] X-Envoy-Internal:[true] X-Envoy-Peer-Metadata:[ChoKCkNMVVNURVJfSUQSDBoKS3ViZXJuZXRlcwodCgxJTlNUQU5DRV9JUFMSDRoLMTcyLjE3LjAuMTIKlgIKBkxBQkVMUxKLAiqIAgodCgNhcHASFhoUaXN0aW8taW5ncmVzc2dhdGV3YXkKEwoFY2hhcnQSChoIZ2F0ZXdheXMKFAoIaGVyaXRhZ2USCBoGVGlsbGVyChkKBWlzdGlvEhAaDmluZ3Jlc3NnYXRld2F5CiEKEXBvZC10ZW1wbGF0ZS1oYXNoEgwaCjg0NWNjYzU5OTkKEgoHcmVsZWFzZRIHGgVpc3Rpbwo5Ch9zZXJ2aWNlLmlzdGlvLmlvL2Nhbm9uaWNhbC1uYW1lEhYaFGlzdGlvLWluZ3Jlc3NnYXRld2F5Ci8KI3NlcnZpY2UuaXN0aW8uaW8vY2Fub25pY2FsLXJldmlzaW9uEggaBmxhdGVzdAoaCgdNRVNIX0lEEg8aDWNsdXN0ZXIubG9jYWwKLwoETkFNRRInGiVpc3Rpby1pbmdyZXNzZ2F0ZXdheS04NDVjY2M1OTk5LWRwam05ChsKCU5BTUVTUEFDRRIOGgxpc3Rpby1zeXN0ZW0KXQoFT1dORVISVBpSa3ViZXJuZXRlczovL2FwaXMvYXBwcy92MS9uYW1lc3BhY2VzL2lzdGlvLXN5c3RlbS9kZXBsb3ltZW50cy9pc3Rpby1pbmdyZXNzZ2F0ZXdheQo5Cg9TRVJWSUNFX0FDQ09VTlQSJhokaXN0aW8taW5ncmVzc2dhdGV3YXktc2VydmljZS1hY2NvdW50CicKDVdPUktMT0FEX05BTUUSFhoUaXN0aW8taW5ncmVzc2dhdGV3YXk=] X-Envoy-Peer-Metadata-Id:[router~172.17.0.12~istio-ingressgateway-845ccc5999-dpjm9.istio-system~istio-system.svc.cluster.local] X-Forwarded-For:[172.17.0.1] X-Forwarded-Proto:[http] X-Request-Id:[ca10652c-7872-9c7a-83dc-15a735ace717]]
	log.Printf("head:%+v", c.Request.Header)

	if err := opentracing.GlobalTracer().Inject(
		sp.Context(),
		opentracing.HTTPHeaders,
		opentracing.HTTPHeadersCarrier(c.Request.Header)); err != nil {
		panic(err)
	}

	//statusCode := c.Writer.Status()
	//ext.HTTPStatusCode.Set(sp, uint16(statusCode))
	//ext.HTTPMethod.Set(sp, c.Request.Method)
	//ext.HTTPUrl.Set(sp, c.Request.URL.EscapedPath())
	//if statusCode >= http.StatusInternalServerError {
	//	ext.Error.Set(sp, true)
	//}

	//ctx := context.TODO()
	//ctx = opentracing.ContextWithSpan(ctx, sp)
	//ctx = metadata.NewContext(ctx, md)
	//c.Set(contextTracerKey, ctx)

	requestID := c.GetHeader("x-request-id")
	if len(requestID) == 0 {
		c.Set("x-request-id", uuid.New().String())
	}
	c.Set("x-request-id", requestID)
	c.Set("parentSpanCtx", sp.Context())
	c.Set("ctx", opentracing.ContextWithSpan(context.Background(), sp))

	c.Next()
}

func httpServer() *gin.Engine {
	r := gin.Default()
	r.Use(TracerWrapper)
	//r.Use(ginzap.Ginzap(zap.L(), time.RFC3339, true))8001
	//r.Use(ginzap.RecoveryWithZap(zap.L(), true))
	r.GET("/api/product", getProduceDetails)
	r.GET("/api/reviews", getProductReviews)
	return r
}

func getProduceDetails(c *gin.Context) {
	var XRequestID string
	value, exists := c.Get("x-request-id")
	if exists {
		XRequestID = value.(string)
	}

	config.Log.WithField("x-request-id", XRequestID).Info("获取产品信息")
	config.Log.WithField("x-request-id", XRequestID).Info("检查令牌")
	checkToken(c)
	config.Log.WithField("x-request-id", XRequestID).Info("令牌检查成功")
	psc, _ := c.Get("ctx")
	ctx := psc.(context.Context)

	config.Log.WithField("x-request-id", XRequestID).Info("读取redis")
	doSomething1(c, ctx)
	config.Log.WithField("x-request-id", XRequestID).Info("读取redis成功，读取mysql")
	doSomething2(c, ctx)
	config.Log.WithField("x-request-id", XRequestID).Info("读取mysql成功")

	c.String(200, XRequestID)
}

func getProductReviews(c *gin.Context) {
	psc, _ := c.Get("ctx")
	ctx := psc.(context.Context)
	reqSpan, _ := opentracing.StartSpanFromContext(ctx, "getProduceDetails")
	defer reqSpan.Finish()

	//spanContext := reqSpan.Context().(jaeger.SpanContext)
	//log.Println(spanContext.TraceID())
	//log.Println(spanContext.SpanID())
	checkToken(c)

	if value, exists := c.Get("x-request-id"); exists {
		c.String(200, value.(string))
	}
}

func doSomething1(c *gin.Context, ctx context.Context) {
	var XRequestID string
	value, exists := c.Get("x-request-id")
	if exists {
		XRequestID = value.(string)
	}

	config.Log.WithField("x-request-id", XRequestID).Info("连接redis成功,开始读取数据")
	span, _ := opentracing.StartSpanFromContext(ctx, "doSomething1 (进程内)")
	defer span.Finish()
	time.Sleep(time.Second)
	fmt.Println("pong")
}

func doSomething2(c *gin.Context, ctx context.Context) {
	var XRequestID string
	value, exists := c.Get("x-request-id")
	if exists {
		XRequestID = value.(string)
	}

	config.Log.WithField("x-request-id", XRequestID).Info("连接mysql成功,开始读取数据")
	span, _ := opentracing.StartSpanFromContext(ctx, "doSomething2 (进程内)")
	defer span.Finish()
	time.Sleep(time.Second)
	fmt.Println("pong")
}

func checkToken(c *gin.Context) context.Context {
	ConnectgRPCServer(c)
	client := service.NewGreeterClient(Conn)
	defer Conn.Close()

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
	var closer io.Closer
	tracer, closer := config.TraceInit("gin-sample-tracing")
	defer closer.Close()
	opentracing.SetGlobalTracer(tracer)

	r := httpServer()
	r.Run()
}
