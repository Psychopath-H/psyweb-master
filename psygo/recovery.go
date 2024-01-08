package psygo

import (
	"errors"
	"fmt"
	"github.com/Psychopath-H/psyweb-master/psygo/psyerror"
	"log"
	"net/http"
	"runtime"
	"strings"
)

// 在 trace() 中，调用了 runtime.Callers(3, pcs[:])，Callers 用来返回调用栈的程序计数器,
// 第 0 个 Caller 是 Callers 本身，第 1 个是上一层 trace，第 2 个是再上一层的 defer func。
// 因此，为了日志简洁一点，我们跳过了前 3 个 Caller。
// 接下来，通过 runtime.FuncForPC(pc) 获取对应的函数，在通过 fn.FileLine(pc) 获取到调用该函数的文件名和行号，打印在日志中。
func trace(message string) string {
	var pcs [32]uintptr //uintptr 是一个整数类型，通常用于存储函数指针或程序计数器的地址。
	n := runtime.Callers(3, pcs[:])
	var str strings.Builder
	str.WriteString(message + "\nTraceback:")
	for _, pc := range pcs[:n] {
		fn := runtime.FuncForPC(pc)                           //获取与函数指针 pc 相关的函数信息，包括函数的名称等。
		file, line := fn.FileLine(pc)                         //这一行获取函数所在的文件和代码行号，并将它们分别存储在 file 和 line 变量中。
		str.WriteString(fmt.Sprintf("\n\t%s:%d", file, line)) //这一行将文件名和代码行号格式化为字符串，并将其附加到字符串构建器 sb 中，表示调用栈信息。
	}
	return str.String()
}

// Recovery 的实现非常简单，使用 defer 挂载上错误恢复的函数，在这个函数中调用 *recover()*，
// 捕获 panic，并且将堆栈信息打印在日志中，向用户返回 Internal Server Error。
func Recovery() HandlerFunc {
	return func(c *Context) {
		defer func() {
			if err := recover(); err != nil {
				errAssert := err.(error)
				if errAssert != nil {
					var psyError *psyerror.PsyError
					if errors.As(errAssert, &psyError) {
						psyError.ExecResult()
						c.Abort()
						return
					}
				}
				message := fmt.Sprintf("%s", err)
				c.Engine.Logger.Error(trace(message))
				log.Printf("%s\n\n", trace(message))
				c.Fail(http.StatusInternalServerError, "internal Server Error")
				c.Abort()
			}
		}()
		c.Next()
	}

}
