package inlay

import (
	"github.com/sohaha/zlsgo/zlog"
)

var log *zlog.Logger

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
