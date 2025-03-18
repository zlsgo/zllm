package message

import (
	"testing"

	"github.com/sohaha/zlsgo"
	"github.com/sohaha/zlsgo/ztype"
)

func Test_outputJSONFormat_Parse(t *testing.T) {
	tt := zlsgo.NewTest(t)
	c := CustomOutputFormat(map[string]string{
		"姓名": "{}",
	})

	tt.Log(c.String())
	tt.Equal(`{"姓名":"{}"}`, c.String())

	data, err := c.Format(`{"姓名":"张三"}`)
	tt.Log(data)
	tt.NoError(err)
	tt.Equal(`{"姓名":"{}"}`, string(data))

	resp, err := c.Parse([]byte(`{"姓名":"张三"}`))
	tt.Log(resp)
	tt.NoError(err)
	tt.Equal(ztype.Map{"姓名": "张三"}, resp)

	resp, err = c.Parse([]byte("```json\n{\"姓名\":\"李四\"}\n```"))
	tt.Log(resp)
	tt.NoError(err)
	tt.Equal(ztype.Map{"姓名": "李四"}, resp)

	resp, err = c.Parse([]byte("测试\n```json\n{\"姓名\":\"无名\"}\n```"))
	tt.Log(resp, err)
	tt.EqualTrue(err != nil)
}
