package runtime

import (
	"github.com/sohaha/zlsgo/zlog"
)

var log *zlog.Logger

// Log 记录调试消息
func Log(v ...any) {
	if log == nil {
		return
	}
	log.Debug(v...)
}
