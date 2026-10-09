---
title: Java 调用 GPT API：OpenAI Java SDK 和 Spring AI 配置 base-url 完整示例
description: 用 OpenAI 官方 Java SDK（com.openai:openai-java）和 Spring AI 调用 HiveGPT 等 OpenAI 兼容接口：baseUrl 怎么填、对话、流式输出、重试超时和错误处理；Spring AI 2.x 的 base-url 要带 /v1、1.x 不要带，附 application.properties 和 ChatClient 示例。
---

# Java 调用 GPT API

**用 OpenAI 官方 Java SDK 时，`OpenAIOkHttpClient.builder().baseUrl("https://hivegpt.cn/v1")` 加上你的 HiveGPT Key 就能用。用 Spring AI 时注意版本：2.x 的 `spring.ai.openai.base-url` 填 `https://hivegpt.cn/v1`，1.x 填 `https://hivegpt.cn`（不带 /v1）。** 下面的代码都实际运行过。

> 更新于 2026-10，基于 openai-java 4.79、Spring AI 2.0.1 / 1.1.8，Java 17 及以上。模型名以你的 Key 能用的为准，见 [查看可用模型](/connect/#查看可用模型)。

还没有 Key：到 [API 密钥](https://hivegpt.cn/keys?action=create) 页创建一个，分组选 GPT 类分组。Key 放进环境变量 `HIVEGPT_API_KEY`，不要写死在代码里。

## 一、OpenAI Java SDK

Maven 依赖：

```xml
<dependency>
  <groupId>com.openai</groupId>
  <artifactId>openai-java</artifactId>
  <version>4.79.0</version>
</dependency>
```

Gradle：`implementation("com.openai:openai-java:4.79.0")`

### 基本对话

```java
package demo;

import com.openai.client.OpenAIClient;
import com.openai.client.okhttp.OpenAIOkHttpClient;
import com.openai.models.chat.completions.ChatCompletion;
import com.openai.models.chat.completions.ChatCompletionCreateParams;

public class Basic {
    public static void main(String[] args) {
        OpenAIClient client = OpenAIOkHttpClient.builder()
                .baseUrl("https://hivegpt.cn/v1")
                .apiKey(System.getenv("HIVEGPT_API_KEY"))
                .build();

        ChatCompletionCreateParams params = ChatCompletionCreateParams.builder()
                .model("gpt-5.5")
                .addSystemMessage("你是一位耐心的编程老师，回答简洁。")
                .addUserMessage("Java 的接口和抽象类有什么区别？")
                .build();

        ChatCompletion completion = client.chat().completions().create(params);
        System.out.println(completion.choices().get(0).message().content().orElse(""));
        completion.usage().ifPresent(u -> System.out.println("本次用量：" + u.totalTokens() + " tokens"));
    }
}
```

`content()` 返回的是 `Optional<String>`，用 `orElse("")` 取值。

### 流式输出

长回答一定要用流式，见 [请求超时](/connect/errors/timeout)：

```java
package demo;

import com.openai.client.OpenAIClient;
import com.openai.client.okhttp.OpenAIOkHttpClient;
import com.openai.core.http.StreamResponse;
import com.openai.models.chat.completions.ChatCompletionChunk;
import com.openai.models.chat.completions.ChatCompletionCreateParams;

public class Stream {
    public static void main(String[] args) {
        OpenAIClient client = OpenAIOkHttpClient.builder()
                .baseUrl("https://hivegpt.cn/v1")
                .apiKey(System.getenv("HIVEGPT_API_KEY"))
                .build();

        ChatCompletionCreateParams params = ChatCompletionCreateParams.builder()
                .model("gpt-5.5")
                .addUserMessage("写一首关于秋天的四行小诗")
                .build();

        try (StreamResponse<ChatCompletionChunk> stream = client.chat().completions().createStreaming(params)) {
            stream.stream()
                    .flatMap(chunk -> chunk.choices().stream())
                    .flatMap(choice -> choice.delta().content().stream())
                    .forEach(System.out::print);
        }
        System.out.println();
    }
}
```

### 重试、超时和错误处理

```java
package demo;

import com.openai.client.OpenAIClient;
import com.openai.client.okhttp.OpenAIOkHttpClient;
import com.openai.errors.OpenAIIoException;
import com.openai.errors.OpenAIServiceException;
import com.openai.models.chat.completions.ChatCompletionCreateParams;
import java.time.Duration;

public class Errors {
    public static void main(String[] args) {
        OpenAIClient client = OpenAIOkHttpClient.builder()
                .baseUrl("https://hivegpt.cn/v1")
                .apiKey(System.getenv("HIVEGPT_API_KEY"))
                .maxRetries(3)                     // 429 / 5xx 自动重试
                .timeout(Duration.ofMinutes(2))    // 单次请求超时
                .build();

        ChatCompletionCreateParams params = ChatCompletionCreateParams.builder()
                .model("gpt-5.5")
                .addUserMessage("你好")
                .build();
        try {
            System.out.println(client.chat().completions().create(params).choices().get(0).message().content().orElse(""));
        } catch (OpenAIServiceException e) {
            // 状态码 + HiveGPT 返回的报错内容（中文说明在 message 字段里）
            System.out.println("请求失败：" + e.statusCode() + " " + e.body());
        } catch (OpenAIIoException e) {
            System.out.println("连不上服务器，检查网络和代理：" + e.getMessage());
        }
    }
}
```

Key 无效时会打印：

```text
请求失败：401 {code=INVALID_API_KEY, message=API Key 无效或已被删除，请到控制台「API 密钥」复制一个有效的 Key 后重试（Invalid API key）, type=authentication_error}
```

::: tip 只改环境变量
SDK 也能读环境变量：设置 `OPENAI_BASE_URL=https://hivegpt.cn/v1` 和 `OPENAI_API_KEY`，用 `OpenAIOkHttpClient.fromEnv()` 创建客户端。
:::

## 二、Spring AI

### 先看版本：base-url 的填法相反

| Spring AI 版本 | Spring Boot | `spring.ai.openai.base-url` | 模型属性 |
|---|---|---|---|
| **2.x**（2.0.1） | 4.x | `https://hivegpt.cn/v1` | `spring.ai.openai.chat.model` |
| **1.x**（1.1.8） | 3.x | `https://hivegpt.cn` | `spring.ai.openai.chat.options.model` |

实测：1.x 会自己在后面加 `/v1/chat/completions`，填了 `/v1` 就变成 `/v1/v1/chat/completions`，HiveGPT 会返回「接口地址多了一个 /v1」；2.x 不会加 `/v1`，所以要自己写上。

### 依赖

```xml
<dependencyManagement>
  <dependencies>
    <dependency>
      <groupId>org.springframework.ai</groupId>
      <artifactId>spring-ai-bom</artifactId>
      <version>2.0.1</version>
      <type>pom</type>
      <scope>import</scope>
    </dependency>
  </dependencies>
</dependencyManagement>

<dependencies>
  <dependency>
    <groupId>org.springframework.ai</groupId>
    <artifactId>spring-ai-starter-model-openai</artifactId>
  </dependency>
</dependencies>
```

### 配置（application.properties）

::: code-group

```properties [Spring AI 2.x]
spring.ai.openai.base-url=https://hivegpt.cn/v1
spring.ai.openai.api-key=${HIVEGPT_API_KEY}
spring.ai.openai.chat.model=gpt-5.5
```

```properties [Spring AI 1.x]
spring.ai.openai.base-url=https://hivegpt.cn
spring.ai.openai.api-key=${HIVEGPT_API_KEY}
spring.ai.openai.chat.options.model=gpt-5.5
```

:::

### 用 ChatClient 调用

```java
package demo;

import org.springframework.ai.chat.client.ChatClient;
import org.springframework.boot.CommandLineRunner;
import org.springframework.boot.SpringApplication;
import org.springframework.boot.autoconfigure.SpringBootApplication;
import org.springframework.context.annotation.Bean;

@SpringBootApplication
public class App {
    public static void main(String[] args) {
        SpringApplication.run(App.class, args);
    }

    @Bean
    CommandLineRunner run(ChatClient.Builder builder) {
        return args -> {
            ChatClient chat = builder.build();
            System.out.println("ANSWER: " + chat.prompt().user("用一句话介绍 Spring AI").call().content());
        };
    }
}
```

流式输出用 `chat.prompt().user("...").stream().content()`，返回 `Flux<String>`。

## 常见问题

| 现象 | 原因 | 处理 |
|---|---|---|
| 404，「接口地址多了一个 /v1」 | Spring AI 1.x 的 base-url 带了 `/v1` | 改成 `https://hivegpt.cn` |
| 404，「接口地址少了 /v1」 | openai-java 或 Spring AI 2.x 没写 `/v1` | 改成 `https://hivegpt.cn/v1` |
| 401 | Key 不对，或 `HIVEGPT_API_KEY` 没设置 | 见 [401 报错](/connect/errors/401) |
| 404，提示分组「不支持模型」 | 模型名不在你的分组里 | 见 [模型不存在](/connect/errors/model) |
| 请求失败后很久才报错 | Spring AI 1.x 默认会重试很多次 | 设置 `spring.ai.retry.max-attempts` 调小重试次数 |
| 启动报「no ReactiveWebServerFactory bean」 | 命令行程序没有 Web 依赖 | 设置 `spring.main.web-application-type=none`，或加上 Web 依赖 |
| 400：`temperature` 不支持 | 推理类模型只接受默认值 | 去掉 temperature 配置 |

## 下一步

- 其他语言：[Python](/connect/python)、[Node.js](/connect/nodejs)、[Go](/connect/go)
- 系统学做 AI 应用：[A 路线 · AI 应用开发入门](/a/)
