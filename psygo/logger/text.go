package logger

import (
	"fmt"
	"strings"
	"time"
)

type TextFormatter struct {
}

func (t *TextFormatter) Format(param *LogFormatterParam) string {
	now := time.Now()
	fieldString := ""
	if param.LoggerFields != nil {
		var sb strings.Builder
		count := 0
		num := len(param.LoggerFields)
		for k, v := range param.LoggerFields {
			fmt.Fprintf(&sb, "%s=%v", k, v)
			if count < num-1 {
				fmt.Fprintf(&sb, ",")
				count++
			}
		}
		fieldString = sb.String()
	}
	var msgInfo = "msg: "
	if param.Level == LevelError {
		msgInfo = "Error Cause By: "
	}
	var levelColor, msgColor string
	if param.IsOutputColor {
		levelColor = t.LevelColor(param.Level)
		msgColor = t.MsgColor(param.Level)
	}
	return fmt.Sprintf("[PSYLOGGER] %s%v%s | level = %s %s %s %s %s %v %s %s ",
		magenta, now.Format("2006/01/02 - 15:04:05"), reset,
		levelColor, param.Level.Level(), reset, msgColor, msgInfo, param.Msg, reset, fieldString,
	)

}

func (t *TextFormatter) LevelColor(level LoggerLevel) string {
	switch level {
	case LevelDebug:
		return blue
	case LevelInfo:
		return green
	case LevelError:
		return red
	default:
		return cyan
	}
}

func (t *TextFormatter) MsgColor(level LoggerLevel) string {
	switch level {
	case LevelDebug:
		return ""
	case LevelInfo:
		return ""
	case LevelError:
		return red
	default:
		return cyan
	}
}
