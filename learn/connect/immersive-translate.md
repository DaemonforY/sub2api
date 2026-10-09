---
title: 沉浸式翻译接入 GPT API：自定义 API 接口地址怎么填
description: 在沉浸式翻译（Immersive Translate）浏览器插件里接入 HiveGPT 的 OpenAI 兼容接口，用 GPT 翻译网页、PDF 和视频字幕：自定义 API 接口地址要填完整的 /v1/chat/completions，附测试失败、翻译慢、费用控制的处理方法。
---

# 沉浸式翻译接入 GPT API

**关键一点：沉浸式翻译的「自定义 API 接口地址」要填完整地址** `https://hivegpt.cn/v1/chat/completions`，不能只填 `https://hivegpt.cn/v1`。APIKEY 填你的 HiveGPT Key，模型选「自定义模型」并填 `gpt-5.5` 等你分组里的模型名，点「测试服务」通过后，把它设为默认翻译服务即可。

> 更新于 2026-10，基于沉浸式翻译浏览器插件当前版本。插件改版时菜单位置可能略有不同。

## 准备

- 浏览器里装好 [沉浸式翻译](https://immersivetranslate.com) 插件（Chrome、Edge、Firefox、Safari 都有）。
- 一个 HiveGPT 的 API Key：在 [API 密钥](https://hivegpt.cn/keys?action=create) 页创建，分组选 GPT 类分组。还没有账号先 [注册](https://hivegpt.cn/register?utm_source=learn&utm_medium=immersive-translate)。

## 第一步：添加 OpenAI 兼容服务

1. 点浏览器工具栏里的沉浸式翻译图标，打开 **设置**。
2. 在左侧选 **翻译服务**，点 **添加 OpenAI 兼容服务**（也可以直接编辑内置的 **OpenAI** 服务）。

## 第二步：填写接口信息

| 字段 | 填什么 |
|---|---|
| 自定义翻译服务名称 | 随便写，例如 `HiveGPT` |
| 自定义 API 接口地址 | `https://hivegpt.cn/v1/chat/completions` |
| APIKEY | 你的 HiveGPT Key（`sk-` 开头） |
| 模型 | 选「自定义模型」，填 `gpt-5.5` 等你分组里的模型名 |

::: warning 地址要写到 /chat/completions
大多数工具只填 `https://hivegpt.cn/v1`，会自己补后面的路径，沉浸式翻译不会。只填到 `/v1` 时，请求会发到网站首页，测试时会报错或提示返回内容不是 JSON。
:::

模型名以 [`/v1/models`](/connect/#查看可用模型) 的返回为准。翻译不需要最强的模型，分组里有更便宜、更快的模型时优先选它。

## 第三步：测试并设为默认

1. 点 **测试服务**，显示成功就配置好了。
2. 回到 **翻译服务** 列表，把刚添加的 `HiveGPT` 设为默认翻译服务；也可以只在「鼠标悬停翻译」「输入框翻译」等单项里选它。
3. 打开任意英文网页，点插件的「翻译」按钮，看是否出现双语对照。

测试成功后，每次翻译的用量和费用都能在 HiveGPT 的 [使用记录](https://hivegpt.cn/usage) 里查到。

## 测试失败怎么办

| 现象 | 原因 | 处理 |
|---|---|---|
| 提示返回的不是 JSON，或报错里有一段 HTML | 地址只填到了 `/v1` 或 `https://hivegpt.cn` | 改成完整的 `https://hivegpt.cn/v1/chat/completions` |
| 404 | 地址拼写错了，例如多了一个 `/v1` | 对照上面的表格重新填 |
| 401、提示 API Key 无效 | Key 复制不全，或已删除 / 停用 | 到 [API 密钥](https://hivegpt.cn/keys) 页重新复制完整的 Key |
| 提示模型不存在、分组不支持 | 模型名不在你 Key 的分组里 | 用 `/v1/models` 查可用模型，填准确的模型名 |
| 429、提示请求太频繁 | 整页翻译时并发请求太多 | 在服务的高级设置里调低「每秒最大请求数」，过 1–2 分钟再试 |
| 提示余额不足 | 余额为 0，或订阅额度用完 | [充值或订阅](https://hivegpt.cn/purchase) 后重试 |

更多地址相关的报错，看 [Base URL 末尾要不要加 /v1](/connect/base-url)。

## 省钱和提速

- **整页翻译很耗 token。** 一篇长文翻译一次可能有几万 token。日常浏览可以只开「鼠标悬停翻译」，需要时再整页翻译。
- **选便宜的模型。** 翻译对推理能力要求不高，用分组里较快、较便宜的模型就够了。
- **给 Key 设额度上限。** 在 [API 密钥](https://hivegpt.cn/keys) 页给这个 Key 单独设一个额度上限，避免一次翻译太多超出预算。
- **翻译慢时调高并发。** 如果没有遇到 429，可以适当调高「每秒最大请求数」，整页翻译会快一些。

## 常见问题

**需要开通沉浸式翻译的会员吗？** 用自己的 API Key 接入 OpenAI 兼容服务，本身不需要它的会员；PDF、字幕等部分功能是否要会员，以插件里的提示为准。

**PDF、视频字幕也能用吗？** 能用。沉浸式翻译的 PDF 翻译、视频双语字幕会使用你设为默认的翻译服务。PDF 一般比网页更长，注意上面的费用提示。

**和网页版 ChatGPT 会员是一回事吗？** 不是。插件调用的是 API 接口，按用量计费，和 ChatGPT Plus 会员无关。
