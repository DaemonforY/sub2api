---
title: A · AI 应用开发入门
description: 面向开发者和计算机专业学生：从第一次调用大模型 API 到结构化输出、Function Calling，再到 RAG、Agent 和上线。
---

# A · AI 应用开发入门

<TrackPage id="a" />

## 这条路线讲什么

很多教程停在「调通一个 API」。真做项目时，马上会遇到：回答要边生成边显示、要记住上下文、要稳定输出 JSON 给程序用、要让模型去查数据库或调接口。这条路线按做一个真实应用的顺序讲：

1. **调通**：拿到 Key，用 curl / Python / JavaScript 发出第一个请求，看懂返回。
2. **对话**：流式输出和多轮对话，理解 messages 和上下文成本。
3. **结构化**：用 JSON Schema 让模型返回程序能直接用的数据。
4. **工具**：Function Calling，让模型决定什么时候调用你的函数。
5. 之后是 Embedding / RAG、Agent 循环、MCP 和上线（即将上线）。

每节课的代码都可以复制到本地运行；页面里的「动手试试」用 HiveGPT 的接口直接运行，登录后每天有免费次数。
