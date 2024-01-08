package psyerror

type PsyError struct {
	err     error
	ErrFunc ErrorFunc
}

type ErrorFunc func(psyError *PsyError)

func Default() *PsyError {
	return &PsyError{}
}

func (e *PsyError) Error() string {
	return e.err.Error()
}

func (e *PsyError) Put(err error) {
	e.check(err)
}

func (e *PsyError) check(err error) {
	if err != nil {
		e.err = err
		panic(e)
	}
}

// Result 对外暴露出错误处理方法，可由框架使用者自行进行错误处理代码的编写
func (e *PsyError) Result(errFunc ErrorFunc) {
	e.ErrFunc = errFunc
}

// ExecResult 执行错误处理方法
func (e *PsyError) ExecResult() {
	e.ErrFunc(e)
}
