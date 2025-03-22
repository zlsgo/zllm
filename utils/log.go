package utils

import (
	"github.com/sohaha/zlsgo/zlog"
)

var log *zlog.Logger

// func init() {
// 	log = zlog.NewZLog(os.Stderr, "[LLMX] ", zlog.BitLevel, zlog.LogWarn, true, 4)
// }

func Log(v ...any) {
	if log == nil {
		return
	}
	log.Debug(v...)
}

func Warn(v ...any) {
	if log == nil {
		return
	}
	log.Warn(v...)
}
