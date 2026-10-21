#!/usr/bin/env bash
# HiveGPT 一键安装与配置（macOS / Linux）：Claude Code、Claude 桌面 App、Codex CLI、Codex App
#
#   curl -fsSL https://hivegpt.cn/install.sh | bash
#   curl -fsSL https://hivegpt.cn/install.sh | HIVEGPT_KEY=sk-xxx bash
#   curl -fsSL https://hivegpt.cn/install.sh | bash -s -- --restore      # 恢复原配置
#   curl -fsSL https://hivegpt.cn/install.sh | bash -s -- --tools=codex  # 只配 Codex
#
# 这个脚本只负责准备 Node.js（没有 22+ 版本就从国内镜像下载免安装版到 ~/.hivegpt/node，
# 不用 sudo、不改系统），然后运行 hivegpt-setup.mjs 完成安装和配置。

set -euo pipefail

main() {
  local base="${HIVEGPT_BASE_URL:-https://hivegpt.cn}"
  base="${base%/}"
  base="${base%/v1}"
  local hive_dir="${HIVEGPT_HOME:-$HOME/.hivegpt}"
  HIVEGPT_TMP="$(mktemp -d "${TMPDIR:-/tmp}/hivegpt.XXXXXX")"
  trap 'rm -rf "$HIVEGPT_TMP"' EXIT
  local tmp="$HIVEGPT_TMP"

  say() { printf '\033[36m›\033[0m %s\n' "$*"; }
  die() { printf '\033[31m✗\033[0m %s\n' "$*" >&2; exit 1; }

  fetch() { # fetch URL FILE
    if command -v curl >/dev/null 2>&1; then
      curl -fsSL --retry 2 --connect-timeout 15 -o "$2" "$1"
    elif command -v wget >/dev/null 2>&1; then
      wget -q -T 15 -O "$2" "$1"
    else
      die "需要 curl 或 wget，请先安装其中一个（例如 sudo apt install curl）。"
    fi
  }

  node_ok() {
    [ -n "$1" ] && [ -x "$1" ] && "$1" -e 'process.exit(+process.versions.node.split(".")[0] >= 22 ? 0 : 1)' >/dev/null 2>&1
  }

  local os arch
  case "$(uname -s)" in
    Darwin) os=darwin ;;
    Linux) os=linux ;;
    *) die "不支持的系统 $(uname -s)。Windows 请在 PowerShell 里运行：irm $base/install.ps1 | iex" ;;
  esac
  case "$(uname -m)" in
    x86_64 | amd64) arch=x64 ;;
    arm64 | aarch64) arch=arm64 ;;
    *) die "不支持的 CPU 架构 $(uname -m)（unsupported architecture）。" ;;
  esac

  local node=""
  if node_ok "$(command -v node 2>/dev/null || true)"; then
    node="$(command -v node)"
  elif node_ok "$hive_dir/node/bin/node"; then
    node="$hive_dir/node/bin/node"
  fi

  if [ -z "$node" ]; then
    if [ "$os" = linux ] && ldd --version 2>&1 | grep -qi musl; then
      die "检测到 Alpine / musl 系统，暂不支持自动安装 Node。请先安装 Node.js 22 以上版本（apk add nodejs npm）后重新运行。"
    fi
    say "没有找到 Node.js 22+，从国内镜像下载免安装版到 $hive_dir/node …"
    local mirror version ok=""
    for mirror in https://cdn.npmmirror.com/binaries/node https://mirrors.cloud.tencent.com/nodejs-release https://mirrors.huaweicloud.com/nodejs; do
      fetch "$mirror/index.json" "$tmp/index.json" 2>/dev/null || { say "$mirror 连不上，换下一个镜像…"; continue; }
      version="$(grep -E '"lts":"[A-Za-z]' "$tmp/index.json" | head -n 1 | sed -E 's/.*"version":"(v[0-9.]+)".*/\1/' || true)"
      [ -n "$version" ] || continue
      local name="node-$version-$os-$arch"
      say "下载 Node.js $version（$mirror）…"
      if fetch "$mirror/$version/$name.tar.gz" "$tmp/node.tar.gz" && tar -xzf "$tmp/node.tar.gz" -C "$tmp"; then
        mkdir -p "$hive_dir"
        rm -rf "$hive_dir/node"
        mv "$tmp/$name" "$hive_dir/node"
        ok=1
        break
      fi
      say "从 $mirror 下载失败，换下一个镜像…"
    done
    [ -n "$ok" ] || die "Node.js 下载失败，请检查网络后重试（npmmirror / 腾讯云 / 华为云镜像都连不上）。"
    node="$hive_dir/node/bin/node"
    node_ok "$node" || die "Node.js 安装后无法运行，请把上面的输出发给客服。"
  fi

  fetch "$base/hivegpt-setup.mjs" "$tmp/hivegpt-setup.mjs" || die "下载配置脚本失败：$base/hivegpt-setup.mjs，请检查网络后重试。"

  export HIVEGPT_BASE_URL="$base"
  export HIVEGPT_NODE_BIN
  HIVEGPT_NODE_BIN="$(dirname "$node")"
  # `curl | bash` 时 stdin 是管道；把终端接给配置脚本，它才能提问。
  if (exec </dev/tty) 2>/dev/null; then
    "$node" "$tmp/hivegpt-setup.mjs" "$@" </dev/tty
  else
    "$node" "$tmp/hivegpt-setup.mjs" "$@"
  fi
}

main "$@"
