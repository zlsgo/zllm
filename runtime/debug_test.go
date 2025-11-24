package runtime

import (
	"testing"

	"github.com/sohaha/zlsgo/zlog"
)

func TestSetDebug(t *testing.T) {
	SetDebug(true)
	if !IsDebug() {
		t.Error("Expected debug to be true after SetDebug(true)")
	}

	SetDebug(false)
	if IsDebug() {
		t.Error("Expected debug to be false after SetDebug(false)")
	}
}

func TestIsDebug(t *testing.T) {
	SetDebug(false)
	if IsDebug() {
		t.Error("Expected initial debug state to be false")
	}

	SetDebug(true)
	if !IsDebug() {
		t.Error("Expected debug to be true after SetDebug(true)")
	}
}

func TestSetLog(t *testing.T) {
	customLog := zlog.New("[Custom] ")

	SetLog(customLog)

	SetDebug(true)
	if !IsDebug() {
		t.Error("Expected debug to work with custom logger")
	}
}
