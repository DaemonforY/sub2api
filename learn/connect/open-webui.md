---
title: Open WebUI 接入 OpenAI 兼容 API：Docker 部署 + 添加连接（不用 Ollama）
description: 用 Docker 或 pip 部署 Open WebUI，在 设置 → 管理员 → 连接 里添加 OpenAI API 连接（URL 填 https://hivegpt.cn/v1），只显示指定模型、关闭 Ollama、设置便宜的任务模型省 token、配置 gpt-image-2 画图，以及模型不显示、连接失败等常见问题。
---

# Open WebUI 接入 OpenAI 兼容 API

**三步接入：** 用 Docker 启动 Open WebUI，第一个注册的账号就是管理员；进入 **设置 → 管理员 → 连接（Settings > Admin > Connections）**，在 OpenAI API 下点 ➕，URL 填 `https://hivegpt.cn/v1`，API Key 填你的 HiveGPT Key，点验证后保存；回到聊天页，在左上角模型列表里选 `gpt-6.1-sol` 等模型即可。

> 更新于 2026-10，基于 Open WebUI 0.11。界面可切换中文，下文同时给出英文菜单名，版本不同时位置可能略有变化。

## 准备

- 一台装了 Docker 的电脑或服务器（本机自己用也可以）。
- 一个 HiveGPT 的 API Key：在 [API 密钥](https://hivegpt.cn/keys?action=create) 页创建，分组选 GPT 类分组。

## 第一步：部署

最简单的方式，一条命令（可以顺便通过环境变量填好连接）：

```bash
docker run -d -p 3000:8080 \
  -v open-webui:/app/backend/data \
  -e WEBUI_SECRET_KEY="$(openssl rand -hex 32)" \
  -e OPENAI_API_BASE_URL=https://hivegpt.cn/v1 \
  -e OPENAI_API_KEY=sk-你的Key \
  -e ENABLE_OLLAMA_API=false \
  --name open-webui --restart always \
  ghcr.io/open-webui/open-webui:main
```

浏览器打开 `http://localhost:3000`（服务器上就是 `http://服务器IP:3000`）。

- `ENABLE_OLLAMA_API=false`：只用在线 API、不跑本地模型时关掉 Ollama，省得它一直尝试连接。
- **环境变量只在第一次启动时生效。** 之后在网页里改的设置存在数据库里，以网页上的为准。
- 用 `:main` 镜像，不要用 `:slim`：slim 版不带本地嵌入模型，上传文档、知识库功能会用不了（原因见下文）。

不用 Docker 的话，也可以用 Python 3.11 安装：

```bash
pip install open-webui
open-webui serve        # 默认端口 8080
```

## 第二步：创建管理员账号

第一次打开会让你创建账号，**第一个注册的账号自动成为管理员**。之后别人注册需要管理员审核。只给自己用、不想让别人注册，启动时加 `-e ENABLE_SIGNUP=false`。

## 第三步：添加连接

部署时已经用环境变量填好连接的，可以跳过这一步，直接看模型列表里有没有模型。

1. 点左下角头像 → **设置** → **管理员**（Settings > Admin）→ **连接**（Connections）。
2. 在「OpenAI API」那一栏点 **➕** 添加连接。
3. 填写：

| 字段 | 填什么 |
|---|---|
| URL | `https://hivegpt.cn/v1`（带 `/v1`，末尾不要加 `/`） |
| Auth | Bearer |
| API Key | 你的 HiveGPT Key |
| API Type | Chat Completions（默认值，不用改） |
| Model IDs | 可以不填；填了就只显示你填的这几个模型 |

4. 点 URL 旁边的 **验证连接**（Verify Connection），提示连接成功后点 **保存**。注意：直接保存不会检查连接对不对，一定先点验证。

### 只显示你要用的模型

不填 Model IDs 时，Open WebUI 会列出接口返回的所有模型（二十多个）。在 Model IDs 里逐个输入 `gpt-6.1-sol`、`gpt-6-luna`、`gpt-5.5` 并点 **+**，模型列表里就只剩这几个，用起来清爽很多。

## 第四步：开始聊天

回到聊天页，点左上角的模型名，选一个 HiveGPT 的模型，发一句话试试。日常用 `gpt-6.1-sol`，简单任务用便宜的 `gpt-6-luna`，各模型区别见 [GPT 模型怎么选](/connect/models)。

## 省 token：设置任务模型

Open WebUI 会在后台自动调用模型：给对话起标题、打标签、生成「你可能还想问」。**默认用的是你当前聊天的模型**，如果你聊天用的是贵的模型，这些后台任务也按贵的算。

改成便宜的模型：**设置 → 管理员 → 界面**（Interface）→ **任务**（Tasks）→ **外部任务模型**（External Task Model）选 `gpt-6-luna`。不需要的功能（如追问建议 Follow Up Generation）也可以在同一页关掉。

## 每个模型的默认参数

在 **工作空间 → 模型**（Workspace > Models）里可以给模型设置默认的系统提示词和高级参数（温度、推理强度 reasoning_effort、最大 token 数等）。

- 推理强度：简单问答设 `low` 更快更省，难题再调高。
- 报错 `Unsupported parameter: 'temperature'` 时，把这个模型和当前对话高级参数里的 Temperature 改回「默认」。

## 画图（gpt-image-2）

1. **设置 → 管理员 → 图像**（Images），打开图像生成。
2. 引擎选 **Open AI**，API 地址填 `https://hivegpt.cn/v1`，填上 Key，模型填 `gpt-image-2`，尺寸选 `1024x1024`（默认的 512x512 太小）。
3. 在 **工作空间 → 模型** 里给聊天用的模型勾上「图像生成」能力，聊天时在输入框的工具菜单里打开图像开关。

画图按张计费，价格见 [模型广场](https://hivegpt.cn/model-plaza)，提示词怎么写见 [生图提示词](/prompts/)。

## 上传文档和知识库

Open WebUI 的文档问答（RAG）需要「嵌入模型」把文档转成向量。HiveGPT 目前不提供嵌入模型，所以：

- **嵌入模型保持默认的本地模型**（设置 → 管理员 → 文档里的 Embedding Model Engine 不要改成 OpenAI），它在你自己的机器上运行，大约占 500MB 内存。
- 回答问题仍然用 HiveGPT 的模型。
- 这也是为什么要用 `:main` 镜像，`:slim` 不带本地嵌入模型。

## 常见问题

| 现象 | 原因 | 处理 |
|---|---|---|
| 验证连接失败、模型列表是空的 | URL 写错，或服务器连不上外网 | 确认 URL 是 `https://hivegpt.cn/v1`；在服务器上 `curl https://hivegpt.cn/v1/models` 测一下网络 |
| 浏览器能打开 hivegpt.cn，Open WebUI 却连不上 | 请求是 Open WebUI 的后端（容器里）发的，不是浏览器 | 检查服务器或容器的网络、代理设置 |
| 401 | Key 填错或已停用 | 重新复制 Key，见 [401 报错](/connect/errors/401) |
| 提示分组「不支持模型」 | 选的模型不在你的分组里 | 在 Model IDs 里只保留 `/v1/models` 返回的模型，见 [模型不存在](/connect/errors/model) |
| 改了环境变量不生效 | 环境变量只在第一次启动时生效 | 在网页里改，或启动时加 `-e RESET_CONFIG_ON_START=true` |
| 回复出现乱码一样的 `##`、`**`，或卡住不动 | 前面套了 Nginx，开了缓冲 | Nginx 里加 `proxy_buffering off;`，并开启 WebSocket 转发 |
| 余额消耗比预期快 | 后台起标题、追问建议也在用聊天模型 | 按上面「设置任务模型」改成 gpt-6-luna |
| 上传文档后报错 | 用了 `:slim` 镜像，或把嵌入引擎改成了 OpenAI | 换 `:main` 镜像，嵌入引擎保持默认 |

**能联网搜索吗？** Open WebUI 的联网搜索需要单独配置搜索引擎（在 设置 → 管理员 → 联网搜索 里选服务商并填它的 Key），和 HiveGPT 无关。

**手机上能用吗？** 能，用手机浏览器打开部署地址即可。要在外网访问，记得配 HTTPS 和强密码。

其他自建聊天站：[LobeChat、NextChat 接入](/connect/lobechat-nextchat) · 桌面客户端：[Cherry Studio](/connect/cherry-studio) · [Chatbox](/connect/chatbox)
