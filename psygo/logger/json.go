package logger

import (
	"encoding/json"
	"fmt"
	"time"
)

type JsonFormatter struct {
	TimeDisplay bool
}

func (j *JsonFormatter) Format(param *LogFormatterParam) string {
	if param.LoggerFields == nil {
		param.LoggerFields = make(Fields)
	}
	now := time.Now()
	if j.TimeDisplay {
		param.LoggerFields["log_time"] = now.Format("2006/01/02 - 15:04:05")
	}
	param.LoggerFields["msg"] = param.Msg
	param.LoggerFields["log_level"] = param.Level.Level()
	marshal, err := json.Marshal(param.LoggerFields)
	if err != nil {
		panic(err)
	}
	return fmt.Sprintf("%s", string(marshal))
}
