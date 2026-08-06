package log

import (
	"context"
	"fmt"
	"os"

	"code.byted.org/gopkg/logs/v2"
	"code.byted.org/gopkg/logs/v2/writer"
)

var (
	logger *logs.ByteDLogger
)

type InfoWriter = *logs.Log

func Init() {
	writers := make([]writer.LogWriter, 0, 2)
	writers = append(writers, writer.NewAsyncWriter(writer.NewFileWriter("/root/log/worker_v2.1.log", writer.Hourly), true))
	if os.Getenv("FOR_CLEANER_TEST") == "1" {
		writers = append(writers, writer.NewConsoleWriter())
	}
	logger = logs.NewByteDLogger(logs.SetWriter(logs.DebugLevel, writers...), logs.SetCallDepth(3))
}

func NewInfoWriter(ctx context.Context) InfoWriter {
	return logger.Info().With(ctx)
}

func Error(ctx context.Context, msg string, kvs ...KV) {
	logItem := logger.Error().With(ctx).Str(msg)
	for _, kv := range kvs {
		logItem = logItem.KV(kv.Key, kv.Value)
	}
	logItem.Emit()
}

func Info(ctx context.Context, msg string, kvs ...KV) {
	logItem := logger.Info().With(ctx).Str(msg)
	for _, kv := range kvs {
		logItem = logItem.KV(kv.Key, kv.Value)
	}
	logItem.Emit()
}

func Infof(ctx context.Context, msg string, args ...any) {
	logItem := logger.Info().With(ctx).Str(fmt.Sprintf(msg, args...))
	logItem.Emit()
}

func KVPair(key string, value interface{}) KV {
	return KV{Key: key, Value: value}
}

type KV struct {
	Key   string
	Value interface{}
}

func Println(anys ...any) {
	if logger == nil {
		fmt.Println(anys...)
		return
	}
	logger.Info().Str(fmt.Sprintln(anys...)).Emit()
}

func Printf(format string, anys ...any) {
	if logger == nil {
		fmt.Printf(format, anys...)
		return
	}
	logger.Info().Str(fmt.Sprintf(format, anys...)).Emit()
}
