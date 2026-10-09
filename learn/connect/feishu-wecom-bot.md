---
title: 飞书、企业微信群机器人接入 GPT：Webhook 定时推送和签名校验教程
description: 用 Python 让 GPT 生成内容（早会提醒、日报、新闻摘要），通过飞书自定义机器人和企业微信群机器人的 Webhook 发到群里：添加机器人、签名校验算法、消息格式、频率限制和报错码；以及要做「@机器人就回复」需要飞书开放平台应用。
---

# 飞书、企业微信群机器人接入 GPT

**群机器人的 Webhook 只能往群里发消息：先用 HiveGPT 的接口让模型写好内容，再 POST 到机器人的 Webhook 地址。** 飞书在群设置里添加「自定义机器人」，企业微信在群里「添加群机器人」，拿到 Webhook 地址后运行下面的脚本即可；配合定时任务，就是每天自动推送的日报 / 早会提醒。

> 更新于 2026-10。脚本的消息格式和飞书签名已在本地核对过。

## 准备

- 一个 HiveGPT 的 API Key：在 [API 密钥](https://hivegpt.cn/keys?action=create) 页创建。
- `pip install -U openai requests`

## 第一步：添加群机器人

**飞书**

1. 打开群聊，点右上角 **更多 → 设置 → 群机器人 → 添加机器人**，选 **自定义机器人**。
2. 填好名称和描述，点 **添加**，复制 **Webhook 地址**（`https://open.feishu.cn/open-apis/bot/v2/hook/...`）。
3. 安全设置里建议打开 **签名校验**，复制密钥。

**企业微信**

1. 在内部群聊里点右上角 **···**，选 **添加群机器人**，新建一个机器人。
2. 复制 **Webhook 地址**（`https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=...`）。

::: warning Webhook 地址等于发言权
拿到 Webhook 地址的人都能往群里发消息，不要发到公开的地方或提交到 Git。
:::

## 第二步：运行脚本

```python
"""让模型写一段内容，发到飞书群或企业微信群（群机器人 Webhook）。

用法：python daily_digest.py
可以配合 cron / 任务计划每天定时运行。
"""
import base64
import hashlib
import hmac
import os
import time

import requests
from openai import OpenAI

client = OpenAI(base_url="https://hivegpt.cn/v1", api_key=os.environ["HIVEGPT_API_KEY"])

FEISHU_WEBHOOK = os.environ.get("FEISHU_WEBHOOK")   # https://open.feishu.cn/open-apis/bot/v2/hook/xxxx
FEISHU_SECRET = os.environ.get("FEISHU_SECRET")     # 开启了「签名校验」才需要
WECOM_WEBHOOK = os.environ.get("WECOM_WEBHOOK")     # https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=xxxx


def ask(prompt: str) -> str:
    resp = client.chat.completions.create(model="gpt-5.5", messages=[{"role": "user", "content": prompt}])
    return resp.choices[0].message.content.strip()


def feishu_sign(timestamp: str, secret: str) -> str:
    # 飞书签名：以「timestamp + 换行 + 密钥」为 key，对空字符串做 HmacSHA256，再 Base64
    string_to_sign = f"{timestamp}\n{secret}"
    digest = hmac.new(string_to_sign.encode("utf-8"), digestmod=hashlib.sha256).digest()
    return base64.b64encode(digest).decode("utf-8")


def send_feishu(text: str):
    body = {"msg_type": "text", "content": {"text": text}}
    if FEISHU_SECRET:
        timestamp = str(int(time.time()))
        body["timestamp"] = timestamp
        body["sign"] = feishu_sign(timestamp, FEISHU_SECRET)
    result = requests.post(FEISHU_WEBHOOK, json=body, timeout=10).json()
    if result.get("code") != 0:
        raise RuntimeError(f"飞书发送失败：{result}")


def send_wecom(markdown: str):
    body = {"msgtype": "markdown", "markdown": {"content": markdown}}
    result = requests.post(WECOM_WEBHOOK, json=body, timeout=10).json()
    if result.get("errcode") != 0:
        raise RuntimeError(f"企业微信发送失败：{result}")


if __name__ == "__main__":
    content = ask("给一个 10 人的产品团队写今天的早会提醒：一句鼓励的话，加 3 条今天值得关注的工作习惯，每条不超过 20 字。")
    if FEISHU_WEBHOOK:
        send_feishu(content)
    if WECOM_WEBHOOK:
        send_wecom("**今日早会提醒**\n" + content)
    print("已发送：\n" + content)
```

设置环境变量后运行（只配哪个就发到哪个）：

```bash
export HIVEGPT_API_KEY="sk-你的Key"
export FEISHU_WEBHOOK="https://open.feishu.cn/open-apis/bot/v2/hook/xxxx"
export FEISHU_SECRET="签名校验的密钥"
export WECOM_WEBHOOK="https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=xxxx"
python daily_digest.py
```

**每天定时发送**：macOS / Linux 用 `crontab -e` 加一行 `0 9 * * 1-5 cd /路径 && python daily_digest.py`（工作日 9 点）；Windows 用「任务计划程序」。也可以用 [n8n](/connect/n8n) 搭成工作流。

## 消息格式和限制

| | 飞书自定义机器人 | 企业微信群机器人 |
|---|---|---|
| 纯文本 | `{"msg_type":"text","content":{"text":"..."}}` | `{"msgtype":"text","text":{"content":"..."}}` |
| 富文本 | `msg_type` 为 `post`（富文本）或 `interactive`（卡片，支持 markdown） | `msgtype` 为 `markdown` 或 `markdown_v2` |
| 长度 | 请求体不超过 20 KB | 文本 2048 字节，markdown 4096 字节 |
| 频率 | 每分钟 100 次、每秒 5 次 | 每分钟 20 条 |
| @所有人 | 文本里写 `<at user_id="all">所有人</at>` | `text` 里加 `"mentioned_list":["@all"]` |
| 成功返回 | `{"code":0,"msg":"success"}` | `{"errcode":0,"errmsg":"ok"}` |

模型写的内容可能超过长度限制，在提示词里限制字数，或者发送前截断。

## 常见报错

| 报错 | 原因 | 处理 |
|---|---|---|
| 飞书 `19001` incoming webhook access token invalid | Webhook 地址不对或机器人已删除 | 重新复制地址 |
| 飞书 `19021` sign match fail | 签名不对，或服务器时间偏差超过 1 小时 | 检查密钥；同步服务器时间 |
| 飞书 `19024` Key Words Not Found | 开了「自定义关键词」，消息里没有关键词 | 在内容里加上关键词，或关掉这个设置 |
| 飞书 `11232`、企业微信 `45009` | 发得太频繁 | 降低频率 |
| 企业微信 `93000` invalid webhook url | key 不对或机器人已被移出群 | 重新复制地址 |
| 脚本报 HiveGPT 的错误 | 调用模型失败 | 见 [报错速查](/connect/errors/) |

## 想要「@机器人就回复」

群机器人只能发、不能收消息。要做能回答问题的机器人，需要在 [飞书开放平台](https://open.feishu.cn) 创建 **企业自建应用**：

1. 创建应用，添加 **机器人** 能力。
2. 开通权限：接收群聊中 @机器人的消息（`im:message.group_at_msg:readonly`）、以应用身份发消息（`im:message:send_as_bot`）。
3. 在 **事件与回调** 里选 **使用长连接接收事件**（不需要公网地址），订阅 **接收消息**（`im.message.receive_v1`）。
4. 收到消息后调用 HiveGPT 生成回答，再用「回复消息」接口（`POST /open-apis/im/v1/messages/:message_id/reply`）回复。
5. 创建版本并发布，把机器人拉进群。

事件要在 3 秒内响应，模型回答通常更慢，所以收到事件后先返回、在后台生成回答再回复。完整的开发步骤见飞书开放平台的官方文档。
