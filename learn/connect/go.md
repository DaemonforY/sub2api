---
title: Go 调用 GPT API 完整示例：openai-go 配置 Base URL、流式输出、多轮对话、错误处理
description: 用 OpenAI 官方 Go SDK（github.com/openai/openai-go/v3）调用 HiveGPT 等 OpenAI 兼容接口：option.WithBaseURL 填 https://hivegpt.cn/v1，基本对话、流式输出、多轮对话、重试超时，以及怎么从报错里取出中文说明。
---

# Go 调用 GPT API 完整示例

**用官方的 `github.com/openai/openai-go/v3`，创建客户端时加上 `option.WithBaseURL("https://hivegpt.cn/v1")` 和你的 HiveGPT Key，其余代码和调用 OpenAI 官方接口一样。** 下面每段代码都是完整的 `main.go`，可以直接 `go run`。

> 更新于 2026-10，基于 openai-go v3.74，Go 1.22 及以上。模型名以你的 Key 能用的为准，见 [查看可用模型](/connect/#查看可用模型)。

## 准备

```bash
go mod init demo
go get github.com/openai/openai-go/v3
```

注意导入路径带 `/v3`。不带版本号的 `github.com/openai/openai-go` 是旧的 v1，API 不一样。

Key 放进环境变量，不要写死在代码里：

::: code-group

```bash [macOS / Linux]
export HIVEGPT_API_KEY="sk-你的Key"
```

```powershell [Windows PowerShell]
$env:HIVEGPT_API_KEY="sk-你的Key"
```

:::

还没有 Key：到 [API 密钥](https://hivegpt.cn/keys?action=create) 页创建一个，分组选 GPT 类分组。

## 基本对话

```go
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
)

func main() {
	client := openai.NewClient(
		option.WithBaseURL("https://hivegpt.cn/v1"),
		option.WithAPIKey(os.Getenv("HIVEGPT_API_KEY")),
	)

	resp, err := client.Chat.Completions.New(context.Background(), openai.ChatCompletionNewParams{
		Model: "gpt-5.5",
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.SystemMessage("你是一位耐心的编程老师，回答简洁。"),
			openai.UserMessage("Go 的 goroutine 和线程有什么区别？"),
		},
	})
	if err != nil {
		panic(err)
	}
	fmt.Println(resp.Choices[0].Message.Content)
	fmt.Println("本次用量：", resp.Usage.TotalTokens, "tokens")
}
```

## 流式输出

回答边生成边打印。长回答一定要用流式，见 [请求超时](/connect/errors/timeout)：

```go
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
)

func main() {
	client := openai.NewClient(
		option.WithBaseURL("https://hivegpt.cn/v1"),
		option.WithAPIKey(os.Getenv("HIVEGPT_API_KEY")),
	)

	stream := client.Chat.Completions.NewStreaming(context.Background(), openai.ChatCompletionNewParams{
		Model:    "gpt-5.5",
		Messages: []openai.ChatCompletionMessageParamUnion{openai.UserMessage("写一首关于秋天的四行小诗")},
	})
	for stream.Next() {
		chunk := stream.Current()
		if len(chunk.Choices) > 0 {
			fmt.Print(chunk.Choices[0].Delta.Content)
		}
	}
	if err := stream.Err(); err != nil {
		panic(err)
	}
	fmt.Println()
}
```

## 多轮对话

模型不记得上一轮说了什么，每次请求都要带上之前的对话。`ToParam()` 把模型的回复转成可以放回消息列表的格式：

```go
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
)

func main() {
	client := openai.NewClient(
		option.WithBaseURL("https://hivegpt.cn/v1"),
		option.WithAPIKey(os.Getenv("HIVEGPT_API_KEY")),
	)
	ctx := context.Background()

	messages := []openai.ChatCompletionMessageParamUnion{openai.SystemMessage("你是一位旅行规划助手。")}
	for _, question := range []string{"我五月想去云南玩 5 天", "预算 5000 元够吗？", "那第一天怎么安排？"} {
		messages = append(messages, openai.UserMessage(question))
		resp, err := client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{Model: "gpt-5.5", Messages: messages})
		if err != nil {
			panic(err)
		}
		answer := resp.Choices[0].Message
		messages = append(messages, answer.ToParam())
		fmt.Printf("问：%s\n答：%s\n\n", question, answer.Content)
	}
}
```

## 重试、超时和错误处理

```go
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
)

// reason 取出报错说明：HiveGPT 返回 {"code","message"}，OpenAI 官方返回 {"error":{"message"}}，两种都支持。
func reason(e *openai.Error) string {
	if e.Message != "" {
		return e.Message
	}
	dump := e.DumpResponse(true)
	if i := bytes.Index(dump, []byte("\r\n\r\n")); i >= 0 {
		var body struct {
			Message string `json:"message"`
			Error   struct {
				Message string `json:"message"`
			} `json:"error"`
		}
		if json.Unmarshal(dump[i+4:], &body) == nil {
			if body.Message != "" {
				return body.Message
			}
			return body.Error.Message
		}
	}
	return e.Error()
}

func main() {
	client := openai.NewClient(
		option.WithBaseURL("https://hivegpt.cn/v1"),
		option.WithAPIKey(os.Getenv("HIVEGPT_API_KEY")),
		option.WithMaxRetries(3),                 // 429 / 5xx 自动重试
		option.WithRequestTimeout(2*time.Minute), // 单次请求超时
	)

	resp, err := client.Chat.Completions.New(context.Background(), openai.ChatCompletionNewParams{
		Model:    "gpt-5.5",
		Messages: []openai.ChatCompletionMessageParamUnion{openai.UserMessage("你好")},
	})
	var apiErr *openai.Error
	switch {
	case errors.As(err, &apiErr):
		fmt.Println("请求失败：", apiErr.StatusCode, reason(apiErr))
	case err != nil:
		fmt.Println("连不上服务器，检查网络和代理：", err)
	default:
		fmt.Println(resp.Choices[0].Message.Content)
	}
}
```

::: warning 报错说明要自己取出来
openai-go 只认 OpenAI 官方的报错格式 `{"error":{"message":...}}`。HiveGPT 网关自己的报错（如 Key 无效、余额不足）是 `{"code":...,"message":...}`，这时 `apiErr.Message` 是空的，`err.Error()` 也只有「401 Unauthorized」。上面的 `reason` 函数两种格式都能取出中文说明，例如：

```text
请求失败： 401 API Key 无效或已被删除，请到控制台「API 密钥」复制一个有效的 Key 后重试（Invalid API key）
```
:::

按状态码处理：401 是 Key 的问题，404 是地址或模型名的问题，429 是请求太快或额度用完。见 [报错速查](/connect/errors/)。

## 常见问题

| 现象 | 原因 | 处理 |
|---|---|---|
| 编译报 `undefined: openai.ChatCompletionNewParams` 等 | 导入了旧版 `github.com/openai/openai-go`（v1） | 改成 `github.com/openai/openai-go/v3` |
| 404，「接口地址少了 /v1」或「多了一个 /v1」 | `WithBaseURL` 写错 | 写成 `https://hivegpt.cn/v1`，见 [Base URL 要不要加 /v1](/connect/base-url) |
| 401，`reason` 显示 Key 无效 | Key 不对，或环境变量没读到 | 见 [401 报错](/connect/errors/401) |
| 404，提示分组「不支持模型」 | 模型名不在你的分组里 | 见 [模型不存在](/connect/errors/model) |
| `context deadline exceeded` | 请求超时 | 调大 `WithRequestTimeout`，长回答用流式 |
| 连接被拒绝、`proxyconnect` 报错 | 设置了 `HTTPS_PROXY` 但代理没开 | 关掉代理环境变量，HiveGPT 在国内可直接访问 |

## 下一步

- 其他语言：[Python](/connect/python)、[Node.js](/connect/nodejs)、[Java](/connect/java)
- 系统学做 AI 应用：[A 路线 · AI 应用开发入门](/a/)
