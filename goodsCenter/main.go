package main

import (
	"errors"
	"github.com/Psychopath-H/psyweb-master/goodsCenter/service"
	"github.com/Psychopath-H/psyweb-master/psygo"
	"github.com/Psychopath-H/psyweb-master/psygo/tracer"
	"github.com/opentracing/opentracing-go"
	"github.com/uber/jaeger-client-go"
	"github.com/uber/jaeger-client-go/config"
	"log"
	"net"
	"net/http"
	"rpc"
	"rpc/breaker"
	"rpc/registry"
	"sync"
	"time"
)

func startRegistry(wg *sync.WaitGroup) {
	l, _ := net.Listen("tcp", ":9999")
	registry.HandleHTTP()
	wg.Done()
	_ = http.Serve(l, nil)
}

func startServer(registryAddr string, wg *sync.WaitGroup) { // registryAddr -> "http://localhost:9999/_rpc_/registry"
	var gs service.GoodsService
	l, _ := net.Listen("tcp", ":0")
	server := rpc.NewServer()
	_ = server.Register(&gs)

	server.SetLimiter(20, 100, time.Millisecond*time.Duration(200))

	settings := breaker.Settings{}
	//降级实现
	settings.Fallback = func(err error) (any, error) {
		return "降级处理", errors.New("downgrading solution invoked")
	}
	server.CircuitBreaker = breaker.NewCircuitBreaker(settings)

	registry.Heartbeat(registryAddr, "tcp@"+l.Addr().String(), 0)
	wg.Done()
	server.Accept(l)
}

func main() {
	r := psygo.Default()
	registryAddr := "http://localhost:9999/_rpc_/registry"
	var wg sync.WaitGroup
	wg.Add(1)
	go startRegistry(&wg)
	wg.Wait()

	time.Sleep(time.Second)
	wg.Add(1)
	go startServer(registryAddr, &wg)
	wg.Wait()

	//使用链路追踪
	createTracer, closer, err := tracer.CreateTracer("goodsCenter", &config.SamplerConfig{
		Type:  jaeger.SamplerTypeConst,
		Param: 1,
	}, &config.ReporterConfig{
		LogSpans:          true,
		CollectorEndpoint: "http://192.168.100.100:14268/api/traces",
	}, config.Logger(jaeger.StdLogger))
	if err != nil {
		panic(err)
	}
	defer closer.Close()

	v1 := r.Group("/goodsCenter")
	{
		v1.GET("/jaeger", func(c *psygo.Context) {
			span := createTracer.StartSpan("Upstream Service")
			defer span.Finish()
			DownstreamService(createTracer, span)
			c.JSON(http.StatusOK, "jaeger invoked")
		})
	}
	_ = r.Run(":9001")

}

func DownstreamService(createTracer opentracing.Tracer, span opentracing.Span) {
	log.Println("DownstreamService is invoked")
	startSpan := createTracer.StartSpan("Downstream Service", opentracing.ChildOf(span.Context()))
	defer startSpan.Finish()
}
