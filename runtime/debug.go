package runtime

import (
	"github.com/sohaha/zlsgo/zlog"
	"github.com/sohaha/zlsgo/zutil"
)

var isDebug = zutil.NewBool(false)

// SetDebug 设置调试模式
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

// IsDebug 获取调试模式状态
func IsDebug() bool {
	return isDebug.Load()
}

// SetLog 设置自定义日志记录器
func SetLog(l *zlog.Logger) {
	log = l
}
