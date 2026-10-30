# HiveGPT 桌面助手（骨架）

This is a zero-config desktop app: non-developers get Codex-level agent work without touching keys, terminals or OpenAI accounts. The plan, which contains business numbers, is kept locally as `docs/DESKTOP_ASSISTANT_PLAN.md` and is not in git (`docs/*` is ignored).

This folder is a separate pnpm project. It is not part of the web frontend build.

## What the skeleton does

1. **Engine.** On first run it downloads the pinned `@openai/codex` platform build (`src-tauri/src/engine.rs`, version `CODEX_VERSION`).
   - Source: npmmirror, falling back to the Tencent and Huawei npm mirrors.
   - Check: the tarball is verified against the registry's sha512 `integrity`.
   - Location: unpacked into the app data dir under `engine/<version>/`.
2. **Sign-in.** The app signs in with WeChat, device-authorization style (`src-tauri/src/login.rs`, server side `backend/internal/service/desktop_login.go`):
   - The app calls `POST /api/v1/auth/desktop/start` and shows a QR of `<site>/desktop-login?t=<ticket>`.
   - The user scans it in WeChat, signs in on the site with the normal web login (sign-up included) and taps 确认登录. That page is `frontend/src/views/user/DesktopLoginView.vue`.
   - The app polls `POST /api/v1/auth/desktop/poll` with a secret device code that never leaves the Rust side. On approval it receives this device's key, 「桌面助手 · <电脑名> #<id>」.
   - The key goes into the user's longest-running OpenAI subscription group (the Codex trial counts), otherwise the pay-as-you-go group. Signing in again on the same computer reuses and re-enables that key.
   - The key is stored in the system credential store. The account name and group are kept in `account.json`.
   - Debug builds still offer 「开发用：粘贴 Key」.
3. **Bridge.** `bridge.rs` starts `codex app-server` with its own `CODEX_HOME` (`<app data>/codex-home`):
   - It writes `config.toml` there (provider `hivegpt` → `https://hivegpt.cn/v1`, key via `HIVEGPT_API_KEY`) and a Chinese `AGENTS.md`.
   - It relays stdout lines as `engine-message` events and takes input through `engine_send`.
   - The user's own `~/.codex` is never read or written.
4. **Office kit.** `office.rs` installs a private Python plus the office libraries:
   - Python: python-build-standalone 3.12.15 from the npmmirror mirror, sha256 pinned and cross-checked with the GitHub release.
   - Libraries: openpyxl, xlrd, pandas, python-docx, python-pptx, pdfplumber, fpdf2, pillow and chardet, installed with pip from the Tsinghua, Aliyun or Huawei PyPI mirror.
   - Size: about 290 MB on disk. It installs right after the engine.
   - Wiring: when the kit is ready, its Python goes first on Codex's PATH and AGENTS.md gains a section on handling Excel, Word, PPT and PDF.
5. **Templates.** `src/lib/templates.ts` holds 8 office tasks, shown as cards in an empty chat. Each card has two actions:
   - 「用这个文件夹里的文件」 puts an editable prompt in the composer.
   - 「先用示例试试」 has Codex make fake files in `示例/` and run the task on them.
6. **UI.** The Vue UI (`src/lib/codex.ts` + `src/lib/chat.ts`) speaks the JSON-RPC:
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

Checks: `pnpm typecheck` (Vue) and `cargo build` in `src-tauri/` (Rust 1.80+). Windows installer: `pnpm tauri build --bundles nsis` on Windows, or the Desktop App Build workflow.

## Not done yet

- **`GET /api/v1/desktop/version`**, to serve the engine version and per-platform hashes. Today the hash comes from the same mirror as the file, so it catches corruption, not a malicious mirror.
- History, analytics events, auto-update, and more templates. The office kit and the 8 templates are done.
- **Testing on a real Windows PC.** `.github/workflows/desktop-build.yml` builds the installer: NSIS, per-user, about 5 MB, artifact `hivegpt-desktop-windows-x64`. It runs on `desktop/**` changes on main and on `feat/desktop-*`, or by hand. The installer has not been run on Windows yet. On Windows the app runs Codex's one-time unelevated sandbox setup when it connects.
- **Code signing** for Windows and macOS.
