package inlay

import (
	"github.com/sohaha/zlsgo/zlog"
	"github.com/sohaha/zlsgo/zutil"
)

var isDebug = zutil.NewBool(false)

func SetDebug(debug bool) {
	isDebug.Store(debug)
	if log == nil {
		log = zlog.New("[LLMX] ")
	}
	if debug {
		log.SetLogLevel(zlog.LogDebug)
	} else {
		log.SetLogLevel(zlog.LogWarn)
	}
}

func IsDebug() bool {
	return isDebug.Load()
}

func SetLog(l *zlog.Logger) {
	log = l
}
