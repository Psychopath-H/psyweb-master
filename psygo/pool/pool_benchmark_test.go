package pool

import (
	"runtime"
	"sync"
	"testing"
	"time"
)

const (
	RunTimes           = 1e6
	PoolCap            = 5e4
	BenchParam         = 10
	DefaultExpiredTime = 10 * time.Second
)

const (
	_   = 1 << (10 * iota)
	KiB // 1024
	MiB // 1048576
	// GiB // 1073741824
	// TiB // 1099511627776             (超过了int32的范围)
	// PiB // 1125899906842624
	// EiB // 1152921504606846976
	// ZiB // 1180591620717411303424    (超过了int64的范围)
	// YiB // 1208925819614629174706176
)

const (
	Param    = 100
	PoolSize = 1000
	TestSize = 10000
)

var curMem uint64

// demoFunc 沉睡10ms
func demoFunc() {
	time.Sleep(time.Duration(BenchParam) * time.Millisecond)
}

func TestNoPool(t *testing.T) {
	var wg sync.WaitGroup
	for i := 0; i < RunTimes; i++ {
		wg.Add(1)
		go func() {
			demoFunc()
			wg.Done()
		}()
	}

	wg.Wait()
	mem := runtime.MemStats{}
	runtime.ReadMemStats(&mem)
	curMem = mem.TotalAlloc/MiB - curMem
	t.Logf("memory usage:%d MB", curMem)
}

func TestHasPool(t *testing.T) {
	var wg sync.WaitGroup
	pool, _ := NewPool(PoolCap, WithExpiryDuration(DefaultExpiredTime)) //这里要设置cap为max.MaxInt32的目的就是为了压测看看到底能创建多少个workers，所以是不会存在阻塞的情况的
	defer pool.Release()

	for i := 0; i < RunTimes; i++ {
		wg.Add(1)
		_ = pool.Submit(func() {
			demoFunc()
			wg.Done()
		})
	}
	wg.Wait()

	mem := runtime.MemStats{}
	runtime.ReadMemStats(&mem)
	curMem = mem.TotalAlloc/MiB - curMem
	t.Logf("memory usage:%d MB", curMem)
	t.Logf("running worker:%d", pool.Running())
	t.Logf("free worker:%d", pool.Free())

}
