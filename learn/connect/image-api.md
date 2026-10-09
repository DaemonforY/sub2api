---
title: gpt-image-2 API 调用教程：Python / Node.js / curl 生成图片和改图
description: 用 OpenAI 官方 SDK 调用 HiveGPT 的图片接口：/v1/images/generations 文生图、/v1/images/edits 上传图片局部修改、流式返回防超时，保存 b64_json 为文件，按张计费怎么算，以及 403 / 504 / 413 等常见报错。
---

# gpt-image-2 API 调用教程

**两行改动就能用：** 用 OpenAI 官方 SDK，把 `base_url` 换成 `https://hivegpt.cn/v1`、API Key 换成你的 HiveGPT Key，调用 `client.images.generate(model="gpt-image-2", prompt=...)` 生成图片，`client.images.edit(...)` 修改图片。返回的是 base64 编码的图片（`b64_json`），解码后保存成文件。

> 更新于 2026-10，代码在 openai Python 3.26 上实际跑过。不想写代码，直接用 [无限画布](https://canvas.hivegpt.cn/?utm_source=learn&utm_medium=image-api) 画图；提示词怎么写见 [生图提示词](/prompts/)。

## 准备

- 一个 HiveGPT 的 API Key，分组要支持画图（GPT 类分组）。在 [API 密钥](https://hivegpt.cn/keys?action=create) 页创建。
- 用 `/v1/models` 确认分组里有 `gpt-image-2`，见 [查看可用模型](/connect/#查看可用模型)。
- Python：`pip install -U openai`；Node.js：`npm i openai`。

Key 不要写进代码，放在环境变量里：

```bash
export HIVEGPT_API_KEY="sk-你的Key"
```

## 文生图

```python
import base64, os
from openai import OpenAI

client = OpenAI(
    base_url="https://hivegpt.cn/v1",
    api_key=os.environ["HIVEGPT_API_KEY"],
    timeout=600,  # 画图要几十秒，默认超时可能不够
)

img = client.images.generate(
    model="gpt-image-2",
    prompt="一只在图书馆看书的橘猫，水彩风格",
    size="1024x1024",
)
with open("cat.png", "wb") as f:
    f.write(base64.b64decode(img.data[0].b64_json))
print(img.size, img.usage)
```

![AI 生成：一只在图书馆看书的橘猫，水彩风格](/learn-img/api/cat.webp)

*上面这段代码生成的图（AI 生成），用了 25 秒。*

常用参数：

| 参数 | 说明 |
|---|---|
| `model` | `gpt-image-2`。不填也默认用它 |
| `prompt` | 画面描述，越具体越好。中文可以 |
| `size` | `1024x1024`、`1024x1536`（竖）、`1536x1024`（横），或 `auto`。实际尺寸以返回的 `img.size` 为准，可能和你填的略有不同 |
| `quality` | `low` / `medium` / `high` / `auto` |
| `n` | 一次生成几张，按实际返回的张数计费 |
| `output_format` | `png`（默认）、`jpeg`、`webp` |
| `background` | `transparent` 要透明背景（配合 png / webp） |

## 改图（上传图片）

把图片和修改要求一起发给 `/v1/images/edits`。下面给上一步那只猫戴一顶帽子：

```python
ed = client.images.edit(
    model="gpt-image-2",
    image=open("cat.png", "rb"),
    prompt="给这只猫戴上一顶红色的小圆帽，其他保持不变",
)
with open("cat-hat.png", "wb") as f:
    f.write(base64.b64decode(ed.data[0].b64_json))
```

![AI 生成：给猫戴上红色小圆帽，其他不变](/learn-img/api/cat-hat.webp)

*改图结果（AI 生成）：只加了帽子，背景、书和文字都没变。*

- 多张参考图：`image=[open("a.png", "rb"), open("b.png", "rb")]`，在提示词里说清楚「第一张图的人物，穿上第二张图的衣服」。
- 提示词里写「其他保持不变」，模型会尽量只改你说的部分。
- 单张图片不要超过 20MB，整个请求太大时会报 413，先压缩再传。

## 流式返回（防超时）

高质量、大尺寸的图可能要一两分钟。普通请求要等图全部画完才返回，网络中间的代理可能先把连接断了。加 `stream=True`，连接在生成过程中一直有数据，不会被断开：

```python
stream = client.images.generate(
    model="gpt-image-2",
    prompt="极简风格的咖啡杯图标，白色背景",
    size="1024x1024",
    stream=True,
)
for ev in stream:
    if ev.type == "image_generation.completed":
        with open("cup.png", "wb") as f:
            f.write(base64.b64decode(ev.b64_json))
```

流里还会有 `keepalive` 等其他事件，只处理 `image_generation.completed`（改图是 `image_edit.completed`）即可。

## Node.js

```js
import fs from 'node:fs'
import OpenAI from 'openai'

const client = new OpenAI({
  baseURL: 'https://hivegpt.cn/v1',
  apiKey: process.env.HIVEGPT_API_KEY,
  timeout: 600_000,
})

const img = await client.images.generate({
  model: 'gpt-image-2',
  prompt: '一只在图书馆看书的橘猫，水彩风格',
  size: '1024x1024',
})
fs.writeFileSync('cat.png', Buffer.from(img.data[0].b64_json, 'base64'))

// 改图
const ed = await client.images.edit({
  model: 'gpt-image-2',
  image: fs.createReadStream('cat.png'),
  prompt: '给这只猫戴上一顶红色的小圆帽，其他保持不变',
})
fs.writeFileSync('cat-hat.png', Buffer.from(ed.data[0].b64_json, 'base64'))
```

## curl

```bash
# 文生图：返回 JSON，图片在 data[0].b64_json
curl https://hivegpt.cn/v1/images/generations \
  -H "Authorization: Bearer $HIVEGPT_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"model":"gpt-image-2","prompt":"一只在图书馆看书的橘猫，水彩风格","size":"1024x1024"}' \
  | python3 -c 'import sys,json,base64;open("cat.png","wb").write(base64.b64decode(json.load(sys.stdin)["data"][0]["b64_json"]))'

# 改图：multipart 上传
curl https://hivegpt.cn/v1/images/edits \
  -H "Authorization: Bearer $HIVEGPT_API_KEY" \
  -F model=gpt-image-2 \
  -F image=@cat.png \
  -F prompt="给这只猫戴上一顶红色的小圆帽，其他保持不变"
```

## 怎么计费

- **按张计费**，和提示词长短、返回的 token 数无关。
- 按实际输出的尺寸分档，常规尺寸一张约 $0.2，更大的尺寸更贵。每次花了多少，在 [使用记录](https://hivegpt.cn/usage) 里能看到。
- **失败不扣费**：报错信息里写着「本次请求未扣费」的，就没有扣钱。
- **客户端超时不等于没画**：你这边超时断开后，服务端可能还在生成，画完照样计费。所以超时要设长一点（上面写的 600 秒），不要一超时就马上重发。

批量出图前，先用一两张确认效果和花费。

## 常见问题

| 现象 | 原因 | 处理 |
|---|---|---|
| 403：`Image generation is not enabled for this group` | Key 所在分组没开画图 | 换 GPT 类、支持画图的分组的 Key |
| 400：`images endpoint requires an image model` | `model` 填了对话模型 | 图片接口只能用 `gpt-image-2` 等图片模型 |
| 504：上游生成超时（超过 2 分钟没有返回） | 上游繁忙，或尺寸、质量太高 | 稍后重试、调低 `quality`，或改用 `stream=True` |
| 413：请求内容太大 | 参考图太大或太多 | 压缩图片、减少参考图数量 |
| 429 | 同时画的图太多 | 控制并发，见 [429 报错](/connect/errors/429) |
| 400，提到内容政策（moderation / safety） | 提示词或参考图触发了内容审核 | 修改描述，不要用真人照片、品牌商标等 |
| 客户端报 `APITimeoutError` | SDK 默认超时太短 | 设 `timeout=600`，或用流式 |
| 返回的图尺寸和 `size` 不一样 | 模型按画面调整了尺寸 | 以返回的 `img.size` 为准，需要固定尺寸就后期裁切 |

**能返回图片链接（URL）而不是 base64 吗？** 目前返回的是 base64 数据，自己解码保存成文件，或上传到你的存储后再给用户链接。

更多：[生图提示词](/prompts/) · [中文字不乱码的写法](/prompts/chinese-text) · [Python 完整示例](/connect/python) · [报错速查](/connect/errors/)
