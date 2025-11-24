# zllm 文档

> 简化的 LLM 集成包，支持多提供商、负载均衡和灵活提示管理。

## 🚀 安装指南

### 环境要求

- Go 1.23 或更高版本
- 支持 OpenAI、DeepSeek、Ollama、Anthropic 或 Gemini 的 API 访问权限

### 安装依赖

```bash
# 直接使用 go get
go get github.com/zlsgo/zllm
```

### 快速验证安装

创建一个简单的测试文件来验证安装：

```go
package main

import (
    "context"
    "fmt"
    "log"
    
    "github.com/zlsgo/zllm/agent"
    "github.com/zlsgo/zllm/message"
    "github.com/zlsgo/zllm"
)

func main() {
    // 使用 OpenAI (需要设置 OPENAI_API_KEY 环境变量)
    llm := agent.NewOpenAI(func(oa *agent.OpenAIOptions) {
        oa.Model = "gpt-4o-mini"
    })

    messages := message.NewMessages()
    messages.AppendUser("你好，请简单回复确认连接正常")

    resp, err := zllm.CompleteLLM(context.Background(), llm, messages)
    if err != nil {
        log.Fatalf("连接失败: %v", err)
    }

    fmt.Printf("✅ 连接成功! AI回复: %s\n", resp)
}
```

运行测试：
```bash
# 设置环境变量（支持多个 Key，用逗号分隔）
export OPENAI_API_KEY="key1,key2,key3"
# 或者
export DEEPSEEK_API_KEY="dskey1,dskey2"
# 或者
export ANTHROPIC_API_KEY="antkey1,antkey2,antkey3"
# 或者
export GEMINI_API_KEY="gemkey1,gemkey2"

# 可选：配置多个端点（用逗号分隔）
export OPENAI_BASE_URL="https://api.openai.com/v1,https://backup1.com/v1"

# 运行测试
go run test.go
```

**支持的环境变量：**

**OpenAI:**
- `OPENAI_API_KEY` - API Key（支持多个，逗号分隔）
- `OPENAI_MODEL` - 模型名称（默认 gpt-4.1）
- `OPENAI_BASE_URL` - 基础 URL（支持多个，逗号分隔）
- `OPENAI_API_URL` - API 路径（默认 /chat/completions）

**DeepSeek:**
- `DEEPSEEK_API_KEY` - API Key（支持多个，逗号分隔）
- `DEEPSEEK_MODEL` - 模型名称（默认 deepseek-chat）
- `DEEPSEEK_BASE_URL` - 基础 URL（支持多个，逗号分隔）

**Anthropic:**
- `ANTHROPIC_API_KEY` - API Key（支持多个，逗号分隔）
- `ANTHROPIC_MODEL` - 模型名称（默认 claude-3-5-sonnet-latest）
- `ANTHROPIC_BASE_URL` - 基础 URL（支持多个，逗号分隔）

**Gemini:**
- `GEMINI_API_KEY` - API Key（支持多个，逗号分隔）
- `GEMINI_MODEL` - 模型名称（默认 gemini-2.0-flash-exp）
- `GEMINI_BASE_URL` - 基础 URL（支持多个，逗号分隔）

**Ollama:**
- `OLLAMA_API_KEY` - API Key（可选）
- `OLLAMA_MODEL` - 模型名称（默认 qwen2.5:3b）
- `OLLAMA_BASE_URL` - 基础 URL（默认 http://localhost:11434）

### 核心组件

- **zllm 包**：核心 API 和负载均衡
- **agent 包**：LLM 提供商适配器
- **message 包**：消息和提示管理
- **runtime 包**：调试和日志功能

## ⚡ 快速开始

### 基础用法

```go
import "github.com/zlsgo/zllm"

llm := agent.NewOpenAI(func(oa *agent.OpenAIOptions) {
    oa.Model = "gpt-4o-mini"
    oa.APIKey = "your-api-key"
})

messages := message.NewMessages()
messages.AppendUser("你好，请介绍一下你自己")

resp, _ := zllm.CompleteLLM(ctx, llm, messages)
fmt.Printf("AI: %s\n", resp)
```

### 多 API Key 配置

```go
// 配置多个 API Key 和端点，实现负载均衡和故障转移
llm := agent.NewOpenAI(func(oa *agent.OpenAIOptions) {
    oa.APIKey = "key1,key2,key3"  // 多个 Key 用逗号分隔
    oa.BaseURL = "https://api.openai.com/v1,https://backup1.com/v1,https://backup2.com/v1"
    oa.Model = "gpt-4o-mini"
})

// 同样支持 DeepSeek、Anthropic、Gemini 和 Ollama
deepseek := agent.NewDeepseek(func(oa *agent.DeepseekOptions) {
    oa.APIKey = "dskey1,dskey2"  // 多个 DeepSeek Key
    oa.BaseURL = "https://api.deepseek.com,https://api.deepseek-backup.com"
})

anthropic := agent.NewAnthropic(func(oa *agent.AnthropicOptions) {
    oa.APIKey = "antkey1,antkey2"  // 多个 Anthropic Key
    oa.BaseURL = "https://api.anthropic.com,https://api.anthropic-backup.com"
})

gemini := agent.NewGemini(func(oa *agent.GeminiOptions) {
    oa.APIKey = "gemkey1,gemkey2"  // 多个 Gemini Key
    oa.BaseURL = "https://generativelanguage.googleapis.com,https://backup.googleapis.com"
})
```

**环境变量配置：**
```bash
# 多个 Key 用逗号分隔
export OPENAI_API_KEY="key1,key2,key3"
export OPENAI_BASE_URL="https://api.openai.com/v1,https://backup1.com/v1"
```

## ✨ 主要特性

- 🚀 **多提供商支持** - OpenAI、DeepSeek、Ollama、Anthropic、Gemini
- ⚡ **多 Key 负载均衡** - 自动随机选择 API Key 和端点，实现负载分散
- 🔄 **自动故障转移** - API Key 或端点失败时自动切换到其他可用选项
- 📝 **灵活提示** - 模板化提示管理
- 🔄 **流式输出** - 实时响应体验
- 🎯 **格式化输出** - JSON 和自定义格式
- 🛡️ **错误重试** - 智能重试机制
- 📊 **调试支持** - 完善的日志记录
- 🧰 **工具闭环** - 提示词触发工具调用，自动执行工具并续写

## 🔗 LLM 提供商对比

| 提供商    | 优势               | 适用场景           |
| --------- | ------------------ | ------------------ |
| OpenAI    | 性能最佳，功能丰富 | 复杂任务、创意工作 |
| Anthropic | 高质量对齐与推理   | 安全合规、代码写作 |
| DeepSeek  | 成本低，中文友好   | 日常对话、中文应用 |
| Gemini    | 多模态能力强       | 图像理解、创意生成 |
| Ollama    | 本地部署，隐私保护 | 离线环境、数据敏感 |

## 🎯 核心概念

### Agent - LLM 代理
与具体 LLM 服务交互的适配器：

```go
type LLM interface {
    Generate(ctx context.Context, data []byte) (resp *zjson.Res, err error)
    Stream(ctx context.Context, data []byte, callback func(string, []byte)) (done <-chan *zjson.Res, err error)
    PrepareRequest(messages *message.Messages, options ...func(ztype.Map) ztype.Map) (body []byte, err error)
    ParseResponse(*zjson.Res) (*Response, error)
}
```

### Message - 消息管理
处理对话历史和上下文：

```go
messages := message.NewMessages()
messages.AppendUser("用户输入")
messages.AppendAssistant("AI 回复")
```

### Prompt - 提示模板
支持变量替换和格式化：

```go
prompt := message.NewPrompt("用{{语言}}回答: {{问题}}", func(p *message.PromptOptions) {
    p.Placeholder = map[string]string{
        "语言": "中文",
        "问题": "什么是人工智能？",
    }
})
```

## 💡 常见使用场景

### 1. 简单对话
```go
resp, err := zllm.CompleteLLM(ctx, llm, messages)
```

### 2. 结构化输出  
```go
result, err := zllm.CompleteLLMJSON(ctx, llm, messages)
```

### 3. 负载均衡

负载均衡可以在多个 LLM 提供商之间自动分配请求，提高可靠性和性能。

#### 基础负载均衡配置

```go
// 创建多个提供商
openaiProvider := agent.NewOpenAI(func(oa *agent.OpenAIOptions) {
    oa.Model = "gpt-4o-mini"
    oa.APIKey = "openai-key-1"
    oa.MaxRetries = 2
})

deepseekProvider := agent.NewDeepseek(func(da *agent.DeepseekOptions) {
    da.Model = "deepseek-chat"
    da.APIKey = "deepseek-key-1"
    da.MaxRetries = 2
})

// 创建负载均衡器
balancer := zpool.NewBalancer([]agent.LLM{
    openaiProvider,
    deepseekProvider,
})

// 使用负载均衡
resp, err := zllm.BalancerCompleteLLM(ctx, balancer, messages)
```

#### 高级负载均衡策略

```go
package main

import (
    "context"
    "fmt"
    "time"
    
    "github.com/zlsgo/zllm"
    "github.com/zlsgo/zllm/agent"
    "github.com/zlsgo/zllm/message"
    "github.com/sohaha/zpool"
)

// 自定义负载均衡配置
func advancedLoadBalancing() {
    // 1. 多个 OpenAI 实例（不同 API Key）
    openai1 := agent.NewOpenAI(func(oa *agent.OpenAIOptions) {
        oa.Model = "gpt-4o-mini"
        oa.APIKey = "sk-key1"
        oa.BaseURL = "https://api.openai.com/v1"
        oa.MaxRetries = 1
    })
    
    openai2 := agent.NewOpenAI(func(oa *agent.OpenAIOptions) {
        oa.Model = "gpt-4o-mini"
        oa.APIKey = "sk-key2"
        oa.BaseURL = "https://api.openai.com/v1"
        oa.MaxRetries = 1
    })
    
    // 2. DeepSeek 实例
    deepseek := agent.NewDeepseek(func(da *agent.DeepseekOptions) {
        da.Model = "deepseek-chat"
        da.APIKey = "deepseek-key"
        da.MaxRetries = 1
    })
    
    // 3. Gemini 实例
    gemini := agent.NewGemini(func(go *agent.GeminiOptions) {
        go.Model = "gemini-2.0-flash-exp"
        go.APIKey = "gemini-key"
        go.MaxRetries = 1
    })
    
    // 4. Ollama 本地实例
    ollama := agent.NewOllama(func(oo *agent.OllamaOptions) {
        oo.Model = "llama2"
        oo.BaseURL = "http://localhost:11434"
        oo.MaxRetries = 1
    })
    
    // 创建负载均衡器
    balancer := zpool.NewBalancer([]agent.LLM{
        openai1,    // 主要服务 1
        deepseek,   // 备用服务（成本低）
        gemini,     // 多模态服务
        ollama,     // 本地备用（隐私保护）
    })
    
    // 使用负载均衡的 JSON 输出
    messages := message.NewMessages()
    messages.AppendUser("分析市场趋势并给出投资建议")
    
    ctx := zllm.WithTimeout(context.Background(), 60*time.Second)
    
    resp, err := zllm.BalancerCompleteLLMJSON(ctx, balancer, messages)
    if err != nil {
        fmt.Printf("负载均衡请求失败: %v\n", err)
        return
    }
    
    fmt.Printf("负载均衡响应: %+v\n", resp)
}

// 带健康检查的负载均衡
func healthCheckedLoadBalancing() {
    // 创建具有不同特性的提供商
    providers := []agent.LLM{
        // 高优先级：快速响应
        agent.NewOpenAI(func(oa *agent.OpenAIOptions) {
            oa.Model = "gpt-4o-mini"
            oa.APIKey = "fast-key"
            oa.Timeout = 10 * time.Second
        }),
        
        // 中优先级：成本效益
        agent.NewDeepseek(func(da *agent.DeepseekOptions) {
            da.Model = "deepseek-chat"
            da.APIKey = "economical-key"
            da.Timeout = 20 * time.Second
        }),
        
        // 中优先级：多模态
        agent.NewGemini(func(go *agent.GeminiOptions) {
            go.Model = "gemini-2.0-flash-exp"
            go.APIKey = "multimodal-key"
            go.Timeout = 25 * time.Second
        }),
        
        // 低优先级：本地备用
        agent.NewOllama(func(oo *agent.OllamaOptions) {
            oo.Model = "mistral"
            oo.BaseURL = "http://localhost:11434"
            oo.Timeout = 30 * time.Second
        }),
    }
    
    // 创建负载均衡器
    balancer := zpool.NewBalancer(providers)
    
    // 执行请求，自动故障转移
    messages := message.NewMessages()
    messages.AppendUser("生成一个简单的 Go 函数示例")
    
    // 请求会自动尝试所有提供商，直到成功或全部失败
    resp, err := zllm.BalancerCompleteLLM(context.Background(), balancer, messages)
    if err != nil {
        fmt.Printf("所有提供商都失败了: %v\n", err)
        return
    }
    
    fmt.Printf("成功获取响应: %s\n", resp)
}

// 负载均衡器使用场景示例
func loadBalancingScenarios() {
    fmt.Println("=== 负载均衡使用场景 ===")
    
    // 场景1：高可用性要求
    fmt.Println("\n1. 高可用性配置：")
    fmt.Println("- 多个云服务商备份")
    fmt.Println("- 本地服务作为最后备用")
    fmt.Println("- 自动故障转移")
    
    // 场景2：成本优化
    fmt.Println("\n2. 成本优化配置：")
    fmt.Println("- 优先使用低成本服务")
    fmt.Println("- 高峰期使用备用服务")
    fmt.Println("- 根据请求类型选择提供商")
    
    // 场景3：性能优化
    fmt.Println("\n3. 性能优化配置：")
    fmt.Println("- 地理位置优化")
    fmt.Println("- 响应时间优先级")
    fmt.Println("- 并发请求分发")
    
    // 场景4：多模态支持
    fmt.Println("\n4. 多模态配置：")
    fmt.Println("- Gemini 处理图像理解")
    fmt.Println("- OpenAI 处理代码生成")
    fmt.Println("- DeepSeek 处理中文对话")
}
```

#### 负载均衡最佳实践

1. **提供商选择策略**：
   - **主备用模式**：主要服务 + 备用服务
   - **成本优先**：优先使用低成本提供商
   - **性能优先**：根据响应时间选择
   - **功能优先**：根据特定功能选择（如多模态）

2. **容错配置**：
```go
// 为每个提供商设置合适的超时和重试
agent.NewOpenAI(func(oa *agent.OpenAIOptions) {
    oa.MaxRetries = 2           // 减少单点重试
    oa.Timeout = 30 * time.Second // 合理超时
})
```

3. **监控和日志**：
```go
// 启用调试模式监控各提供商状态
llm := agent.NewOpenAI(func(oa *agent.OpenAIOptions) {
    oa.OnMessage = func(chunk string, data []byte) {
        log.Printf("Provider response: %s", chunk)
    }
})
```

4. **动态配置**：
```go
// 可以根据业务需求动态调整提供商
func dynamicBalancer(isHighPriority bool, needsMultimodal bool) *zpool.Balancer[agent.LLM] {
    var providers []agent.LLM
    
    if needsMultimodal {
        providers = append(providers, agent.NewGemini(...))
    }
    
    if isHighPriority {
        providers = append(providers, agent.NewOpenAI(...))
    } else {
        providers = append(providers,
            agent.NewDeepseek(...),
            agent.NewOllama(...),
        )
    }
    
    return zpool.NewBalancer(providers)
}
```

### 4. 流式输出
```go
llm := agent.NewOpenAI(func(oa *agent.OpenAIOptions) {
    oa.Stream = true
    oa.OnMessage = func(chunk string, data []byte) {
        fmt.Print(chunk) // 实时输出
    }
})
```

### 5. 工具闭环（通过提示词触发 + 自动执行 + 续写）

工具闭环是 zllm 的核心功能之一，支持 LLM 自动调用外部工具并处理结果。

#### 基础示例：简单的 Echo 工具

```go
package main

import (
    "context"
    "fmt"
    "log"
    
    "github.com/zlsgo/zllm"
    "github.com/zlsgo/zllm/agent"
    "github.com/zlsgo/zllm/message"
)

func main() {
    // 1. 定义工具 schema（OpenAI 兼容格式）
    tools := []map[string]any{{
        "type": "function",
        "function": map[string]any{
            "name": "echo",
            "description": "回显用户输入的文本",
            "parameters": map[string]any{
                "type": "object",
                "properties": map[string]any{
                    "text": map[string]any{
                        "type": "string",
                        "description": "要回显的文本",
                    },
                },
                "required": []string{"text"},
            },
        },
    }}

    // 2. 创建工具执行器
    type echoRunner struct{}
    func (echoRunner) Run(ctx context.Context, name, args string) (string, error) {
        if name != "echo" {
            return "", fmt.Errorf("未知工具: %s", name)
        }
        // 在实际应用中，这里会解析 JSON 参数并执行相应逻辑
        return fmt.Sprintf("工具回显: %s", args), nil
    }

    // 3. 配置 LLM 和工具
    llm := agent.NewOpenAI(func(oa *agent.OpenAIOptions) {
        oa.Model = "gpt-4o-mini"
        oa.APIKey = "your-api-key"
    })

    ctx := zllm.WithToolRunner(context.Background(), echoRunner{})

    messages := message.NewMessages()
    messages.AppendUser("请使用 echo 工具回显这句话: 你好世界")

    // 4. 执行请求（会自动处理工具调用）
    resp, err := zllm.CompleteLLM(ctx, llm, messages, agent.WithToolCallHint(tools))
    if err != nil {
        log.Fatalf("请求失败: %v", err)
    }

    fmt.Printf("最终回复: %s\n", resp)
}
```

#### 进阶示例：多工具集成和错误处理

```go
package main

import (
    "context"
    "encoding/json"
    "fmt"
    "time"
    
    "github.com/sohaha/zlsgo/zjson"
    "github.com/zlsgo/zllm"
    "github.com/zlsgo/zllm/agent"
    "github.com/zlsgo/zllm/message"
)

// 定义工具参数结构体
type EchoArgs struct {
    Text string `json:"text"`
}

type WeatherArgs struct {
    City string `json:"city"`
}

// 复杂工具执行器
type advancedToolRunner struct{}

func (r *advancedToolRunner) Run(ctx context.Context, name, args string) (string, error) {
    switch name {
    case "echo":
        var echoArgs EchoArgs
        if err := json.Unmarshal([]byte(args), &echoArgs); err != nil {
            return "", fmt.Errorf("echo 工具参数解析失败: %w", err)
        }
        return fmt.Sprintf("回显结果: %s", echoArgs.Text), nil
        
    case "get_time":
        return time.Now().Format("2006-01-02 15:04:05"), nil
        
    case "get_weather":
        var weatherArgs WeatherArgs
        if err := json.Unmarshal([]byte(args), &weatherArgs); err != nil {
            return "", fmt.Errorf("天气工具参数解析失败: %w", err)
        }
        
        // 模拟天气 API 调用
        if weatherArgs.City == "" {
            return "", fmt.Errorf("城市参数不能为空")
        }
        
        return fmt.Sprintf("%s 当前天气: 晴天 25°C", weatherArgs.City), nil
        
    default:
        return "", fmt.Errorf("不支持的工具: %s", name)
    }
}

func advancedToolExample() {
    // 定义多个工具
    tools := []map[string]any{
        {
            "type": "function",
            "function": map[string]any{
                "name": "echo",
                "description": "回显用户输入的文本",
                "parameters": map[string]any{
                    "type": "object",
                    "properties": map[string]any{
                        "text": map[string]any{
                            "type": "string",
                            "description": "要回显的文本",
                        },
                    },
                    "required": []string{"text"},
                },
            },
        },
        {
            "type": "function", 
            "function": map[string]any{
                "name": "get_time",
                "description": "获取当前时间",
                "parameters": map[string]any{
                    "type": "object",
                    "properties": map[string]any{},
                },
            },
        },
        {
            "type": "function",
            "function": map[string]any{
                "name": "get_weather",
                "description": "获取指定城市的天气信息",
                "parameters": map[string]any{
                    "type": "object",
                    "properties": map[string]any{
                        "city": map[string]any{
                            "type": "string",
                            "description": "城市名称",
                        },
                    },
                    "required": []string{"city"},
                },
            },
        },
    }

    llm := agent.NewOpenAI(func(oa *agent.OpenAIOptions) {
        oa.Model = "gpt-4o-mini"
        oa.APIKey = "your-api-key"
    })

    // 自定义工具结果格式化器
    ctx := zllm.WithToolRunner(context.Background(), advancedToolRunner{})
    ctx = zllm.WithToolResultFormatter(ctx, func(results []zllm.ToolResult) string {
        var output string
        for _, result := range results {
            if result.Err != "" {
                output += fmt.Sprintf("❌ 工具 %s 执行失败: %s\n", result.Name, result.Err)
            } else {
                output += fmt.Sprintf("✅ 工具 %s 执行成功: %s\n", result.Name, result.Result)
            }
        }
        return output
    })
    
    // 设置最大工具调用迭代次数
    ctx = zllm.WithMaxToolIterations(ctx, 5)

    messages := message.NewMessages()
    messages.AppendUser("请帮我回显'Hello World'，然后获取当前时间和北京的天气")

    resp, err := zllm.CompleteLLM(ctx, llm, messages, agent.WithToolCallHint(tools))
    if err != nil {
        fmt.Printf("请求失败: %v\n", err)
        return
    }

    fmt.Printf("最终回复:\n%s\n", resp)
}
```

#### 便捷的 MapToolRunner 使用

```go
// 使用内置的 MapToolRunner 简化开发
func mapToolRunnerExample() {
    runner := zllm.NewMapToolRunner(map[string]zllm.MapToolHandler{
        "echo": func(ctx context.Context, args *zjson.Res) (string, error) {
            return args.Get("text").String(), nil
        },
        "time.now": func(ctx context.Context, args *zjson.Res) (string, error) {
            return time.Now().Format(time.RFC3339), nil
        },
        "calculator.add": func(ctx context.Context, args *zjson.Res) (string, error) {
            a := args.Get("a").Float()
            b := args.Get("b").Float()
            return fmt.Sprintf("%.2f", a+b), nil
        },
    })

    ctx := zllm.WithToolRunner(context.Background(), runner)
    
    // 正常的 LLM 调用...
}
```

#### 工具调用配置选项

```go
// 禁用工具调用
ctx := zllm.WithAllowTools(ctx, false)

// 自定义超时时间
ctx = zllm.WithTimeout(ctx, 120*time.Second)

// 设置最大工具迭代次数
ctx = zllm.WithMaxToolIterations(ctx, 10)
```

#### 工具调用流程说明

1. **工具定义**: 使用 OpenAI 兼容的 schema 格式定义工具
2. **工具执行**: LLM 决定调用工具时，框架自动调用对应的 `ToolRunner`
3. **结果处理**: 工具执行结果自动注入到对话中
4. **继续对话**: LLM 基于工具结果继续生成最终回复
5. **迭代控制**: 支持多轮工具调用，直到达到最大迭代次数或获得最终结果

## 🔑 多 API Key 最佳实践

### 1. 配置策略

**推荐配置数量：**
- **开发环境**: 1-2 个 Key
- **测试环境**: 2-3 个 Key
- **生产环境**: 3-5 个 Key

**配置示例：**
```go
// 生产环境推荐配置
llm := agent.NewOpenAI(func(oa *agent.OpenAIOptions) {
    // 使用不同账户或不同区域的 Key
    oa.APIKey = "primary-key,secondary-key,backup-key"
    // 配置主备端点
    oa.BaseURL = "https://api.openai.com/v1,https://api.openai-alt.com/v1"
    oa.MaxRetries = 3  // 配合多 Key 的重试策略
})
```

### 2. 负载分散策略

**随机选择 vs 轮询：**
```go
// 当前实现：随机选择（推荐）
// 每次请求随机选择一个 Key，确保负载均匀分布

// 轮询策略（可选实现）
// 按顺序轮换使用 Key，确保每个 Key 使用次数相近
```

**配额管理：**
```go
// 为不同业务场景配置不同的 Key
businessLlm := agent.NewOpenAI(func(oa *agent.OpenAIOptions) {
    oa.APIKey = "business-key1,business-key2"
})

devLlm := agent.NewOpenAI(func(oa *agent.OpenAIOptions) {
    oa.APIKey = "dev-key1,dev-key2"
})
```

### 3. 故障处理

**错误分类和处理：**
- **429 错误**：自动重试其他 Key
- **401 错误**：Key 失效，跳过该 Key
- **5xx 错误**：端点故障，尝试其他端点

**监控和告警：**
```go
// 建议添加使用量监控
func monitorAPIKeyUsage(key string, success bool) {
    // 记录每个 Key 的成功率和响应时间
    // 当某个 Key 失败率过高时发出告警
}
```

### 4. 安全性建议

**Key 管理：**
- 定期轮换 API Key
- 使用环境变量或配置管理服务
- 不要在代码中硬编码 Key
- 为不同环境使用不同的 Key

**权限控制：**
- 为每个 Key 设置适当的权限
- 限制 Key 的使用配额
- 监控异常使用模式

## ⚙️ 配置建议

### 开发环境
```go
llm := agent.NewOpenAI(func(oa *agent.OpenAIOptions) {
    oa.Model = "gpt-4o-mini"  // 成本低
    oa.MaxRetries = 1         // 快速失败
})
```

### 生产环境
```go
llm := agent.NewOpenAI(func(oa *agent.OpenAIOptions) {
    oa.Model = "gpt-4o"       // 高性能
    oa.APIKey = "key1,key2"   // 多Key负载均衡
    oa.MaxRetries = 3         // 高可靠性
})
```

## 🔧 故障排除

### 常见错误

| 错误类型     | 解决方法                   |
| ------------ | -------------------------- |
| API Key 错误 | 检查密钥有效性，确认未过期 |
| 网络连接     | 检查网络，考虑代理设置     |
| 模型不可用   | 确认模型名称，检查配额     |

### 调试配置
```go
// 全局调试
runtime.SetDebug(true)

// 实例调试
llm := agent.NewOpenAI(func(oa *agent.OpenAIOptions) {
    oa.OnMessage = func(chunk string, data []byte) {
        log.Printf("消息: %s", chunk)
    }
})
```

## ❓ 常见问题 FAQ

### 使用问题

**Q: 工具调用不生效，怎么办？**
A: 检查以下几点：
1. 确认使用了 `agent.WithToolCallHint(tools)` 注入工具定义
2. 确认通过 `zllm.WithToolRunner()` 注入了工具执行器
3. 检查工具 schema 是否符合 OpenAI 格式
4. 开启调试模式查看详细日志

**Q: 流式输出卡住或中断怎么办？**
A: 可能的原因和解决方案：
```go
llm := agent.NewOpenAI(func(oa *agent.OpenAIOptions) {
    oa.Stream = true
    oa.OnMessage = func(chunk string, data []byte) {
        // 检查是否有数据输出
        if len(chunk) > 0 {
            fmt.Print(chunk)
        }
    }
    // 设置较短的超时时间
    oa.Timeout = 30 * time.Second
})
```

**Q: JSON 格式化输出失败怎么办？**
A: 确保提示词明确要求 JSON 格式：
```go
prompt := message.NewPrompt("请以JSON格式回复用户问题：{{问题}}", func(p *message.PromptOptions) {
    p.Rules = []string{
        "回复必须是有效的JSON格式",
        "使用标准JSON语法，不要有注释",
    }
})
```

### 性能优化

**Q: 如何提高响应速度？**
A: 几种优化方法：
1. 使用更快的模型（如 gpt-4o-mini）
2. 启用流式输出获得即时反馈
3. 减少上下文长度
4. 使用负载均衡分配请求

**Q: 如何控制成本？**
A: 成本控制策略：
```go
// 开发环境使用低成本模型
llm := agent.NewDeepseek(func(da *agent.DeepseekOptions) {
    da.Model = "deepseek-chat"  // 比OpenAI便宜
})

// 设置合理的超时时间
ctx := zllm.WithTimeout(ctx, 30*time.Second)

// 限制工具调用迭代次数
ctx = zllm.WithMaxToolIterations(ctx, 2)
```

### 错误处理

**Q: 遇到 "max tool iterations reached" 错误怎么办？**
A: 这个错误表示工具调用次数过多。解决方案：
1. 增加最大迭代次数：`zllm.WithMaxToolIterations(ctx, 10)`
2. 检查工具逻辑是否存在死循环
3. 简化用户提示词，减少不必要的工具调用

**Q: 如何处理网络不稳定问题？**
A: 启用重试机制和负载均衡：
```go
llm := agent.NewOpenAI(func(oa *agent.OpenAIOptions) {
    oa.APIKey = "key1,key2,key3"  // 多个API Key负载均衡
    oa.MaxRetries = 3             // 启用重试
})

// 或者使用不同提供商负载均衡
balancer := zpool.NewBalancer([]agent.LLM{
    agent.NewOpenAI(...),
    agent.NewDeepseek(...),
    agent.NewGemini(...),
})
```

### 高级用法

**Q: 如何实现对话记忆？**
A: 使用 Messages 对象维护对话历史：
```go
messages := message.NewMessages()

// 第一轮对话
messages.AppendUser("你好")
resp, _ := zllm.CompleteLLM(ctx, llm, messages)
messages.AppendAssistant(resp)

// 第二轮对话，包含历史
messages.AppendUser("我刚刚说了什么？")
resp2, _ := zllm.CompleteLLM(ctx, llm, messages)
```

**Q: 如何自定义工具结果格式？**
A: 实现 ToolResultFormatter：
```go
ctx = zllm.WithToolResultFormatter(ctx, func(results []zllm.ToolResult) string {
    var builder strings.Builder
    for _, r := range results {
        if r.Err != "" {
            builder.WriteString(fmt.Sprintf("❌ %s 失败: %s\n", r.Name, r.Err))
        } else {
            builder.WriteString(fmt.Sprintf("✅ %s: %s\n", r.Name, r.Result))
        }
    }
    return builder.String()
})
```

## 🎯 实际使用案例

### 案例1：智能客服机器人

```go
package main

import (
    "context"
    "fmt"
    "log"
    
    "github.com/zlsgo/zllm"
    "github.com/zlsgo/zllm/agent"
    "github.com/zlsgo/zllm/message"
)

// 客服系统示例
func customerServiceBot() {
    // 初始化 LLM
    llm := agent.NewOpenAI(func(oa *agent.OpenAIOptions) {
        oa.Model = "gpt-4o-mini"
        oa.APIKey = "your-api-key"
        oa.MaxRetries = 3
        oa.Temperature = 0.7
    })

    // 客服知识库工具
    tools := []map[string]any{{
        "type": "function",
        "function": map[string]any{
            "name": "search_faq",
            "description": "搜索常见问题答案",
            "parameters": map[string]any{
                "type": "object",
                "properties": map[string]any{
                    "query": map[string]any{
                        "type": "string",
                        "description": "用户问题关键词",
                    },
                },
                "required": []string{"query"},
            },
        },
    }}

    // 客服工具执行器
    type customerServiceToolRunner struct{}
    func (r customerServiceToolRunner) Run(ctx context.Context, name, args string) (string, error) {
        if name == "search_faq" {
            // 这里可以连接真实的知识库或FAQ系统
            return "根据我们的知识库，这个问题最常见的解决方案是...", nil
        }
        return "", fmt.Errorf("未知工具: %s", name)
    }

    ctx := zllm.WithToolRunner(context.Background(), customerServiceToolRunner{})
    
    // 设置客服上下文
    systemPrompt := `你是一个专业的客服助手。请：
1. 友好耐心地回答用户问题
2. 优先使用工具搜索知识库
3. 如果问题复杂，建议联系人工客服
4. 保持回答简洁明了`

    messages := message.NewMessages()
    messages.Append(message.Message{Role: "system", Content: systemPrompt})
    
    // 模拟客户对话
    questions := []string{
        "我的订单什么时候能送达？",
        "如何申请退款？",
        "你们支持哪些支付方式？",
    }

    for _, question := range questions {
        messages.AppendUser(question)
        
        resp, err := zllm.CompleteLLM(ctx, llm, messages, agent.WithToolCallHint(tools))
        if err != nil {
            log.Printf("客服回复失败: %v", err)
            continue
        }
        
        fmt.Printf("客户: %s\n", question)
        fmt.Printf("客服: %s\n\n", resp)
        messages.AppendAssistant(resp)
    }
}
```

### 案例2：代码生成和重构助手

```go
package main

import (
    "context"
    "fmt"
    "strings"
    
    "github.com/sohaha/zlsgo/zjson"
    "github.com/zlsgo/zllm"
    "github.com/zlsgo/zllm/agent"
    "github.com/zlsgo/zllm/message"
)

// 代码助手工具
type codeAssistantToolRunner struct{}

func (r codeAssistantToolRunner) Run(ctx context.Context, name, args string) (string, error) {
    switch name {
    case "analyze_code":
        // 代码分析逻辑
        return "代码分析结果：发现3个潜在的性能优化点", nil
        
    case "generate_tests":
        // 生成测试代码逻辑
        return `func TestExample(t *testing.T) {
    // 自动生成的测试代码
    result := yourFunction()
    assert.Equal(t, expected, result)
}`, nil
        
    case "refactor_code":
        // 代码重构逻辑
        return "重构建议：可以使用工厂模式简化代码结构", nil
        
    default:
        return "", fmt.Errorf("未知工具: %s", name)
    }
}

func codeAssistantExample() {
    llm := agent.NewOpenAI(func(oa *agent.OpenAIOptions) {
        oa.Model = "gpt-4o" // 使用更强大的模型处理代码
        oa.APIKey = "your-api-key"
        oa.Temperature = 0.2 // 降低温度获得更准确的代码
    })

    tools := []map[string]any{
        {
            "type": "function",
            "function": map[string]any{
                "name": "analyze_code",
                "description": "分析代码质量和性能",
                "parameters": map[string]any{
                    "type": "object",
                    "properties": map[string]any{
                        "code": map[string]any{
                            "type": "string",
                            "description": "要分析的代码",
                        },
                        "language": map[string]any{
                            "type": "string",
                            "description": "编程语言",
                        },
                    },
                    "required": []string{"code", "language"},
                },
            },
        },
        {
            "type": "function",
            "function": map[string]any{
                "name": "generate_tests",
                "description": "为代码生成单元测试",
                "parameters": map[string]any{
                    "type": "object",
                    "properties": map[string]any{
                        "function": map[string]any{
                            "type": "string",
                            "description": "函数签名",
                        },
                    },
                    "required": []string{"function"},
                },
            },
        },
    }

    ctx := zllm.WithToolRunner(context.Background(), codeAssistantToolRunner{})
    
    codeSnippet := `
func calculateTotal(items []Item) float64 {
    var total float64
    for i := 0; i < len(items); i++ {
        total += items[i].Price
    }
    return total
}`
    
    messages := message.NewMessages()
    messages.AppendUser(fmt.Sprintf("请分析以下Go代码并生成测试：\n```go\n%s\n```", codeSnippet))
    
    resp, err := zllm.CompleteLLM(ctx, llm, messages, agent.WithToolCallHint(tools))
    if err != nil {
        fmt.Printf("代码助手错误: %v\n", err)
        return
    }
    
    fmt.Println("代码助手分析结果：")
    fmt.Println(resp)
}
```

### 案例3：数据分析和报告生成

```go
package main

import (
    "context"
    "fmt"
    "encoding/json"
    
    "github.com/zlsgo/zllm"
    "github.com/zlsgo/zllm/agent"
    "github.com/zlsgo/zllm/message"
)

type DataAnalysisResult struct {
    Summary     string            `json:"summary"`
    Insights    []string          `json:"insights"`
    Recommendations []string       `json:"recommendations"`
    Confidence  float64           `json:"confidence"`
}

type dataAnalysisToolRunner struct{}

func (r dataAnalysisToolRunner) Run(ctx context.Context, name, args string) (string, error) {
    switch name {
    case "analyze_sales_data":
        // 模拟销售数据分析
        result := DataAnalysisResult{
            Summary: "本月销售额相比上月增长15%",
            Insights: []string{
                "周末销售额明显高于工作日",
                "电子产品类别增长最快",
                "新客户转化率提升20%",
            },
            Recommendations: []string{
                "增加周末的营销投入",
                "优化电子产品库存管理",
                "继续推进新客户获取策略",
            },
            Confidence: 0.87,
        }
        
        data, _ := json.Marshal(result)
        return string(data), nil
        
    case "generate_report":
        return "数据分析报告已生成，包含趋势分析和预测模型", nil
        
    default:
        return "", fmt.Errorf("未知数据分析工具: %s", name)
    }
}

func dataAnalysisExample() {
    llm := agent.NewOpenAI(func(oa *agent.OpenAIOptions) {
        oa.Model = "gpt-4o"
        oa.APIKey = "your-api-key"
        oa.Temperature = 0.3
    })

    tools := []map[string]any{
        {
            "type": "function",
            "function": map[string]any{
                "name": "analyze_sales_data",
                "description": "分析销售数据并生成洞察",
                "parameters": map[string]any{
                    "type": "object",
                    "properties": map[string]any{
                        "period": map[string]any{
                            "type": "string",
                            "description": "分析周期（如：本月、本季度）",
                        },
                        "metrics": map[string]any{
                            "type": "array",
                            "items": map[string]any{"type": "string"},
                            "description": "分析指标列表",
                        },
                    },
                    "required": []string{"period"},
                },
            },
        },
    }

    ctx := zllm.WithToolRunner(context.Background(), dataAnalysisToolRunner{})
    
    messages := message.NewMessages()
    messages.AppendUser("请帮我分析本月的销售数据，重点关注销售额、客户增长率和产品类别的表现")
    
    resp, err := zllm.CompleteLLM(ctx, llm, messages, agent.WithToolCallHint(tools))
    if err != nil {
        fmt.Printf("数据分析错误: %v\n", err)
        return
    }
    
    fmt.Println("📊 数据分析报告：")
    fmt.Println(resp)
}
```

### 案例4：多轮对话与上下文管理

```go
package main

import (
    "context"
    "fmt"
    "sync"
    
    "github.com/zlsgo/zllm"
    "github.com/zlsgo/zllm/agent"
    "github.com/zlsgo/zllm/message"
)

// 对话会话管理器
type ConversationManager struct {
    sessions map[string]*message.Messages
    mutex    sync.RWMutex
    llm      agent.LLM
}

func NewConversationManager(apiKey string) *ConversationManager {
    return &ConversationManager{
        sessions: make(map[string]*message.Messages),
        llm: agent.NewOpenAI(func(oa *agent.OpenAIOptions) {
            oa.Model = "gpt-4o-mini"
            oa.APIKey = apiKey
            oa.Temperature = 0.8
        }),
    }
}

func (cm *ConversationManager) NewSession(userID string) {
    cm.mutex.Lock()
    defer cm.mutex.Unlock()
    
    messages := message.NewMessages()
    messages.Append(message.Message{
        Role: "system",
        Content: "你是一个智能助手，请记住对话历史，提供连贯的回复。",
    })
    
    cm.sessions[userID] = messages
}

func (cm *ConversationManager) Chat(userID string, userMessage string) (string, error) {
    cm.mutex.Lock()
    messages, exists := cm.sessions[userID]
    if !exists {
        messages = message.NewMessages()
        cm.sessions[userID] = messages
    }
    cm.mutex.Unlock()
    
    messages.AppendUser(userMessage)
    
    resp, err := zllm.CompleteLLM(context.Background(), cm.llm, messages)
    if err != nil {
        return "", err
    }
    
    messages.AppendAssistant(resp)
    
    return resp, nil
}

func (cm *ConversationManager) ClearSession(userID string) {
    cm.mutex.Lock()
    defer cm.mutex.Unlock()
    delete(cm.sessions, userID)
}

func conversationExample() {
    manager := NewConversationManager("your-api-key")
    
    // 模拟多用户对话
    conversations := map[string][]string{
        "user123": {
            "你好，我想了解一下Go语言",
            "Go语言有什么优势？",
            "能推荐一些学习资源吗？",
            "刚才你提到了Go的并发特性，能详细说说吗？",
        },
        "user456": {
            "帮我分析一下这段代码的性能问题",
            "代码运行时内存占用很高",
            "有没有优化建议？",
        },
    }
    
    for userID, messages := range conversations {
        fmt.Printf("=== 用户 %s 的对话 ===\n", userID)
        manager.NewSession(userID)
        
        for _, msg := range messages {
            fmt.Printf("用户: %s\n", msg)
            
            resp, err := manager.Chat(userID, msg)
            if err != nil {
                fmt.Printf("错误: %v\n", err)
                continue
            }
            
            fmt.Printf("助手: %s\n\n", resp)
        }
        
        manager.ClearSession(userID)
    }
}
```

### 案例5：多模态内容生成（使用 Gemini）

```go
package main

import (
    "context"
    "fmt"
    
    "github.com/zlsgo/zllm"
    "github.com/zlsgo/zllm/agent"
    "github.com/zlsgo/zllm/message"
)

func multimodalExample() {
    // 使用 Gemini 支持多模态处理
    gemini := agent.NewGemini(func(go *agent.GeminiOptions) {
        go.Model = "gemini-2.0-flash-exp"
        go.APIKey = "your-gemini-api-key"
        go.Temperature = 0.7
    })

    messages := message.NewMessages()
    messages.AppendUser("请描述这张图片的内容，并生成一个相关的创意标题")
    
    // 注意：实际使用中需要添加图片数据
    // messages.AppendUserWithImage("请分析这张图片", imageData)
    
    resp, err := zllm.CompleteLLM(context.Background(), gemini, messages)
    if err != nil {
        fmt.Printf("多模态处理失败: %v\n", err)
        return
    }
    
    fmt.Printf("多模态AI回复: %s\n", resp)
}
```

---

## 📝 更新日志

### v1.7.20
- ✨ 新增 Gemini 提供商支持
- 🔄 优化负载均衡器泛型参数
- 🛠️ 修复函数命名不一致问题
- 📚 完善文档和示例代码

---

## 🤝 贡献指南

欢迎提交 Issue 和 Pull Request！

## 📄 许可证

MIT License