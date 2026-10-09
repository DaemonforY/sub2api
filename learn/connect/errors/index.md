---
title: API 报错速查：401、403、404、429、503 分别是什么意思
description: 调用 HiveGPT 等 OpenAI 兼容接口时常见的报错代码速查：401 Key 无效、403 余额不足或已过期、404 地址或模型不对、429 限流或额度用完、503 上游繁忙，每种报错的原因、是否扣费和处理方法。
---

# API 报错速查

**先看状态码，再看报错里的中文说明。** HiveGPT 的报错都会写清楚原因和该怎么做，括号里是英文原文：

| 状态码 | 大致意思 | 详细排查 |
|---|---|---|
| 401 | Key 没填、填错、被删除或停用 | [401 报错：API Key 无效 / 未授权](/connect/errors/401) |
| 403 | 余额不足、订阅无效、Key 已过期，或分组不可用 | [余额不足 / 额度已用完](/connect/errors/quota)、[401 报错](/connect/errors/401#_403-key-已过期或分组不可用) |
| 404 | 接口地址不对，或模型不在你的分组里 | [Base URL 要不要加 /v1](/connect/base-url)、[模型不存在 / 分组不支持模型](/connect/errors/model) |
| 429 | 请求太快、并发太多、上游限流，或 Key 额度用完 | [429 报错：请求太频繁](/connect/errors/429)、[余额不足 / 额度已用完](/connect/errors/quota) |
| 502 / 503 | 上游模型服务繁忙或故障 | [429 报错](/connect/errors/429#_503-上游繁忙、暂时没有可用账号) |

> 更新于 2026-10。

## 按报错代码查

报错 JSON 里的 `code` 字段：

| code | 意思 | 看这里 |
|---|---|---|
| `API_KEY_REQUIRED` | 请求里没带 Key | [401](/connect/errors/401) |
| `INVALID_API_KEY` | Key 无效或已被删除 | [401](/connect/errors/401) |
| `API_KEY_DISABLED` | Key 已停用 | [401](/connect/errors/401) |
| `API_KEY_EXPIRED` | Key 过了你设置的过期时间 | [401](/connect/errors/401#_403-key-已过期或分组不可用) |
| `GROUP_DISABLED`、`GROUP_DELETED` | Key 所属的分组已停用或删除 | [401](/connect/errors/401#_403-key-已过期或分组不可用) |
| `INSUFFICIENT_BALANCE` | 账户余额不足 | [余额 / 额度](/connect/errors/quota) |
| `SUBSCRIPTION_NOT_FOUND`、`SUBSCRIPTION_INVALID` | 订阅分组的 Key，但没有有效订阅，或订阅额度用完 | [余额 / 额度](/connect/errors/quota) |
| `API_KEY_QUOTA_EXHAUSTED`、`insufficient_quota` | 你给这个 Key 设的额度上限用完了 | [余额 / 额度](/connect/errors/quota#key-额度上限用完) |
| `rate_limit_error`、`INVALID_AUTH_RATE_LIMITED` | 请求太频繁，或上游限流 | [429](/connect/errors/429) |
| `API_PATH_MISSING_V1`、`API_PATH_DUPLICATED_V1`、`API_BASE_URL`、`API_NOT_FOUND` | 接口地址填错了 | [Base URL 要不要加 /v1](/connect/base-url) |

## 失败的请求扣钱吗？

不扣。报错的请求（包括上游繁忙、限流、超时）都不计费。每次请求是否扣费、扣了多少，都能在 [使用记录](https://hivegpt.cn/usage) 里查到。

## 自己排查的顺序

1. **用 curl 测 Key。** 用 [接入教程首页](/connect/#验证-key-能用) 的命令，能返回回答说明 Key、余额和网络都正常，问题在工具的配置上。
2. **看使用记录。** 请求出现在 [使用记录](https://hivegpt.cn/usage) 里，说明已经到达 HiveGPT；没有记录，说明请求没发过来（地址或网络问题）。
3. **看报错里的中文说明。** 按它说的改，大多数问题当场就能解决。

还是解决不了，问首页右下角的智能客服，把报错原文贴给它。
