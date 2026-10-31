# HiveGPT 桌面助手

This is a zero-config desktop app: non-developers get Codex-level agent work without touching keys, terminals or OpenAI accounts. The plan, which contains business numbers, is kept locally as `docs/DESKTOP_ASSISTANT_PLAN.md` and is not in git (`docs/*` is ignored).

This folder is a separate pnpm project. It is not part of the web frontend build.

## What the skeleton does

1. **Engine.** On first run it downloads the pinned `@openai/codex` platform build (`src-tauri/src/engine.rs`, version `CODEX_VERSION`).
   - Source: npmmirror, falling back to the Tencent and Huawei npm mirrors.
   - Check: the tarball must match a sha512 pinned per platform in `engine.rs` (taken from registry.npmjs.org, cross-checked with npmmirror). A mirror listing another hash is skipped. Update the hashes together with `CODEX_VERSION`.
   - Location: unpacked into the app data dir under `engine/<version>/`.
2. **Sign-in.** The app signs in with WeChat, device-authorization style (`src-tauri/src/login.rs`, server side `backend/internal/service/desktop_login.go`):
   - The app calls `POST /api/v1/auth/desktop/start` and shows a QR of `<site>/desktop-login?t=<ticket>`.
   - The user scans it in WeChat, signs in on the site with the normal web login (sign-up included) and taps 确认登录. That page is `frontend/src/views/user/DesktopLoginView.vue`.
   - The app polls `POST /api/v1/auth/desktop/poll` with a secret device code that never leaves the Rust side. On approval it receives this device's key, 「桌面助手 · <电脑名> #<id>」.
   - Anti-phishing: the phone page shows the computer's network (last part hidden) and whether it is the phone's network (same IP, IPv4 /24 or IPv6 /48). On another network, the server refuses to approve unless the user ticks 「这台电脑就在我面前」. Device names posing as HiveGPT or an official check are replaced with 未命名电脑.
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
5. **Skills (技能库).** Operators edit skills in 后台 → 桌面助手 (`/admin/desktop`, table `desktop_skills`, migration 321).
   - On every launch, before the engine starts, `skills.rs` fetches `GET /api/v1/desktop/skills`.
   - Each skill's `SKILL.md` goes to `<codex-home>/skills/hivegpt-<slug>/`, rewritten only when its version changed; skills taken down are removed. Codex then also uses a skill when the user just describes the task.
   - The cards in the empty chat come from the same list. Offline, the last list is used; with none at all, the built-in `src/lib/templates.ts` (the same 8 office tasks the migration seeds).
   - `config.toml` switches off every other skill: the user's own `~/.agents/skills` and Codex's bundled system skills.
   Each card has two actions:
   - 「用这个文件夹里的文件」 puts an editable prompt in the composer.
   - 「先用示例试试」 has Codex make fake files in `示例/` and run the task on them.
6. **UI.** The Vue UI (`src/lib/codex.ts` + `src/lib/chat.ts`) speaks the JSON-RPC:
   - Startup: `initialize` → `thread/start` (cwd = chosen folder, sandbox `workspace-write`, approval `on-request`).
   - Each message: `turn/start`, with streamed `item/agentMessage/delta`.
   - Work items: commands and file changes are shown as steps.
   - Approvals: command/file approvals become a Chinese dialog.
7. **History.** A sidebar lists past conversations from every folder (`thread/list`). Opening one runs `thread/resume` + `thread/read` and continues it in its own folder. Rename uses `thread/name/set`; ✕ archives with `thread/archive`. Only messages come back from `thread/read`; the command and file steps of old turns are not shown.
8. **Undo a turn (撤销这一轮).**
   - Before every turn, `snapshot.rs` copies the working folder to `<app data>/snapshots/<thread>/`. Folders over 200 MB or 5,000 files are skipped: that turn can then only be undone in the conversation.
   - Undo puts every saved file back, then runs `thread/rollback` (1 turn). Files the turn created are moved to `<app data>/撤销回收站/<time>/`, never deleted.
   - The last 10 snapshots per conversation are kept; conversations idle for 14 days lose theirs.
9. **Balance.** The header shows the wallet balance or subscription allowance and today's spend, from the gateway's `GET /v1/usage`, plus 充值 below $1. After each turn it shows what the turn cost.
10. **Analytics.** `track.rs` posts `desktop_*` events to `/api/v1/events` (app `desktop`, visitor = device id). No prompts or file names are sent. The events cover install, open, engine ready/failed, sign-in, task start/done/failed (template, seconds, reason), undo, history and 充值. Debug builds send nothing unless `HIVEGPT_DEV_TRACK=1`.

## Run

```bash
cd desktop
pnpm install
pnpm tauri dev
```

For development you can skip the key form: `HIVEGPT_API_KEY=sk-… pnpm tauri dev`. `HIVEGPT_BASE_URL` points it at another sub2api. Both are read only by debug builds; release builds always use hivegpt.cn and the key from the credential store.

Debug-build switches:
- `HIVEGPT_DEV_PROMPT`: finish setup and send this prompt.
- `HIVEGPT_DEV_WORKSPACE`: the working folder.
- `HIVEGPT_DEV_UNDO=1`: undo that turn once it finishes.
- `HIVEGPT_DEV_TRACK=1`: send analytics.

**UI preview without Tauri.** Run `pnpm dev`, then open `http://localhost:1420/dev/mock.html` in a browser. `dev/mock-tauri.ts` fakes every command and the Codex engine, so the screens can be clicked through. Nothing in `dev/` is bundled.

Checks: `pnpm typecheck` (Vue) and `cargo build` in `src-tauri/` (Rust 1.80+). Windows installer: `pnpm tauri build --bundles nsis` on Windows, or the Desktop App Build workflow.

## Not done yet

- Auto-update, server-delivered skills (phase 2 B), scheduled tasks (phase 2 C).
- **Testing on a real Windows PC.** `.github/workflows/desktop-build.yml` builds the installer: NSIS, per-user, about 5 MB, artifact `hivegpt-desktop-windows-x64`. It runs on `desktop/**` changes on main and on `feat/desktop-*`, or by hand. The installer has not been run on Windows yet. On Windows the app runs Codex's one-time unelevated sandbox setup when it connects.
- **Code signing** for Windows and macOS.
