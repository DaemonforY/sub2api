# HiveGPT 桌面助手（骨架）

This is a zero-config desktop app: non-developers get Codex-level agent work without touching keys, terminals or OpenAI accounts. The plan, which contains business numbers, is kept locally as `docs/DESKTOP_ASSISTANT_PLAN.md` and is not in git (`docs/*` is ignored).

This folder is a separate pnpm project. It is not part of the web frontend build.

## What the skeleton does

1. **Engine.** On first run it downloads the pinned `@openai/codex` platform build (`src-tauri/src/engine.rs`, version `CODEX_VERSION`).
   - Source: npmmirror, falling back to the Tencent and Huawei npm mirrors.
   - Check: the tarball is verified against the registry's sha512 `integrity`.
   - Location: unpacked into the app data dir under `engine/<version>/`.
2. **Key.** The key is stored in the system credential store (`settings.rs`). Until WeChat sign-in exists, it is pasted on the setup screen.
3. **Bridge.** `bridge.rs` starts `codex app-server` with its own `CODEX_HOME` (`<app data>/codex-home`):
   - It writes `config.toml` there (provider `hivegpt` → `https://hivegpt.cn/v1`, key via `HIVEGPT_API_KEY`) and a Chinese `AGENTS.md`.
   - It relays stdout lines as `engine-message` events and takes input through `engine_send`.
   - The user's own `~/.codex` is never read or written.
4. **UI.** The Vue UI (`src/lib/codex.ts` + `src/lib/chat.ts`) speaks the JSON-RPC:
   - Startup: `initialize` → `thread/start` (cwd = chosen folder, sandbox `workspace-write`, approval `on-request`).
   - Each message: `turn/start`, with streamed `item/agentMessage/delta`.
   - Work items: commands and file changes are shown as steps.
   - Approvals: command/file approvals become a Chinese dialog.

## Run

```bash
cd desktop
pnpm install
pnpm tauri dev
```

For development you can skip the key form: `HIVEGPT_API_KEY=sk-… pnpm tauri dev`. `HIVEGPT_BASE_URL` points it at another sub2api.

Checks: `pnpm typecheck` (Vue) and `cargo build` in `src-tauri/` (Rust 1.80+).

## Not done yet

- **WeChat QR sign-in** and `POST /api/v1/desktop/session`, which issues a device key in the Codex trial group.
- **`GET /api/v1/desktop/version`**, to serve the engine version and per-platform hashes. Today the hash comes from the same mirror as the file, so it catches corruption, not a malicious mirror.
- Portable Python + office libraries, the task-template library, history, analytics events, auto-update.
- **Windows build.** It must be built on Windows; plan: a GitHub Actions `windows-latest` job running `pnpm tauri build`. Until then this has only been run on macOS.
- **Code signing** for Windows and macOS.
