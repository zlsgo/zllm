package prompt_test

import (
	"context"
	"testing"

	"github.com/sohaha/zlsgo"
	"github.com/sohaha/zlsgo/zjson"
	"github.com/sohaha/zlsgo/ztype"
	"github.com/zlsgo/zllm/prompt"
)

func TestTranslateEcommerce(t *testing.T) {
	var (
		tt  = zlsgo.NewTest(t)
		got string
		err error
	)

	text := map[string]string{
		"title":       "卡维蒙沙发现代简约轻奢客厅意式极简<b>小户型</b>直排组合2024新款真皮沙发 直排 2.8米 标准款【接触面真皮】海绵座包",
		"description": "卡维蒙沙发是意大利品牌，采用优质真皮材质，简约轻奢的设计风格，适合小户型客厅使用。",
	}

	got, err = prompt.TranslateEcommerce(context.Background(), llm, text, "英文")
	tt.NoError(err, true)
	resp := zjson.Parse(got).Map()
	t.Log(ztype.ToMap(text))
	t.Log(resp)

	text = map[string]string{
		"title":       resp.Get("title").String(),
		"description": resp.Get("description").String(),
	}
	got, err = prompt.TranslateEcommerce(context.Background(), llm, text, "中文")
	tt.NoError(err, true)
	resp = zjson.Parse(got).Map()
	t.Log(resp)
}
