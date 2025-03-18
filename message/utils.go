package message

import (
	"errors"
	"sort"

	"github.com/sohaha/zlsgo/zarray"
	"github.com/sohaha/zlsgo/zjson"
	"github.com/sohaha/zlsgo/ztype"
	"github.com/zlsgo/zllm/utils"
)

type OutputFormat interface {
	Parse([]byte) (any, error)
	Format(string) (string, error)
	String() string
}

type outputJSONFormat ztype.Map

var defaultOutputFormatText OutputFormat = outputJSONFormat{"Assistant": "{}"}

func (p outputJSONFormat) Parse(resp []byte) (any, error) {
	if len(p) == 0 {
		return resp, nil
	}

	j := zjson.ParseBytes(utils.ParseContent(resp))
	if !j.IsObject() {
		return nil, errors.New("invalid response")
	}

	mp := ztype.Map{}
	for k := range p {
		mp[k] = j.Get(k).String()
	}

	return mp, nil
}

func (p outputJSONFormat) String() string {
	if len(p) == 0 {
		return ztype.Map(p).Get("").String()
	}
	return ztype.ToString(ztype.Map(p))
}

func (p outputJSONFormat) Format(str string) (string, error) {
	if len(p) == 0 {
		return str, nil
	}
	return ztype.ToString(p), nil
}

func DefaultOutputFormat() OutputFormat {
	return defaultOutputFormatText
}

func CustomOutputFormat(format map[string]string) OutputFormat {
	mp := ztype.Map{}
	keys := zarray.Keys(format)
	sort.Strings(keys)
	for _, k := range keys {
		mp[k] = format[k]
	}
	return outputJSONFormat(mp)
}
