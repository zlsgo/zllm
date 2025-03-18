package utils

import (
	"os"

	"github.com/sohaha/zlsgo/zlog"
)

var log *zlog.Logger

func init() {
	log = zlog.NewZLog(os.Stderr, "[LLMX] ", zlog.BitLevel, zlog.LogWarn, true, 4)
}

func Log(v ...any) {
	log.Debug(v...)
}

func Warn(v ...any) {
	log.Warn(v...)
}
