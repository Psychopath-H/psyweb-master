package logger

import (
	"errors"
	"fmt"
	"github.com/Psychopath-H/psyweb-master/psygo/internal/psystrings"
	"io"
	"log"
	"os"
	"path"
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

type LoggerLevel int

const (
	LevelDebug LoggerLevel = iota
	LevelInfo
	LevelError
)

// LogFormatter 实现了这个接口可以以自己的形式输出日志
type LogFormatter interface {
	Format(param *LogFormatterParam) string
}

type LogFormatterParam struct {
	Level         LoggerLevel
	IsOutputColor bool
	LoggerFields  Fields
	Msg           any
}

type LogWriter struct {
	level  LoggerLevel
	writer io.Writer
}

type Fields map[string]any

// Logger 是给框架使用者提供的一个日志记录工具，偏于开发者使用其进行调试，记录日志信息
type Logger struct {
	Formatter    LogFormatter //格式化输出函数
	Level        LoggerLevel  //日志工具等级
	LogWriters   []*LogWriter //日志打印器
	LoggerFields Fields       //日志中含有的字段
	LogPath      string       //日志存放的路径
	LogFileSize  int64        //日志文件的设定大小，超过此大小则会进行分页操作
	//engine *psygo.Engine //日志工具要持有
}

// New 返回一个空Logger
func New() *Logger {
	return &Logger{}
}

// Default 返回默认的Logger
func Default() *Logger {
	logger := New()
	logger.Level = LevelDebug
	w := &LogWriter{ //使用了本框架提供的日志工具，那就会默认设置一个LogWriter，且一定会在控制台输出
		level:  LevelDebug,
		writer: os.Stdout,
	}
	logger.LogWriters = append(logger.LogWriters, w)
	logger.Formatter = &JsonFormatter{TimeDisplay: true}
	return logger
}

// SetLogPath 提供了设置要存储日志特定路径的方法
func (l *Logger) SetLogPath(logPath string) {
	l.LogPath = logPath
}

// SetLogWriter 设置io.Writer进LogWriter里
func (l *Logger) SetLogWriter(level LoggerLevel, writer ...io.Writer) error {
	if writer == nil {
		if l.LogPath != "" {
			logWriter := &LogWriter{
				level:  level,
				writer: FileWriter(path.Join(l.LogPath, "default.log")),
			}
			l.LogWriters = append(l.LogWriters, logWriter)
			return nil
		} else {
			return errors.New("LogPath should be set first")
		}
	}
	for _, w := range writer {
		logWriter := &LogWriter{
			level:  level,
			writer: w,
		}
		l.LogWriters = append(l.LogWriters, logWriter)
	}
	return nil
}

func (l *Logger) SetLogWriterOnFile(logPath string, fileName string, level LoggerLevel) {
	l.SetLogPath(logPath)
	l.LogWriters = append(l.LogWriters, &LogWriter{
		level:  level,
		writer: FileWriter(path.Join(l.LogPath, fileName)),
	})
}

// Print 根据当前Logger级别选择是否打印日志
func (l *Logger) Print(level LoggerLevel, msg any) error {
	if l.Level > level { //准入条件，不满足这个条件，根本不会进入日志系统
		return errors.New("loglevel is too low")
	}
	param := &LogFormatterParam{
		Level:        level,
		LoggerFields: l.LoggerFields,
		Msg:          msg,
	}
	var err error
	str := l.Formatter.Format(param)
	for _, logWriter := range l.LogWriters {
		if logWriter.writer == os.Stdout { //承接上面,如果使用了本框架提供的日志工具，那么默认插入的LogWriter一定会在控制台打印输出,或者设置的io.Writer是在控制台输出的，那无论输出等级，全部打印
			param.IsOutputColor = true
			str = l.Formatter.Format(param)
			_, _ = fmt.Fprintln(logWriter.writer, str)
			continue
		}
		if logWriter.level == -1 || level == logWriter.level { //本日志里的这个输出器要么是全打印输出器，或者打印器的级别等于本次日志调试等级
			_, err = fmt.Fprintln(logWriter.writer, str)
			l.CheckFileSize(logWriter)
		}
	}
	return err
}

// WithFields 返回一个带有字段的新Logger
func (l *Logger) WithFields(fields Fields) *Logger {
	l.LoggerFields = fields
	return l
}

// CheckFileSize 判断对应文件的大小
func (l *Logger) CheckFileSize(w *LogWriter) {
	logFile := w.writer.(*os.File)
	if logFile != nil {
		stat, err := logFile.Stat()
		if err != nil {
			log.Println(err)
			return
		}
		size := stat.Size()
		if l.LogFileSize <= 0 {
			l.LogFileSize = 100 << 20 //100M
		}
		if size >= l.LogFileSize {
			_, name := path.Split(stat.Name())           //分离目录和文件
			fileName := name[0:strings.Index(name, ".")] //把文件名字提取出来
			writer := FileWriter(path.Join(l.LogPath, psystrings.JoinStrings(fileName, ".", time.Now().UnixMilli(), ".log")))
			w.writer = writer //换个新的io.Writer流
		}

	}
}

// FileWriter 返回一个能以某种权限进行操作的输出流
func FileWriter(name string) io.Writer {
	w, _ := os.OpenFile(name, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0644)
	return w
}

func (level LoggerLevel) Level() string {
	switch level {
	case LevelDebug:
		return "DEBUG"
	case LevelInfo:
		return "INFO"
	case LevelError:
		return "ERROR"
	default:
		return ""
	}
}

func (l *Logger) Debug(msg any) error {
	return l.Print(LevelDebug, msg)

}

func (l *Logger) Info(msg any) error {
	return l.Print(LevelInfo, msg)
}

func (l *Logger) Error(msg any) error {
	return l.Print(LevelError, msg)
}
