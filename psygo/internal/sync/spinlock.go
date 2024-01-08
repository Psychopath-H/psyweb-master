package sync

import (
	"runtime"
	"sync"
	"sync/atomic"
)

type spinLock uint32

// NewSpinLock instantiates a spin-lock.
func NewSpinLock() sync.Locker {
	return new(spinLock)
}

const maxBackoff = 16

func (sl *spinLock) Lock() {
	backoff := 1
	for !atomic.CompareAndSwapUint32((*uint32)(sl), 0, 1) {
		// Leverage the exponential backoff algorithm, see https://en.wikipedia.org/wiki/Exponential_backoff.
		// 基于CAS机制，尝试获取锁，且使用指数退避算法来提供获取锁的成功率
		for i := 0; i < backoff; i++ {
			//runtime.Gosched()函数功能：使当前goroutine让出CPU时间片（“回避”），让其他的goroutine获得执行的机会。
			//当前的goroutine会在未来的某个时间点继续运行。
			//注意：当一个goroutine发生阻塞，Go会自动地把与该goroutine处于同一系统线程的其他goroutines转移到另一个系统线程上去，
			//以使这些goroutines不阻塞（从GMP模型角度来说，就是当与P绑定的M发生阻塞，P就与其解绑，然后与另一个空闲的M进行绑定 或者 去创建一个M进行绑定）。
			runtime.Gosched()
		}
		if backoff < maxBackoff {
			backoff <<= 1
		}
	}
}

func (sl *spinLock) Unlock() {
	atomic.StoreUint32((*uint32)(sl), 0)
}
