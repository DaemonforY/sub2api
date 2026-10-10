---
title: AstrBot 接入 GPT API：QQ、微信、飞书、钉钉机器人（OpenAI Compatible 配置）
description: 用 Docker 部署 AstrBot，在「模型提供商」里添加 OpenAI Compatible，API Base URL 填 https://hivegpt.cn/v1，选好对话模型；再接入 QQ（官方机器人或 NapCat）、个人微信、企业微信、飞书、钉钉，设置群聊唤醒、上下文轮数和人格，以及常见报错。
---

# AstrBot 接入 GPT API

**三步接入：** 用 Docker 启动 AstrBot，打开 `http://服务器IP:6185` 登录后台；在 **模型提供商 → 对话 → 新增** 里选 **OpenAI Compatible**，API Base URL 填 `https://hivegpt.cn/v1`、填好 API Key，点「保存并获取模型」，启用 `gpt-6-luna` 等模型；在 **配置文件 → AI 配置 → 模型 → 对话模型** 里选中它并保存。最后在「机器人」里接入 QQ、微信、飞书等平台。

> 更新于 2026-10，基于 AstrBot 4.28。界面版本不同时菜单名可能略有变化。

## 准备

- 一台能长期开机的电脑或云服务器（装好 Docker）。要接入 QQ、微信等平台，服务器需要能被平台访问到（云服务器记得在安全组放行端口）。
- 一个 HiveGPT 的 API Key：在 [API 密钥](https://hivegpt.cn/keys?action=create) 页创建，分组选 GPT 类分组。
- 聊天机器人用 `gpt-6-luna` 就够：便宜、回复快；需要更聪明的回答再换 `gpt-6.1-sol`，见 [GPT 模型怎么选](/connect/models)。

## 第一步：部署 AstrBot

```bash
mkdir astrbot && cd astrbot
sudo docker run -itd \
  -p 6185:6185 -p 6199:6199 \
  -v $PWD/data:/AstrBot/data \
  -v /etc/localtime:/etc/localtime:ro \
  --name astrbot --restart always \
  soulter/astrbot:latest
```

- `6185` 是管理后台，`6199` 是给 QQ（NapCat）用的连接端口。
- 国内服务器拉镜像慢，可以把镜像换成 `m.daocloud.io/docker.io/soulter/astrbot:latest`。
- 不用 Docker 的话，可以用 Python 3.12 安装：`uv tool install astrbot --python 3.12`，然后 `astrbot init`、`astrbot run`。Windows / macOS 桌面用户可以下载官方的「AstrBot 启动器」。

## 第二步：登录后台

浏览器打开 `http://服务器IP:6185`。用户名是 `astrbot`，**初始密码是随机生成的，打印在启动日志里**：

```bash
sudo docker logs astrbot | grep -i password
```

登录后先在设置里改成你自己的密码。忘了密码可以运行 `astrbot password` 重设。

## 第三步：添加模型提供商

1. 左侧点 **模型提供商**，切到 **对话**，点 **新增**，选 **OpenAI Compatible**（旧版本叫「接入 OpenAI」）。
2. 填写：

| 字段 | 填什么 |
|---|---|
| 提供商名称 | `HiveGPT`（随意） |
| API Key | 你的 HiveGPT Key（`sk-` 开头） |
| API Base URL | `https://hivegpt.cn/v1`（带 `/v1`） |

3. 点 **保存并获取模型**，在列表里给 `gpt-6-luna`、`gpt-6.1-sol` 点 **+** 启用。列表拉不下来时，用「自定义模型」手动填模型名。
4. 点模型旁边的 **测试模型**，能收到回复就说明通了。

## 第四步：选为对话模型

进入 **配置文件**，选中正在用的配置 → **AI 配置 → 模型 → 对话模型**，选刚才启用的 `gpt-6-luna`，点右下角 **保存配置**。

不做这一步，机器人可能还在用别的提供商，或者根本不回复。

## 第五步：接入聊天平台

在 **机器人 → 创建机器人** 里选平台。常用的几种：

| 平台 | 方式 | 说明 |
|---|---|---|
| QQ | QQ 官方机器人 | 在 QQ 开放平台申请，最稳定，推荐 |
| QQ | OneBot v11 + NapCat | 用个人 QQ 号登录，有被风控封号的风险，建议用小号 |
| 个人微信 | 微信官方接口（扫码登录） | 需要较新版本的微信，按后台提示扫码 |
| 企业微信 | 企微应用 / 企微智能机器人 | 适合公司内部用 |
| 飞书 | 企业自建应用 + 机器人能力，事件订阅选「长连接」 | 不需要公网地址 |
| 钉钉 | 应用 + 机器人能力，事件订阅选 Stream 模式 | 不需要公网地址 |

以 QQ（NapCat）为例：AstrBot 里创建 **OneBot v11** 机器人，反向 WebSocket 主机填 `0.0.0.0`、端口 `6199`；NapCat 后台 **网络配置 → 新建 → WebSockets 客户端**，地址填 `ws://AstrBot所在IP:6199/ws`。日志里出现「适配器已连接」就成功了。

飞书、钉钉的详细步骤（创建应用、开权限、发布版本）按 AstrBot 后台里的指引做，记得把机器人拉进群。

## 群聊设置和省 token

- **群里怎么叫它**：默认要 @机器人，或用唤醒词 `/` 开头。可以设置额外的唤醒前缀，避免群里聊天都触发。
- **会话隔离**：打开后群里每个人有自己的上下文，互不干扰。
- **限制对话轮数**：上下文默认不限长度，聊得越久每次请求越贵。建议把最大对话轮数设成 10–20，或者在群里发 `/reset` 清空。
- **人格**：在 **人格设定** 里写系统提示词，比如「你是 XX 社群的助手，回答简短，不超过 100 字」。限制回答长度也能省钱。

## 画图

AstrBot 本身不带画图，需要在 **插件** 市场装一个支持 OpenAI 图片接口的生图插件，接口地址填 `https://hivegpt.cn/v1`、模型填 `gpt-image-2`。插件都是社区开发的，安装前看一下说明和更新时间。画图按张计费，价格见 [模型广场](https://hivegpt.cn/model-plaza)，群里开放画图前注意设置使用限制。

## 常见问题

| 现象 | 原因 | 处理 |
|---|---|---|
| 测试模型失败 / 404 | API Base URL 写错 | 写成 `https://hivegpt.cn/v1`，不要只写域名，也不要写到 `/chat/completions` |
| 401 | Key 填错或已停用 | 重新复制 Key，见 [401 报错](/connect/errors/401) |
| 模型不存在 / 分组不支持模型 | 模型名不对，或没启用 | 用「自定义模型」填准确的模型名并点 + 启用，见 [模型不存在](/connect/errors/model) |
| 机器人在线但不回复 | 没选对话模型，或群里没 @ 它 | 按第四步选择对话模型并保存；群里 @ 机器人或用唤醒词 |
| 回复超时 | 模型慢或网络慢 | 换 `gpt-6-luna`；在提供商设置里调大超时时间（默认 120 秒） |
| 余额掉得很快 | 上下文太长、群太活跃 | 限制对话轮数、加唤醒前缀、换便宜的模型 |
| 后台打不开 | 端口没放行 | 云服务器安全组放行 6185；不要在容器里用 localhost 访问 |
| 知识库功能用不了 | 需要嵌入模型 | HiveGPT 不提供嵌入模型，知识库需要另配嵌入服务商 |

**会被封号吗？** 用个人 QQ、个人微信号登录的方式存在被平台风控的风险。正式运营建议用 QQ 官方机器人、企业微信、飞书、钉钉这类官方接口。

更多：[飞书 / 企业微信群机器人（写代码版）](/connect/feishu-wecom-bot) · [Dify 接入](/connect/dify) · [n8n 接入](/connect/n8n)
