package message

import (
	"errors"
	"sort"

	"github.com/sohaha/zlsgo/zarray"
	"github.com/sohaha/zlsgo/zjson"
	"github.com/sohaha/zlsgo/ztype"
	"github.com/zlsgo/zllm/runtime"
)

// OutputFormat 定义输出格式接口
type OutputFormat interface {
	Parse([]byte) (any, error)
	Format(string) (string, error)
	String() string
}

// outputJSONFormat JSON格式输出器
type outputJSONFormat ztype.Map

// defaultOutputFormatText 默认JSON输出格式
var defaultOutputFormatText OutputFormat = outputJSONFormat{"Assistant": "{}"}

// Parse 解析JSON响应
func (p outputJSONFormat) Parse(resp []byte) (any, error) {
	if len(p) == 0 {
		return resp, nil
	}

	j := zjson.ParseBytes(runtime.ParseContent(resp))
	if !j.IsObject() {
		return nil, errors.New("invalid response")
	}

	mp := ztype.Map{}
	for k := range p {
		mp[k] = j.Get(k).String()
	}

	return mp, nil
}

// String 返回格式字符串
func (p outputJSONFormat) String() string {
	if len(p) == 0 {
		return ztype.Map(p).Get("").String()
	}
	return ztype.ToString(ztype.Map(p))
}

// Format 格式化字符串
func (p outputJSONFormat) Format(str string) (string, error) {
	if len(p) == 0 {
		return str, nil
	}
	return ztype.ToString(p), nil
}

// DefaultOutputFormat 返回默认输出格式
func DefaultOutputFormat() OutputFormat {
	return defaultOutputFormatText
}

// CustomOutputFormat 创建自定义输出格式
func CustomOutputFormat(format map[string]string) OutputFormat {
	mp := ztype.Map{}
	keys := zarray.Keys(format)
	sort.Strings(keys)
	for _, k := range keys {
		mp[k] = format[k]
	}
	return outputJSONFormat(mp)
}

// outputNilFormat 空格式输出器
type outputNilFormat struct{}

// Parse 空解析，直接返回nil
func (p outputNilFormat) Parse(resp []byte) (any, error) {
	return nil, nil
}

// Format 空格式化，返回原字符串
func (p outputNilFormat) Format(str string) (string, error) {
	return str, nil
}

// String 返回"nil"标识
func (p outputNilFormat) String() string {
	return "nil"
}

// NilOutputFormat 返回空格式输出器
func NilOutputFormat() OutputFormat {
	return outputNilFormat{}
}

// WithNilFormat 返回空格式输出器(便捷方法)
func WithNilFormat() OutputFormat {
	return NilOutputFormat()
}
