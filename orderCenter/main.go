package main

import (
	"context"
	"github.com/Psychopath-H/psyweb-master/orderCenter/service"
	"github.com/Psychopath-H/psyweb-master/psygo"
	"log"
	"net/http"
	"rpc/xclient"
	"sync"
	"time"
)

func main() {
	registryAddr := "http://localhost:9999/_rpc_/registry"

	r := psygo.Default()

	v1 := r.Group("/orderCenter")
	{
		v1.GET("/sum", func(c *psygo.Context) {

			d := xclient.NewRegistryDiscovery(registryAddr, 0)
			xc := xclient.NewXClient(d, xclient.RandomSelect, nil)
			var reply int
			var err error
			defer func() { _ = xc.Close() }()
			// send request & receive response
			var wg sync.WaitGroup
			for i := 0; i < 5; i++ {
				wg.Add(1)
				go func(i int) {
					defer wg.Done()
					err = xc.Call(context.Background(), "GoodsService.Sum", &service.Args{Price1: i, Price2: i * i}, &reply)
					logSumPrint(err, reply)
				}(i)
			}
			wg.Wait()
			if err != nil {
				c.JSON(http.StatusInternalServerError, "rpc failed")
			} else {
				c.JSON(http.StatusOK, "rpc succeed")
			}
		})
	}

	{
		v1.GET("/sum_limiting", func(c *psygo.Context) {
			d := xclient.NewRegistryDiscovery(registryAddr, 0)
			xc := xclient.NewXClient(d, xclient.RandomSelect, nil)
			var reply int
			var err error
			defer func() { _ = xc.Close() }()
			// send request & receive response
			var wg sync.WaitGroup
			for i := 0; i < 100; i++ {
				wg.Add(1)
				go func(i int) {
					defer wg.Done()
					err = xc.Call(context.Background(), "GoodsService.Sum", &service.Args{Price1: i, Price2: i * i}, &reply)
					logSumPrint(err, reply)
				}(i)
			}
			wg.Wait()

			if err != nil {
				c.JSON(http.StatusInternalServerError, "rpc failed")
			} else {
				c.JSON(http.StatusOK, "rpc succeed")
			}
		})
	}

	{
		v1.GET("/sum_melted", func(c *psygo.Context) {
			d := xclient.NewRegistryDiscovery(registryAddr, 0)
			xc := xclient.NewXClient(d, xclient.RandomSelect, nil)
			var reply int
			var err error
			defer func() { _ = xc.Close() }()
			// send request & receive response
			var wg sync.WaitGroup
			for i := 0; i < 20; i++ {
				wg.Add(1)
				go func(i int) {
					defer wg.Done()
					if i < 10 {
						err = xc.Call(context.Background(), "GoodsService.Error", &service.Args{Price1: i, Price2: i * i}, &reply)
						logErrorPrint(err, reply)
					} else {
						err = xc.Call(context.Background(), "GoodsService.Sum", &service.Args{Price1: i, Price2: i * i}, &reply)
						logSumPrint(err, reply)
					}

				}(i)
			}

			time.Sleep(time.Second * 12)

			for i := 20; i < 40; i++ {
				wg.Add(1)
				go func(i int) {
					defer wg.Done()
					err = xc.Call(context.Background(), "GoodsService.Sum", &service.Args{Price1: i, Price2: i * i}, &reply)
					logSumPrint(err, reply)
				}(i)
			}
			wg.Wait()

			if err != nil {
				c.JSON(http.StatusInternalServerError, "rpc failed")
			} else {
				c.JSON(http.StatusOK, "rpc succeed")
			}
		})
	}

	_ = r.Run(":9002")

}

func logErrorPrint(err error, reply int) {
	if err != nil {
		log.Printf("%s error: %v", "GoodsService.Error", err)
	} else {
		log.Printf("%s success: reply = %d", "GoodsService.Error", reply)
	}
}

func logSumPrint(err error, reply int) {
	if err != nil {
		log.Printf("%s error: %v", "GoodsService.Sum", err)
	} else {
		log.Printf("%s success: reply = %d", "GoodsService.Sum", reply)
	}
}
