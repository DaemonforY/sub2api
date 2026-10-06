---
title: Codex 教程
description: OpenAI Codex 中文教程：从安装、接入 HiveGPT 到配置、沙箱、AGENTS.md、Skills、MCP、子代理和 CI/CD，CLI、桌面应用、IDE 扩展、云端都讲到。
---

# Codex 教程

Codex 是 OpenAI 的编程智能体：它在你的项目里读代码、改文件、跑命令、看测试结果，一步步把任务做完。它有命令行（CLI）、IDE 扩展、桌面应用和云端几种用法，本地的几种共用同一份 `~/.codex` 配置。

这套教程按「先用起来，再用顺手，最后用进团队流程」排了 27 篇，内容对照 Codex 的官方源码和命令行帮助整理，会随版本更新。

::: tip 国内直接用
不需要海外账号：在 HiveGPT 创建一个 Key，用「使用密钥」弹窗生成配置，几分钟就能让 Codex 跑起来，按用量计费。见 [接入 HiveGPT](/codex/hivegpt)。
:::

## 怎么读

| 你现在 | 建议从这里开始 |
|---|---|
| 没用过 Codex | [Codex 是什么](/codex/intro) → [安装与登录](/codex/install) → [接入 HiveGPT](/codex/hivegpt) → [第一个任务](/codex/first-task) |
| 已经能用，想更顺手 | [斜杠命令](/codex/slash-commands)、[提示词最佳实践](/codex/prompting)、[AGENTS.md](/codex/agents-md)、[速查表](/codex/cheatsheet) |
| 想改配置、控权限 | [config.toml 基础](/codex/config)、[沙箱与审批](/codex/sandbox)、[高级配置](/codex/config-advanced) |
| 想扩展能力 | [Skills](/codex/skills)、[MCP](/codex/mcp)、[子代理](/codex/subagents)、[规则与钩子](/codex/hooks) |
| 想用进团队和流水线 | [非交互模式与 CI/CD](/codex/exec)、[Worktree 与并行任务](/codex/worktrees)、[安全与团队管理](/codex/security) |
| 遇到报错 | [常见问题排查](/codex/troubleshooting) |

## 目录

**开始**：[Codex 是什么](/codex/intro) · [核心概念](/codex/concepts) · [安装与登录](/codex/install) · [接入 HiveGPT](/codex/hivegpt) · [接入 DeepSeek 等其他模型](/codex/providers) · [第一个任务](/codex/first-task)

**使用形态**：[桌面应用与界面](/codex/app) · [IDE 扩展](/codex/ide) · [云端任务与 GitHub](/codex/cloud) · [Windows 上使用](/codex/windows)

**CLI 详解**：[斜杠命令](/codex/slash-commands) · [命令行参数](/codex/cli-reference) · [config.toml 基础](/codex/config) · [高级配置](/codex/config-advanced) · [沙箱与审批](/codex/sandbox) · [模型与推理强度](/codex/models)

**进阶**：[提示词最佳实践](/codex/prompting) · [AGENTS.md](/codex/agents-md) · [Skills](/codex/skills) · [MCP](/codex/mcp) · [子代理](/codex/subagents) · [规则与钩子](/codex/hooks) · [电脑操控](/codex/computer-use)

**实战**：[非交互模式与 CI/CD](/codex/exec) · [Worktree 与并行任务](/codex/worktrees) · [工作流与实战示例](/codex/workflows) · [安全与团队管理](/codex/security) · [速查表](/codex/cheatsheet) · [常见问题排查](/codex/troubleshooting)

## 想边做边学？

[C · AI 编程助手上手](/c/) 用 Codex 带你从零做一个小网站并发布上线，有动手练习和结业证书，适合配合这套教程一起学。

## 参考

- 官方仓库：[github.com/openai/codex](https://github.com/openai/codex)
- 官方文档：[developers.openai.com/codex](https://developers.openai.com/codex)

Codex 更新很快，命令和配置项以你本机 `codex --help` 和官方文档为准；发现过时的地方，欢迎在 HiveGPT 反馈。
