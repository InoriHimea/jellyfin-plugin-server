package logger

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"
)

type Level string

const (
	DEBUG Level = "DEBUG"
	INFO  Level = "INFO"
	WARN  Level = "WARN"
	ERROR Level = "ERROR"
)

type Logger struct {
	out     io.Writer
	jsonFmt bool
}

var std = &Logger{out: os.Stdout, jsonFmt: false}

func Init(jsonFormat bool) {
	std = &Logger{out: os.Stdout, jsonFmt: jsonFormat}
}

func (l *Logger) log(level Level, msg string, fields map[string]any) {
	if l.jsonFmt {
		entry := map[string]any{
			"time":  time.Now().UTC().Format(time.RFC3339),
			"level": string(level),
			"msg":   msg,
		}
		for k, v := range fields {
			entry[k] = v
		}
		b, _ := json.Marshal(entry)
		fmt.Fprintf(l.out, "%s\n", b)
		return
	}

	ts := time.Now().Format("2006-01-02 15:04:05")
	extra := ""
	for k, v := range fields {
		extra += fmt.Sprintf(" %s=%v", k, v)
	}
	fmt.Fprintf(l.out, "%s [%s] %s%s\n", ts, level, msg, extra)
}

func Info(msg string, fields ...map[string]any)  { std.log(INFO, msg, merge(fields)) }
func Warn(msg string, fields ...map[string]any)  { std.log(WARN, msg, merge(fields)) }
func Error(msg string, fields ...map[string]any) { std.log(ERROR, msg, merge(fields)) }
func Debug(msg string, fields ...map[string]any) { std.log(DEBUG, msg, merge(fields)) }

func merge(ms []map[string]any) map[string]any {
	if len(ms) == 0 {
		return nil
	}
	out := make(map[string]any)
	for _, m := range ms {
		for k, v := range m {
			out[k] = v
		}
	}
	return out
}
