package psygo

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"time"
)

const (
	greenBg   = "\033[97;42m"
	whiteBg   = "\033[90;47m"
	yellowBg  = "\033[90;43m"
	redBg     = "\033[97;41m"
	blueBg    = "\033[97;44m"
	magentaBg = "\033[97;45m"
	cyanBg    = "\033[97;46m"
	green     = "\033[32m"
	white     = "\033[37m"
	yellow    = "\033[33m"
	red       = "\033[31m"
	blue      = "\033[34m"
	magenta   = "\033[35m"
	cyan      = "\033[36m"
	reset     = "\033[0m"
)

// DefaultWriter 是默认的输出函数
var DefaultWriter io.Writer = os.Stdout

type LoggerConfig struct {
	Formatter LoggerFormatter
	Output    io.Writer
	IsColor   bool
}

type LoggerFormatter func(params *LogFormatterParams) string

// LogFormatterParams 包含了日志里所有的一些需要打印的参数
type LogFormatterParams struct {
	Request    *http.Request //context的请求
	TimeStamp  time.Time     //时间戳，服务器返回请求所花的时间
	StatusCode int           //状态码
	Latency    time.Duration //时间间隔，反映出服务器处理请求所花的时间
	ClientIP   net.IP
	Method     string
	Path       string

	IsOutputColor bool
}

func Logger() HandlerFunc {
	return LoggerWithConfig(LoggerConfig{})
}

func LoggerWithConfig(config LoggerConfig) HandlerFunc {
	formatter := config.Formatter
	if formatter == nil {
		formatter = defaultLogFormatter
	}
	out := config.Output
	OutputColor := false
	if out == nil {
		out = DefaultWriter
		OutputColor = true
	}
	return func(c *Context) {
		param := &LogFormatterParams{
			Request:       c.Req,
			IsOutputColor: OutputColor,
		}
		// Start timer
		start := time.Now()
		path := c.Req.URL.Path
		raw := c.Req.URL.RawQuery
		//执行业务
		c.Next()
		// stop timer
		param.TimeStamp = time.Now()
		param.Latency = param.TimeStamp.Sub(start)
		ip, _, _ := net.SplitHostPort(strings.TrimSpace(c.Req.RemoteAddr))
		param.ClientIP = net.ParseIP(ip)
		param.Method = c.Req.Method
		param.StatusCode = c.StatusCode

		if raw != "" {
			path = path + "?" + raw
		}

		param.Path = path

		fmt.Fprint(out, formatter(param))
	}
}

// StatusCodeColor 根据返回http状态设定状态码的颜色
func (p *LogFormatterParams) StatusCodeColor() string {
	code := p.StatusCode
	switch {
	case code >= http.StatusContinue && code < http.StatusOK:
		return whiteBg
	case code >= http.StatusOK && code < http.StatusMultipleChoices:
		return greenBg
	case code >= http.StatusMultipleChoices && code < http.StatusBadRequest:
		return whiteBg
	case code >= http.StatusBadRequest && code < http.StatusInternalServerError:
		return yellowBg
	default:
		return redBg
	}
}

func (p *LogFormatterParams) MethodColor() string {
	method := p.Method
	switch method {
	case http.MethodGet:
		return blueBg
	case http.MethodPost:
		return cyanBg
	case http.MethodPut:
		return yellowBg
	case http.MethodDelete:
		return redBg
	case http.MethodPatch:
		return greenBg
	case http.MethodHead:
		return magentaBg
	case http.MethodOptions:
		return whiteBg
	default:
		return reset
	}
}

func (p *LogFormatterParams) ResetColor() string {
	return reset
}

// defaultLogFormatter 是默认的日志打印函数
var defaultLogFormatter = func(params *LogFormatterParams) string {
	var statusCodeColor, resetColor, methodColor string
	if params.IsOutputColor {
		statusCodeColor = params.StatusCodeColor()
		resetColor = params.ResetColor()
		methodColor = params.MethodColor()
	}
	if params.Latency > time.Minute {
		params.Latency = params.Latency.Truncate(time.Second)
	}
	return fmt.Sprintf("[PSYGO] |%s %v %s| %s %3d %s |%s %13v %s| %15s  |%s %-7s %s %#v \n",
		magenta, params.TimeStamp.Format("2006/01/02 - 15:04:05"), resetColor,
		statusCodeColor, params.StatusCode, resetColor,
		red, params.Latency, resetColor,
		params.ClientIP,
		methodColor, params.Method, resetColor,
		params.Path,
	)
}
