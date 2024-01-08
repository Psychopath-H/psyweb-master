package breaker

import (
	"errors"
	"sync"
	"time"
)

// State 状态
type State int

const (
	StateClosed State = iota
	StateHalfOpen
	StateOpen
)

// Counts 计数
type Counts struct {
	Requests             uint32 //请求数量
	TotalSuccesses       uint32 //总成功数
	TotalFailures        uint32 //总失败数
	ConsecutiveSuccesses uint32 //连续成功数量
	ConsecutiveFailures  uint32 //连续失败数量
}

func (c *Counts) OnRequest() {
	c.Requests++
}

func (c *Counts) OnSuccess() {
	c.TotalSuccesses++
	c.ConsecutiveSuccesses++
	c.ConsecutiveFailures = 0
}

func (c *Counts) OnFail() {
	c.TotalFailures++
	c.ConsecutiveFailures++
	c.ConsecutiveSuccesses = 0
}

func (c *Counts) Clear() {
	c.Requests = 0
	c.TotalSuccesses = 0
	c.ConsecutiveSuccesses = 0
	c.TotalFailures = 0
	c.ConsecutiveFailures = 0
}

type Settings struct {
	Name          string                                  //名字
	MaxRequests   uint32                                  //最大请求数
	Interval      time.Duration                           //间隔时间
	Timeout       time.Duration                           //超时时间
	ReadyToTrip   func(counts Counts) bool                //执行熔断
	OnStateChange func(name string, from State, to State) //状态变更
	IsSuccess     func(err error) bool                    //是否成功
	Fallback      func(err error) (any, error)
}

// CircuitBreaker 断路器
type CircuitBreaker struct {
	name          string                                  //名字
	maxRequests   uint32                                  //最大请求数 当连续请求成功数大于此时 断路器关闭
	interval      time.Duration                           //熔断器 自我更新 的间隔时间
	timeout       time.Duration                           //熔断器由 开->半开 的超时时间
	readyToTrip   func(counts Counts) bool                //根据counts状态决定是否执行熔断
	isSuccess     func(err error) bool                    //是否成功
	onStateChange func(name string, from State, to State) //状态变更函数，当熔断器状态发生变更时，触发的函数

	mutex      sync.Mutex
	state      State                        //状态
	generation uint64                       //代 状态变更 new一个
	counts     Counts                       //数量
	expiry     time.Time                    //到期时间 检查是否从开->半开
	fallback   func(err error) (any, error) //降级函数，当熔断器开启的时候，针对业务的降级处理
}

// NewGeneration 更新熔断器状态
func (cb *CircuitBreaker) NewGeneration() {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()
	cb.generation++
	cb.counts.Clear()
	var zero time.Time
	switch cb.state {
	case StateClosed:
		if cb.interval == 0 {
			cb.expiry = zero
		} else {
			cb.expiry = time.Now().Add(cb.interval)
		}
	case StateOpen:
		cb.expiry = time.Now().Add(cb.timeout)
	case StateHalfOpen:
		cb.expiry = zero
	}
}

// NewCircuitBreaker 根据传入设置新建一个熔断器，并做一些默认设置
func NewCircuitBreaker(st Settings) *CircuitBreaker {
	cb := new(CircuitBreaker)
	cb.name = st.Name
	cb.onStateChange = st.OnStateChange
	cb.fallback = st.Fallback
	if st.MaxRequests == 0 {
		cb.maxRequests = 2
	} else {
		cb.maxRequests = st.MaxRequests
	}
	if st.Interval == 0 {
		cb.interval = time.Duration(0) * time.Second
	} else {
		cb.interval = st.Interval
	}

	if st.Timeout == 0 {
		//断路器 开 -> 半开 需要的时间
		cb.timeout = time.Duration(10) * time.Second
	} else {
		cb.timeout = st.Timeout
	}
	if st.ReadyToTrip == nil {
		cb.readyToTrip = func(counts Counts) bool {
			return counts.ConsecutiveFailures > 5
		}
	} else {
		cb.readyToTrip = st.ReadyToTrip
	}
	if st.IsSuccess == nil {
		cb.isSuccess = func(err error) bool {
			return err == nil
		}
	} else {
		cb.isSuccess = st.IsSuccess
	}
	cb.NewGeneration()
	return cb
}

func (cb *CircuitBreaker) Execute(req func() (any, error)) (any, error) {
	//请求之前 做一个判断 是否执行断路器
	err, generation := cb.beforeRequest()
	if err != nil {
		//发生错误的时候 设置降级方法 进行执行
		if cb.fallback != nil {
			return cb.fallback(err)
		}
		return nil, err
	}
	//这个代表一个请求
	result, err := req()
	cb.counts.OnRequest()
	//请求之后，做一个判断，当前的状态是否需要变更
	cb.afterRequest(generation, cb.isSuccess(err))
	return result, err
}

// beforeRequest 在有请求进来 之前 根据熔断器判断是否允许继续执行业务
func (cb *CircuitBreaker) beforeRequest() (error, uint64) {
	//判断一下当前的状态 在做处置 断路器如果是打开状态 直接返回err
	now := time.Now()
	state, generation := cb.currentState(now)
	if state == StateOpen {
		return errors.New("server are melted, Please try it later"), generation
	}
	if state == StateHalfOpen {
		if cb.counts.Requests > cb.maxRequests {
			return errors.New("too much requests"), generation
		}
	}
	return nil, generation
}

// afterRequest 请求完成后，判断是否需要做状态变更
func (cb *CircuitBreaker) afterRequest(before uint64, success bool) {
	now := time.Now()
	state, generation := cb.currentState(now)
	if generation != before {
		return
	}
	if success {
		cb.OnSuccess(state)
	} else {
		cb.OnFail(state)
	}
}

// currentState 返回了熔断器当前的状态
func (cb *CircuitBreaker) currentState(now time.Time) (State, uint64) {

	switch cb.state {
	case StateClosed: //熔断器处于关闭状态
		if !cb.expiry.IsZero() && cb.expiry.Before(now) { //如果熔断器设置了过期时间且，已经过了过期时间，那么就执行更新熔断器操作
			cb.NewGeneration()
		}
	case StateOpen:
		if cb.expiry.Before(now) { //熔断器开得时间够长了，已经超过了过期时间，由开->半开
			cb.SetState(StateHalfOpen)
		}
	}
	return cb.state, cb.generation
}

// SetState 设置熔断器当前的状态
func (cb *CircuitBreaker) SetState(target State) {
	if cb.state == target {
		return
	}
	before := cb.state
	cb.state = target
	//状态变更之后 应该重新计数
	cb.NewGeneration()

	if cb.onStateChange != nil {
		cb.onStateChange(cb.name, before, target)
	}
}

// OnSuccess 请求被正确执行，根据当前状态判断是否需要切换状态
func (cb *CircuitBreaker) OnSuccess(state State) {
	switch state {
	case StateClosed:
		cb.counts.OnSuccess()
	case StateHalfOpen:
		cb.counts.OnSuccess()
		if cb.counts.ConsecutiveSuccesses > cb.maxRequests {
			cb.SetState(StateClosed)
		}
	}
}

// OnFail 请求未被正确执行，根据当前状态判断是否需要切换状态
func (cb *CircuitBreaker) OnFail(state State) {
	switch state {
	case StateClosed: // 熔断器处于关闭状态，有请求没能被正确处理，检查一下是否需要切换到熔断器开启状态
		cb.counts.OnFail()
		if cb.readyToTrip(cb.counts) {
			cb.SetState(StateOpen)
		}
	case StateHalfOpen: //熔断器半开状态 居然 还是有请求没能被正确处理，那就切换到全开状态
		cb.SetState(StateOpen)
	}
}
