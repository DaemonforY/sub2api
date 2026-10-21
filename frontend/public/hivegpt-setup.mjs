#!/usr/bin/env node
// HiveGPT 一键安装与配置 — Claude Code / Claude 桌面 App / Codex CLI / Codex App
//
// 由 install.sh（macOS / Linux）或 install.ps1（Windows）下载并用 Node 22+ 运行，也可以直接：
//   node hivegpt-setup.mjs [--tools=claude|codex|all] [--yes] [--no-install] [--restore]
// 环境变量：HIVEGPT_KEY、HIVEGPT_BASE_URL、HIVEGPT_TOOLS、HIVEGPT_YES=1、HIVEGPT_NO_INSTALL=1、
//           HIVEGPT_RESTORE=1、HIVEGPT_NPM_REGISTRY、HIVEGPT_HOME
//
// 原则：
// - 所有下载走国内镜像（npmmirror → 腾讯云 → 华为云），不访问 GitHub / claude.ai / openai.com。
// - 改任何文件前先备份到 ~/.hivegpt/backups/<时间>/，--restore 一键还原。
// - 合并而不是覆盖：已有的 MCP、权限、hooks、项目信任等设置原样保留；DeepSeek 等旧配置里
//   和 HiveGPT 冲突的项（地址、Key、不存在的模型名）才会改掉。
// - claude.ai / ChatGPT 账号登录不删除：Claude Code 优先用 ANTHROPIC_AUTH_TOKEN，Codex 用
//   config.toml 里的 Key（API Key 模式，不碰 auth.json），桌面 App 的网关配置单独存放。

import fs from 'node:fs'
import os from 'node:os'
import path from 'node:path'
import crypto from 'node:crypto'
import { spawnSync } from 'node:child_process'
import { Readable } from 'node:stream'
import { pipeline } from 'node:stream/promises'

const IS_WIN = process.platform === 'win32'
const IS_MAC = process.platform === 'darwin'
const HOME = os.homedir()
const HIVE_DIR = process.env.HIVEGPT_HOME || path.join(HOME, '.hivegpt')
const BASE = normalizeBase(process.env.HIVEGPT_BASE_URL || 'https://hivegpt.cn')
const NPM_PREFIX = path.join(HIVE_DIR, 'npm')
const NPM_BIN = IS_WIN ? NPM_PREFIX : path.join(NPM_PREFIX, 'bin')
const NPM_REGISTRIES = unique([
  process.env.HIVEGPT_NPM_REGISTRY,
  'https://registry.npmmirror.com',
  'https://mirrors.cloud.tencent.com/npm/',
  'https://repo.huaweicloud.com/repository/npm/'
])
const CODEX_CLIENT_VERSION = '0.147.0'
const MIN_CODEX_VERSION = '0.130.0'
const DESKTOP_PROFILE_NAME = 'HiveGPT'
const RC_MARK_BEGIN = '# >>> HiveGPT >>>'
const RC_MARK_END = '# <<< HiveGPT <<<'
const RC_COMMENT = '# [HiveGPT 已注释，恢复请运行 --restore] '

// npm postinstall scripts and codex.js (#!/usr/bin/env node) need this Node on PATH.
process.env.PATH = unique([path.dirname(process.execPath), NPM_BIN, ...(process.env.PATH || '').split(path.delimiter)]).join(path.delimiter)

const CLAUDE_AUTH_VARS = ['ANTHROPIC_BASE_URL', 'ANTHROPIC_AUTH_TOKEN', 'ANTHROPIC_API_KEY']
const CLAUDE_MODEL_VARS = [
  'ANTHROPIC_MODEL',
  'ANTHROPIC_SMALL_FAST_MODEL',
  'ANTHROPIC_DEFAULT_OPUS_MODEL',
  'ANTHROPIC_DEFAULT_SONNET_MODEL',
  'ANTHROPIC_DEFAULT_HAIKU_MODEL',
  'ANTHROPIC_DEFAULT_FABLE_MODEL',
  'CLAUDE_CODE_SUBAGENT_MODEL'
]
// Settings that route Claude Code somewhere other than ANTHROPIC_BASE_URL.
const CLAUDE_PROVIDER_SWITCHES = ['CLAUDE_CODE_USE_BEDROCK', 'CLAUDE_CODE_USE_VERTEX', 'CLAUDE_CODE_USE_FOUNDRY', 'CLAUDE_CODE_OAUTH_TOKEN']
const CLAUDE_MODEL_ALIASES = new Set(['default', 'opus', 'sonnet', 'haiku', 'fable', 'opusplan', 'best'])

// ---------- output ----------

const color = process.stdout.isTTY && !process.env.NO_COLOR
const paint = (code) => (s) => (color ? `\x1b[${code}m${s}\x1b[0m` : String(s))
const bold = paint('1')
const green = paint('32')
const yellow = paint('33')
const red = paint('31')
const cyan = paint('36')
const dim = paint('2')

const info = (s) => console.log(`${cyan('›')} ${s}`)
const ok = (s) => console.log(`${green('✓')} ${s}`)
const warn = (s) => console.log(`${yellow('!')} ${s}`)
const step = (s) => console.log(`\n${bold(s)}`)

class SetupError extends Error {}
const fail = (s) => {
  throw new SetupError(s)
}

// ---------- small helpers ----------

function unique(list) {
  return [...new Set(list.filter(Boolean))]
}

function normalizeBase(value) {
  return String(value).trim().replace(/\/+$/, '').replace(/\/v1$/i, '')
}

function exists(p) {
  try {
    fs.accessSync(p)
    return true
  } catch {
    return false
  }
}

function isFile(p) {
  try {
    return fs.statSync(p).isFile()
  } catch {
    return false
  }
}

function readText(p) {
  return fs.readFileSync(p, 'utf8').replace(/^\uFEFF/, '')
}

function writeFileSafe(p, content, mode) {
  fs.mkdirSync(path.dirname(p), { recursive: true })
  let keepMode = mode
  try {
    keepMode = keepMode ?? (fs.statSync(p).mode & 0o777)
  } catch {
    /* new file */
  }
  const tmp = `${p}.hivegpt-tmp-${process.pid}`
  fs.writeFileSync(tmp, content, keepMode ? { mode: keepMode } : undefined)
  fs.renameSync(tmp, p)
  if (keepMode && !IS_WIN) fs.chmodSync(p, keepMode)
}

function readJson(p, fallback) {
  if (!exists(p)) return fallback
  const text = readText(p).trim()
  if (!text) return fallback
  try {
    return JSON.parse(text)
  } catch (error) {
    fail(`${p} 不是合法的 JSON，没有改动它（invalid JSON: ${error.message}）。请修好或删掉这个文件后重新运行。`)
  }
}

function writeJson(p, value, mode) {
  writeFileSafe(p, `${JSON.stringify(value, null, 2)}\n`, mode)
}

function compareVersions(a, b) {
  const pa = String(a).split('.').map((n) => parseInt(n, 10) || 0)
  const pb = String(b).split('.').map((n) => parseInt(n, 10) || 0)
  for (let i = 0; i < Math.max(pa.length, pb.length); i++) {
    if ((pa[i] || 0) !== (pb[i] || 0)) return (pa[i] || 0) - (pb[i] || 0)
  }
  return 0
}

function maskKey(key) {
  return key.length <= 10 ? '***' : `${key.slice(0, 6)}…${key.slice(-4)}`
}

function tomlString(value) {
  return `"${String(value).replace(/\\/g, '\\\\').replace(/"/g, '\\"')}"`
}

function run(cmd, args, options = {}) {
  return spawnSync(cmd, args, { encoding: 'utf8', windowsHide: true, ...options })
}

// ---------- input ----------

const interactive = Boolean(process.stdin.isTTY) && !truthy(process.env.HIVEGPT_YES)

function truthy(value) {
  return /^(1|true|yes|y)$/i.test(String(value || '').trim())
}

function readLine(question, { secret = false } = {}) {
  return new Promise((resolve) => {
    const stdin = process.stdin
    process.stdout.write(question)
    let buffer = ''
    const wasRaw = stdin.isRaw
    stdin.setRawMode?.(true)
    stdin.resume()
    stdin.setEncoding('utf8')
    const onData = (chunk) => {
      for (const ch of chunk) {
        if (ch === '\r' || ch === '\n') {
          stdin.setRawMode?.(wasRaw)
          stdin.pause()
          stdin.off('data', onData)
          process.stdout.write('\n')
          resolve(buffer)
          return
        }
        if (ch === '\u0003') {
          stdin.setRawMode?.(wasRaw)
          process.stdout.write('\n')
          process.exit(130)
        }
        if (ch === '\u007f' || ch === '\b') {
          if (buffer.length) {
            buffer = buffer.slice(0, -1)
            process.stdout.write('\b \b')
          }
          continue
        }
        if (ch < ' ') continue
        buffer += ch
        process.stdout.write(secret ? '*' : ch)
      }
    }
    stdin.on('data', onData)
  })
}

async function ask(question, fallback) {
  if (!interactive) return fallback
  const answer = (await readLine(`${question}${fallback ? dim(` [${fallback}]`) : ''}: `)).trim()
  return answer || fallback
}

async function confirm(question, fallback = true) {
  if (!interactive) return fallback
  const answer = (await readLine(`${question} ${dim(fallback ? '[Y/n]' : '[y/N]')}: `)).trim()
  if (!answer) return fallback
  return /^y/i.test(answer)
}

// ---------- network ----------

async function httpGet(url, headers = {}, timeoutMs = 20000) {
  return fetch(url, { headers, signal: AbortSignal.timeout(timeoutMs), redirect: 'follow' })
}

async function download(url, dest, label) {
  const res = await httpGet(url, {}, 30 * 60 * 1000)
  if (!res.ok || !res.body) fail(`下载 ${label} 失败（HTTP ${res.status}）：${url}`)
  const total = Number(res.headers.get('content-length')) || 0
  let received = 0
  let lastShown = -1
  const body = Readable.fromWeb(res.body)
  body.on('data', (chunk) => {
    received += chunk.length
    if (!total || !process.stdout.isTTY) return
    const pct = Math.floor((received / total) * 100)
    if (pct !== lastShown && pct % 5 === 0) {
      lastShown = pct
      process.stdout.write(`\r  ${label} ${pct}%`)
    }
  })
  fs.mkdirSync(path.dirname(dest), { recursive: true })
  await pipeline(body, fs.createWriteStream(dest))
  if (total && process.stdout.isTTY) process.stdout.write('\r' + ' '.repeat(40) + '\r')
}

async function fetchModels(key) {
  let res
  try {
    res = await httpGet(`${BASE}/v1/models`, {
      Authorization: `Bearer ${key}`,
      'anthropic-version': '2023-06-01',
      Accept: 'application/json'
    })
  } catch (error) {
    fail(`连不上 ${BASE}（network error: ${error.cause?.code || error.message}）。请检查网络后重试；公司网络可以换手机热点试试。`)
  }
  const text = await res.text()
  let payload = null
  try {
    payload = JSON.parse(text)
  } catch {
    /* not JSON */
  }
  const message = payload?.error?.message || payload?.message || ''
  if (res.status === 401) fail(`这个 Key 无效或已被删除（401 Unauthorized）。请到 ${BASE}/keys 重新复制完整的 Key。`)
  if (res.status === 403) {
    fail(`这个 Key 暂时不能用（403 Forbidden${message ? `: ${message}` : ''}）。请到 ${BASE}/keys 检查：Key 是否已禁用、是否设置了 IP 白名单、是否分配了分组。`)
  }
  if (res.status === 429) fail(`请求太频繁或额度已用完（429 Too Many Requests${message ? `: ${message}` : ''}）。请稍后再试，或到 ${BASE} 查看余额。`)
  if (!res.ok) fail(`验证 Key 失败（HTTP ${res.status}${message ? `: ${message}` : ''}）。请稍后重试，或联系客服。`)
  const ids = Array.isArray(payload?.data) ? payload.data.map((m) => m?.id).filter((id) => typeof id === 'string') : []
  return ids
}

async function fetchCodexCatalog(key) {
  try {
    const res = await httpGet(`${BASE}/v1/models?client_version=${CODEX_CLIENT_VERSION}`, {
      Authorization: `Bearer ${key}`,
      Accept: 'application/json'
    })
    if (!res.ok) return null
    const payload = await res.json()
    const models = Array.isArray(payload?.models) ? payload.models : []
    // Codex refuses to start on a catalog entry missing these, so a malformed one is left out.
    const valid = models.length > 0 && models.every((m) => typeof m?.slug === 'string' && typeof m?.description === 'string')
    return valid ? payload : null
  } catch {
    return null
  }
}

// ---------- backups / restore ----------

const backupStamp = (() => {
  const d = new Date()
  const p = (n) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${p(d.getMonth() + 1)}-${p(d.getDate())}_${p(d.getHours())}-${p(d.getMinutes())}-${p(d.getSeconds())}`
})()
const backupDir = path.join(HIVE_DIR, 'backups', backupStamp)
const manifest = { version: 1, createdAt: new Date().toISOString(), base: BASE, entries: [] }

function saveManifest() {
  fs.mkdirSync(backupDir, { recursive: true, mode: 0o700 })
  writeJson(path.join(backupDir, 'manifest.json'), manifest, 0o600)
}

/** Copies a file aside before it is first changed (or records that it didn't exist). */
function backupFile(p) {
  if (manifest.entries.some((e) => e.kind === 'file' && e.path === p)) return
  const entry = { kind: 'file', path: p, existed: isFile(p) }
  if (entry.existed) {
    const name = `${manifest.entries.length}-${path.basename(p)}`
    fs.mkdirSync(path.join(backupDir, 'files'), { recursive: true, mode: 0o700 })
    fs.copyFileSync(p, path.join(backupDir, 'files', name))
    if (!IS_WIN) fs.chmodSync(path.join(backupDir, 'files', name), 0o600)
    entry.backup = name
  }
  manifest.entries.push(entry)
  saveManifest()
}

/** For files other programs keep rewriting (~/.claude.json): remember only the keys we touch. */
function backupJsonKeys(p, obj, keys) {
  const values = {}
  for (const key of keys) {
    values[key] = Object.prototype.hasOwnProperty.call(obj, key) ? { present: true, value: obj[key] } : { present: false }
  }
  manifest.entries.push({ kind: 'json-keys', path: p, existed: isFile(p), values })
  saveManifest()
}

function backupWinEnv(name, value) {
  if (manifest.entries.some((e) => e.kind === 'win-env' && e.name === name)) return
  manifest.entries.push({ kind: 'win-env', name, value: value ?? null })
  saveManifest()
}

function restore() {
  const root = path.join(HIVE_DIR, 'backups')
  const stamps = exists(root)
    ? fs.readdirSync(root).filter((d) => !d.endsWith('.restored') && isFile(path.join(root, d, 'manifest.json'))).sort()
    : []
  const wanted = /^\d{4}-/.test(String(process.env.HIVEGPT_RESTORE || '')) ? process.env.HIVEGPT_RESTORE : stamps[stamps.length - 1]
  if (!wanted || !stamps.includes(wanted)) fail(`没有找到可恢复的备份（${root}）。`)
  const dir = path.join(root, wanted)
  const saved = JSON.parse(readText(path.join(dir, 'manifest.json')))
  step(`恢复 ${wanted} 之前的配置`)
  for (const entry of [...saved.entries].reverse()) {
    try {
      if (entry.kind === 'file') {
        if (entry.existed) {
          writeFileSafe(entry.path, fs.readFileSync(path.join(dir, 'files', entry.backup)))
          ok(`已还原 ${entry.path}`)
        } else if (exists(entry.path)) {
          fs.rmSync(entry.path, { force: true })
          ok(`已删除脚本新建的 ${entry.path}`)
        }
      } else if (entry.kind === 'json-keys') {
        if (!exists(entry.path)) continue
        const obj = readJson(entry.path, {})
        for (const [key, v] of Object.entries(entry.values)) {
          if (v.present) obj[key] = v.value
          else delete obj[key]
        }
        writeJson(entry.path, obj)
        ok(`已还原 ${entry.path} 里的 ${Object.keys(entry.values).join('、')}`)
      } else if (entry.kind === 'win-env') {
        winSetUserEnv(entry.name, entry.value)
        ok(`已还原用户环境变量 ${entry.name}`)
      }
    } catch (error) {
      warn(`还原 ${entry.path || entry.name} 失败：${error.message}`)
    }
  }
  fs.renameSync(dir, `${dir}.restored`)
  console.log('')
  ok('恢复完成。请新开一个终端，并完全退出再重新打开 Claude / Codex App。')
  if (stamps.length > 1) info('之前运行过多次安装的话，可以再运行一次 --restore，每次往回退一步。')
  info(`脚本装的 Node 和命令行工具在 ${HIVE_DIR}，不需要的话可以直接删除这个文件夹。`)
}

// ---------- Windows user environment ----------

function powershell(script, env = {}) {
  const prelude = '[Console]::OutputEncoding=[Text.Encoding]::UTF8;$ErrorActionPreference="Stop";'
  return run('powershell.exe', ['-NoProfile', '-NonInteractive', '-ExecutionPolicy', 'Bypass', '-Command', prelude + script], {
    env: { ...process.env, ...env }
  })
}

function winUserEnv() {
  const r = powershell(
    '$k=[Microsoft.Win32.Registry]::CurrentUser.OpenSubKey("Environment");$h=@{};' +
      'foreach($n in $k.GetValueNames()){$h[$n]=[string]$k.GetValue($n,$null,"DoNotExpandEnvironmentNames")};' +
      '[Console]::Out.Write(($h|ConvertTo-Json -Compress))'
  )
  if (r.status !== 0) return {}
  try {
    return JSON.parse(r.stdout.trim() || '{}') || {}
  } catch {
    return {}
  }
}

function winSetUserEnv(name, value) {
  // Keeps REG_EXPAND_SZ for values with %VARS% (PATH), unlike [Environment]::SetEnvironmentVariable
  // on Windows PowerShell 5.1; the dummy variable afterwards broadcasts WM_SETTINGCHANGE.
  const r = powershell(
    '$k=[Microsoft.Win32.Registry]::CurrentUser.OpenSubKey("Environment",$true);' +
      'if($env:HG_DELETE -eq "1"){$k.DeleteValue($env:HG_NAME,$false)}' +
      'else{$kind=if($env:HG_VALUE -like "*%*"){"ExpandString"}else{"String"};$k.SetValue($env:HG_NAME,$env:HG_VALUE,$kind)};' +
      '$k.Close();[Environment]::SetEnvironmentVariable("HIVEGPT_REFRESH","1","User");[Environment]::SetEnvironmentVariable("HIVEGPT_REFRESH",$null,"User")',
    { HG_NAME: name, HG_VALUE: value ?? '', HG_DELETE: value == null ? '1' : '0' }
  )
  if (r.status !== 0) fail(`写入用户环境变量 ${name} 失败：${(r.stderr || '').trim()}`)
}

// ---------- PATH / shell profiles ----------

function shellRcFiles() {
  const files = []
  const shell = path.basename(process.env.SHELL || '')
  const zshrc = path.join(process.env.ZDOTDIR || HOME, '.zshrc')
  if (IS_MAC || shell === 'zsh' || exists(zshrc)) files.push(zshrc)
  // Never create ~/.bash_profile: bash would then stop reading an existing ~/.profile.
  const bashrc = path.join(HOME, '.bashrc')
  if (!IS_MAC || exists(bashrc)) files.push(bashrc)
  const bashProfile = path.join(HOME, '.bash_profile')
  if (exists(bashProfile) || (IS_MAC && shell === 'bash' && !exists(path.join(HOME, '.profile')))) files.push(bashProfile)
  const profile = path.join(HOME, '.profile')
  if (exists(profile)) files.push(profile)
  return unique(files)
}

function fishConfFile() {
  const dir = path.join(process.env.XDG_CONFIG_HOME || path.join(HOME, '.config'), 'fish')
  return exists(dir) ? path.join(dir, 'conf.d', 'hivegpt.fish') : null
}

function persistPath(dirs) {
  if (!dirs.length) return
  if (IS_WIN) {
    const userEnv = winUserEnv()
    const current = userEnv.Path ?? userEnv.PATH ?? ''
    const parts = current.split(';').filter(Boolean)
    const missing = dirs.filter((d) => !parts.some((p) => p.toLowerCase() === d.toLowerCase()))
    if (!missing.length) return
    backupWinEnv('Path', current || null)
    winSetUserEnv('Path', [...missing, ...parts].join(';'))
    ok(`已把 ${missing.join('、')} 加入用户 PATH`)
    return
  }
  const line = `export PATH="${dirs.join(':')}:$PATH"`
  const block = `${RC_MARK_BEGIN}\n${line}\n${RC_MARK_END}\n`
  for (const rc of shellRcFiles()) {
    const text = exists(rc) ? readText(rc) : ''
    const pattern = new RegExp(`${escapeRe(RC_MARK_BEGIN)}[\\s\\S]*?${escapeRe(RC_MARK_END)}\\n?`)
    const next = pattern.test(text) ? text.replace(pattern, block) : `${text}${text && !text.endsWith('\n') ? '\n' : ''}\n${block}`
    if (next === text) continue
    backupFile(rc)
    writeFileSafe(rc, next)
    ok(`已在 ${rc} 里加入 PATH`)
  }
  const fish = fishConfFile()
  if (fish) {
    const content = `${RC_MARK_BEGIN}\nset -gx PATH ${dirs.map((d) => `"${d}"`).join(' ')} $PATH\n${RC_MARK_END}\n`
    if (!exists(fish) || readText(fish) !== content) {
      backupFile(fish)
      writeFileSafe(fish, content)
    }
  }
}

function escapeRe(s) {
  return s.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
}

/** Comments out `export ANTHROPIC_…=` lines (DeepSeek and other relays) so they can't override HiveGPT. */
function cleanShellClaudeVars(modelIds) {
  if (IS_WIN) return cleanWindowsClaudeVars(modelIds)
  const files = [...shellRcFiles(), path.join(HOME, '.zprofile'), path.join(HOME, '.zshenv')].filter(isFile)
  const fishDir = path.join(process.env.XDG_CONFIG_HOME || path.join(HOME, '.config'), 'fish')
  if (isFile(path.join(fishDir, 'config.fish'))) files.push(path.join(fishDir, 'config.fish'))
  const names = [...CLAUDE_AUTH_VARS, ...CLAUDE_MODEL_VARS, ...CLAUDE_PROVIDER_SWITCHES]
  const re = new RegExp(`^\\s*(?:export\\s+|set\\s+-[a-zA-Z]+\\s+)?(${names.join('|')})(?:=|\\s+)(.*)$`)
  for (const file of unique(files)) {
    const lines = readText(file).split('\n')
    let changed = 0
    const next = lines.map((line) => {
      const m = line.match(re)
      if (!m) return line
      if (CLAUDE_MODEL_VARS.includes(m[1])) {
        const value = m[2].trim().replace(/^["']|["']$/g, '')
        if (modelAvailable(value, modelIds)) return line
      }
      changed++
      return RC_COMMENT + line
    })
    if (!changed) continue
    backupFile(file)
    writeFileSafe(file, next.join('\n'))
    ok(`已注释 ${file} 里 ${changed} 行旧的 Claude 环境变量（DeepSeek 等旧配置）`)
  }
}

function cleanWindowsClaudeVars(modelIds) {
  const env = winUserEnv()
  const names = [...CLAUDE_AUTH_VARS, ...CLAUDE_MODEL_VARS, ...CLAUDE_PROVIDER_SWITCHES]
  const removed = []
  for (const [name, value] of Object.entries(env)) {
    const upper = name.toUpperCase()
    if (!names.includes(upper)) continue
    if (CLAUDE_MODEL_VARS.includes(upper) && modelAvailable(value, modelIds)) continue
    backupWinEnv(name, value)
    winSetUserEnv(name, null)
    removed.push(name)
  }
  if (removed.length) ok(`已移除旧的用户环境变量 ${removed.join('、')}（DeepSeek 等旧配置，已备份）`)
}

function modelAvailable(value, modelIds) {
  const v = String(value || '').trim().replace(/\[1m\]$/i, '')
  if (!v) return false
  if (CLAUDE_MODEL_ALIASES.has(v.toLowerCase())) return true
  return modelIds.includes(v)
}

// ---------- finding / installing CLIs ----------

function which(cmd, extraDirs = []) {
  const exts = IS_WIN ? ['.exe', '.cmd', '.bat', ''] : ['']
  const dirs = unique([...extraDirs, ...(process.env.PATH || '').split(path.delimiter)])
  for (const dir of dirs) {
    for (const ext of exts) {
      const candidate = path.join(dir, cmd + ext)
      if (isFile(candidate)) return candidate
    }
  }
  return null
}

function commandVersion(bin) {
  const r = IS_WIN && /\.(cmd|bat)$/i.test(bin) ? run(`"${bin}"`, ['--version'], { shell: true }) : run(bin, ['--version'])
  const m = `${r.stdout || ''} ${r.stderr || ''}`.match(/(\d+\.\d+\.\d+)/)
  return m ? m[1] : null
}

function npmCliPath() {
  const nodeBin = fs.realpathSync(process.execPath)
  const dir = path.dirname(nodeBin)
  const candidates = IS_WIN
    ? [path.join(dir, 'node_modules', 'npm', 'bin', 'npm-cli.js')]
    : [path.join(dir, '..', 'lib', 'node_modules', 'npm', 'bin', 'npm-cli.js'), path.join(dir, '..', 'libexec', 'lib', 'node_modules', 'npm', 'bin', 'npm-cli.js')]
  return candidates.find(isFile) || null
}

function npmInstall(pkg) {
  const cli = npmCliPath()
  for (const registry of NPM_REGISTRIES) {
    info(`从 ${registry} 安装 ${pkg} …`)
    const args = ['install', '-g', '--prefix', NPM_PREFIX, `${pkg}@latest`, '--registry', registry, '--no-fund', '--no-audit', '--loglevel', 'error']
    const r = cli
      ? spawnSync(process.execPath, [cli, ...args], { stdio: 'inherit', env: { ...process.env, npm_config_registry: registry } })
      : spawnSync(IS_WIN ? 'npm.cmd' : 'npm', args, { stdio: 'inherit', shell: IS_WIN, env: { ...process.env, npm_config_registry: registry } })
    if (r.status === 0) return true
    warn(`从 ${registry} 安装失败，换下一个镜像重试…`)
  }
  return false
}

const pathDirsToPersist = new Set()
const installFailures = []

function ensureCli({ name, pkg, extraDirs, minVersion }) {
  const found = which(name, extraDirs)
  const version = found ? commandVersion(found) : null
  if (found && (!minVersion || (version && compareVersions(version, minVersion) >= 0))) {
    ok(`已安装 ${name} ${version || ''}（${found}）`)
    const onPath = (process.env.PATH || '').split(path.delimiter).some((d) => path.resolve(d) === path.dirname(found))
    if (found.startsWith(NPM_BIN) || !onPath) pathDirsToPersist.add(path.dirname(found))
    return found
  }
  if (truthy(process.env.HIVEGPT_NO_INSTALL)) {
    warn(`没有找到 ${name}，按 --no-install 跳过安装。`)
    return null
  }
  if (found) info(`${name} ${version || ''} 版本过旧，另装一份新版到 ${NPM_PREFIX}（不动原来那份）`)
  if (!npmInstall(pkg)) {
    console.log(`${red('✗')} ${name} 安装失败（配置已照常写好）。可以稍后手动运行：npm install -g ${pkg} --registry=https://registry.npmmirror.com`)
    installFailures.push(name)
    return null
  }
  pathDirsToPersist.add(NPM_BIN)
  const installed = which(name, [NPM_BIN])
  ok(`已安装 ${name} ${installed ? commandVersion(installed) || '' : ''}`)
  return installed
}

function nodeBinDirToPersist() {
  const dir = process.env.HIVEGPT_NODE_BIN || path.dirname(process.execPath)
  return dir.startsWith(HIVE_DIR) ? dir : null
}

// ---------- Git Bash for Claude Code on Windows ----------

function findGitBash() {
  const candidates = [
    process.env.CLAUDE_CODE_GIT_BASH_PATH,
    path.join(process.env.ProgramFiles || 'C:\\Program Files', 'Git', 'bin', 'bash.exe'),
    path.join(process.env['ProgramFiles(x86)'] || 'C:\\Program Files (x86)', 'Git', 'bin', 'bash.exe'),
    path.join(process.env.LOCALAPPDATA || '', 'Programs', 'Git', 'bin', 'bash.exe'),
    path.join(HIVE_DIR, 'git', 'bin', 'bash.exe')
  ]
  const git = which('git')
  if (git) candidates.push(path.join(path.dirname(git), '..', 'bin', 'bash.exe'))
  return candidates.filter(Boolean).map((p) => path.resolve(p)).find(isFile) || null
}

async function installPortableGit() {
  const listing = 'https://registry.npmmirror.com/-/binary/git-for-windows/'
  const versions = await (await httpGet(listing)).json()
  const release = versions
    .map((v) => v.name)
    .filter((n) => /^v\d+\.\d+\.\d+\.windows\.\d+\/$/.test(n))
    .sort((a, b) => compareVersions(a.match(/\d+/g).join('.'), b.match(/\d+/g).join('.')))
    .pop()
  if (!release) fail('没有在镜像上找到 Git for Windows。')
  const files = await (await httpGet(listing + release)).json()
  const arch = process.arch === 'arm64' ? 'arm64' : '64-bit'
  const file = files.find((f) => new RegExp(`^PortableGit-.*-${arch}\\.7z\\.exe$`).test(f.name))
  if (!file) fail('没有在镜像上找到 PortableGit。')
  const exe = path.join(os.tmpdir(), file.name)
  info(`Claude Code 在 Windows 上需要 Git Bash，正在从 npmmirror 下载 ${file.name}（约 60 MB）…`)
  await download(file.url || listing + release + file.name, exe, 'PortableGit')
  const target = path.join(HIVE_DIR, 'git')
  fs.rmSync(target, { recursive: true, force: true })
  const r = run(exe, [`-o${target}`, '-y'])
  fs.rmSync(exe, { force: true })
  const bash = path.join(target, 'bin', 'bash.exe')
  if (r.status !== 0 || !isFile(bash)) fail('解压 PortableGit 失败。可以手动安装 Git for Windows 后重新运行。')
  ok(`已安装 Git Bash（${bash}）`)
  return bash
}

// ---------- Claude Code ----------

async function setupClaudeCode(key, modelIds) {
  step('配置 Claude Code 命令行')
  const claudeExtra = [NPM_BIN, path.join(HOME, '.local', 'bin'), path.join(HOME, '.claude', 'local'), '/opt/homebrew/bin', '/usr/local/bin']
  ensureCli({ name: 'claude', pkg: '@anthropic-ai/claude-code', extraDirs: claudeExtra })

  let gitBash = null
  if (IS_WIN) {
    gitBash = findGitBash()
    if (!gitBash && !truthy(process.env.HIVEGPT_NO_INSTALL)) {
      try {
        gitBash = await installPortableGit()
      } catch (error) {
        warn(`${error.message} Claude Code 可能无法启动，请安装 Git for Windows（https://registry.npmmirror.com/binary.html?path=git-for-windows/）。`)
      }
    }
  }

  const configDir = process.env.CLAUDE_CONFIG_DIR || path.join(HOME, '.claude')
  const settingsPath = path.join(configDir, 'settings.json')
  const settings = readJson(settingsPath, {})
  if (typeof settings !== 'object' || Array.isArray(settings)) fail(`${settingsPath} 的内容不是 JSON 对象，没有改动它。`)
  const env = settings.env && typeof settings.env === 'object' ? settings.env : {}
  const notes = []
  if (env.ANTHROPIC_BASE_URL && normalizeBase(env.ANTHROPIC_BASE_URL) !== BASE) notes.push(`原地址 ${env.ANTHROPIC_BASE_URL}`)
  delete env.ANTHROPIC_API_KEY
  for (const name of CLAUDE_PROVIDER_SWITCHES) delete env[name]
  for (const name of CLAUDE_MODEL_VARS) {
    if (env[name] && !modelAvailable(env[name], modelIds)) {
      notes.push(`${name}=${env[name]}`)
      delete env[name]
    }
  }
  env.ANTHROPIC_BASE_URL = BASE
  env.ANTHROPIC_AUTH_TOKEN = key
  env.CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC = env.CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC || '1'
  if (gitBash) env.CLAUDE_CODE_GIT_BASH_PATH = gitBash
  settings.env = env
  if (!settings.$schema) settings.$schema = 'https://json.schemastore.org/claude-code-settings.json'
  if (settings.apiKeyHelper) {
    notes.push('apiKeyHelper')
    delete settings.apiKeyHelper
  }
  if (typeof settings.model === 'string' && !modelAvailable(settings.model, modelIds)) {
    notes.push(`model=${settings.model}`)
    delete settings.model
  }
  backupFile(settingsPath)
  writeJson(settingsPath, settings, 0o600)
  ok(`已写入 ${settingsPath}${notes.length ? `（替换了旧配置：${notes.join('，')}）` : ''}`)

  // ~/.claude.json: skip the first-run login wizard; drop a /login-managed Console key that would
  // conflict with ANTHROPIC_AUTH_TOKEN. A claude.ai login (oauthAccount) is left alone.
  const statePath = process.env.CLAUDE_CONFIG_DIR ? path.join(process.env.CLAUDE_CONFIG_DIR, '.claude.json') : path.join(HOME, '.claude.json')
  const state = readJson(statePath, {})
  if (state && typeof state === 'object' && !Array.isArray(state)) {
    const touch = ['hasCompletedOnboarding', 'primaryApiKey'].filter((k) => k === 'hasCompletedOnboarding' ? state[k] !== true : k in state)
    if (touch.length) {
      backupJsonKeys(statePath, state, touch)
      state.hasCompletedOnboarding = true
      delete state.primaryApiKey
      writeJson(statePath, state)
    }
    if (state.oauthAccount) {
      info('检测到你登录过 claude.ai 账号：登录保留不动，Claude Code 会优先使用 HiveGPT 的 Key。')
      info('启动时如果提示 Auth conflict 可以忽略；想彻底只用 HiveGPT，在 Claude Code 里输入 /logout。')
    }
  }

  cleanShellClaudeVars(modelIds)
  // settings.json env wins over the shell for the variables it sets (base URL, token); stale
  // ones it doesn't set (a DeepSeek model name, an API key, a Bedrock switch) still leak in.
  const leftover = ['ANTHROPIC_API_KEY', ...CLAUDE_PROVIDER_SWITCHES, ...CLAUDE_MODEL_VARS]
    .filter((n) => process.env[n] && !(CLAUDE_MODEL_VARS.includes(n) && modelAvailable(process.env[n], modelIds)))
  if (leftover.length) warn(`当前终端里还带着旧的 ${leftover.join('、')}，请新开一个终端再运行 claude，否则会用到旧的模型或 Key。`)
  if (exists(path.join(HOME, '.cc-switch'))) {
    warn('检测到 CC Switch：它切换供应商时会改写 ~/.claude/settings.json。建议在「使用密钥」弹窗里一键导入 HiveGPT 到 CC Switch，并在 CC Switch 里选中它。')
  }
}

// ---------- Claude desktop app (third-party inference / gateway mode) ----------

function desktopConfigDirs() {
  if (IS_MAC) return [path.join(HOME, 'Library', 'Application Support', 'Claude-3p')]
  if (!IS_WIN) return [path.join(process.env.XDG_CONFIG_HOME || path.join(HOME, '.config'), 'Claude-3p')]
  const local = process.env.LOCALAPPDATA || path.join(HOME, 'AppData', 'Local')
  const roaming = process.env.APPDATA || path.join(HOME, 'AppData', 'Roaming')
  // Docs name %LOCALAPPDATA%; some builds use %APPDATA%. The MSIX build may keep a private copy.
  const dirs = [path.join(local, 'Claude-3p'), path.join(roaming, 'Claude-3p')]
  const packages = path.join(local, 'Packages')
  if (exists(packages)) {
    for (const name of fs.readdirSync(packages)) {
      if (!/claude/i.test(name)) continue
      for (const sub of [['LocalCache', 'Roaming', 'Claude-3p'], ['LocalCache', 'Local', 'Claude-3p']]) {
        const p = path.join(packages, name, ...sub)
        if (exists(p)) dirs.push(p)
      }
    }
  }
  return dirs
}

function desktopInstalled() {
  if (IS_MAC) return ['/Applications/Claude.app', path.join(HOME, 'Applications', 'Claude.app')].some(exists)
  if (IS_WIN) {
    const local = process.env.LOCALAPPDATA || ''
    const packages = path.join(local, 'Packages')
    return exists(path.join(local, 'AnthropicClaude')) || exists(path.join(process.env.APPDATA || '', 'Claude')) ||
      (exists(packages) && fs.readdirSync(packages).some((n) => /claude/i.test(n)))
  }
  return exists(path.join(process.env.XDG_CONFIG_HOME || path.join(HOME, '.config'), 'Claude')) || Boolean(which('claude-desktop'))
}

function setupClaudeDesktop(key) {
  step('配置 Claude 桌面 App（第三方网关模式）')
  const installed = desktopInstalled()
  for (const dir of desktopConfigDirs()) {
    const lib = path.join(dir, 'configLibrary')
    const metaPath = path.join(lib, '_meta.json')
    const meta = readJson(metaPath, { appliedId: null, entries: [] })
    if (!Array.isArray(meta.entries)) meta.entries = []
    const previous = meta.entries.find((e) => e && e.id === meta.appliedId)
    let entry = meta.entries.find((e) => e && String(e.name).toLowerCase() === DESKTOP_PROFILE_NAME.toLowerCase())
    if (!entry) {
      entry = { id: crypto.randomUUID(), name: DESKTOP_PROFILE_NAME }
      meta.entries.push(entry)
    }
    const profilePath = path.join(lib, `${entry.id}.json`)
    const profile = readJson(profilePath, {})
    Object.assign(profile, {
      inferenceProvider: 'gateway',
      inferenceGatewayBaseUrl: BASE,
      inferenceGatewayApiKey: key,
      inferenceGatewayAuthScheme: 'bearer'
    })
    meta.appliedId = entry.id
    backupFile(profilePath)
    backupFile(metaPath)
    writeJson(profilePath, profile, 0o600)
    writeJson(metaPath, meta)
    ok(`已写入网关配置「${DESKTOP_PROFILE_NAME}」并设为当前配置（${lib}）`)
    if (previous && previous.id !== entry.id) info(`原来启用的配置「${previous.name}」还在，可以在 App 的 Developer → Configure Third-Party Inference 里切回。`)
  }
  if (!installed) {
    warn('没有检测到 Claude 桌面 App。配置已经提前写好，装好 App 后打开就会使用 HiveGPT。')
    info('下载地址：https://claude.ai/download（需要能访问 claude.ai）')
    return
  }
  info('请完全退出 Claude 桌面 App（macOS 按 ⌘Q，Windows 在托盘图标右键退出）后重新打开，模型会自动出现在模型列表里。')
  info('如果 App 之前登录过 claude.ai 账号：登录不会被删除。打开 App 后在登录页选择第三方/组织配置，或在账户菜单里切换到「HiveGPT」配置即可。')
}

// ---------- Codex ----------

function updateCodexToml(text, topLines, providerBlock) {
  const lines = text.replace(/\r\n/g, '\n').split('\n')
  const isHeader = (l) => /^\s*\[/.test(l)
  let firstTable = lines.findIndex(isHeader)
  if (firstTable < 0) firstTable = lines.length
  const ours = new Set(['model_provider', 'model', 'review_model', 'model_reasoning_effort', 'model_catalog_json'])
  const top = []
  for (const line of lines.slice(0, firstTable)) {
    if (/^# HiveGPT /.test(line)) continue
    const m = line.match(/^\s*([A-Za-z0-9_-]+)\s*=/)
    if (m && ours.has(m[1])) continue
    // A top-level `profile = …` (old [profiles.*] style) would override the provider, and newer Codex rejects it.
    if (m && m[1] === 'profile') {
      top.push(`# [HiveGPT 已注释] ${line}`)
      continue
    }
    top.push(line)
  }
  while (top.length && !top[0].trim()) top.shift()

  const tables = []
  let skipping = false
  let featuresSeen = false
  let inFeatures = false
  let goalsSet = false
  const addGoals = () => {
    let at = tables.length
    while (at > 0 && !tables[at - 1].trim()) at--
    tables.splice(at, 0, 'goals = true')
  }
  for (const line of lines.slice(firstTable)) {
    if (isHeader(line)) {
      if (inFeatures && !goalsSet) addGoals()
      skipping = /^\s*\[\s*model_providers\s*\.\s*"?OpenAI"?\s*[\].]/.test(line)
      inFeatures = /^\s*\[\s*features\s*\]/.test(line)
      if (inFeatures) featuresSeen = true
      goalsSet = false
    }
    if (skipping) continue
    if (inFeatures && /^\s*goals\s*=/.test(line)) goalsSet = true
    tables.push(line)
  }
  if (inFeatures && !goalsSet) addGoals()
  while (tables.length && !tables[tables.length - 1].trim()) tables.pop()

  const out = ['# HiveGPT 一键配置写入；恢复原配置：重新运行安装命令并加 --restore', ...topLines, '']
  if (top.length) out.push(...top, '')
  if (tables.length) out.push(...tables, '')
  out.push(...providerBlock)
  if (!featuresSeen) out.push('', '[features]', 'goals = true')
  return `${out.join('\n').replace(/\n{3,}/g, '\n\n')}\n`
}

function pickCodexModel(catalog, modelIds) {
  const slugs = (catalog?.models || []).map((m) => (typeof m?.slug === 'string' ? m.slug.trim() : '')).filter(Boolean)
  const pool = slugs.length ? slugs : modelIds
  if (pool.includes('gpt-5.5')) return 'gpt-5.5'
  return pool.find((id) => /^gpt-/i.test(id)) || pool[0] || 'gpt-5.5'
}

function codexReasoningEffort(catalog, slug) {
  const model = (catalog?.models || []).find((m) => m?.slug === slug)
  if (!model) return null
  const efforts = (model.supported_reasoning_levels || []).map((l) => (typeof l?.effort === 'string' ? l.effort.trim() : '')).filter(Boolean)
  if (!efforts.length) return null
  const def = typeof model.default_reasoning_level === 'string' ? model.default_reasoning_level.trim() : ''
  if (def && efforts.includes(def)) return def === 'none' ? null : def
  return efforts.find((e) => e !== 'none') || null
}

async function setupCodex(key, modelIds) {
  step('配置 Codex 命令行 / Codex App')
  ensureCli({ name: 'codex', pkg: '@openai/codex', extraDirs: [NPM_BIN, '/opt/homebrew/bin', '/usr/local/bin'], minVersion: MIN_CODEX_VERSION })

  const codexHome = process.env.CODEX_HOME || path.join(HOME, '.codex')
  const configPath = path.join(codexHome, 'config.toml')
  const catalog = await fetchCodexCatalog(key)
  const model = pickCodexModel(catalog, modelIds)
  const topLines = ['model_provider = "OpenAI"', `model = ${tomlString(model)}`, `review_model = ${tomlString(model)}`]
  const effort = codexReasoningEffort(catalog, model)
  if (effort) topLines.push(`model_reasoning_effort = ${tomlString(effort)}`)
  if (catalog) {
    const catalogPath = path.join(codexHome, 'codex-models.json')
    backupFile(catalogPath)
    writeJson(catalogPath, catalog)
    topLines.push(`model_catalog_json = ${tomlString(catalogPath)}`)
    ok(`已下载模型目录（${catalog.models.length} 个模型）到 ${catalogPath}`)
  }
  const existing = exists(configPath) ? readText(configPath) : ''
  if (!/^\s*disable_response_storage\s*=/m.test(existing)) topLines.push('disable_response_storage = true')
  if (!/^\s*network_access\s*=/m.test(existing)) topLines.push('network_access = "enabled"')
  if (IS_WIN && !/^\s*windows_wsl_setup_acknowledged\s*=/m.test(existing)) topLines.push('windows_wsl_setup_acknowledged = true')
  const providerBlock = [
    '[model_providers.OpenAI]',
    'name = "OpenAI"',
    `base_url = ${tomlString(`${BASE}/v1`)}`,
    'wire_api = "responses"',
    'requires_openai_auth = false',
    `experimental_bearer_token = ${tomlString(key)}`,
    'http_headers = { "x-openai-actor-authorization" = "local-image-extension" }'
  ]
  const previousProvider = existing.match(/^\s*model_provider\s*=\s*"([^"]+)"/m)?.[1]
  backupFile(configPath)
  writeFileSafe(configPath, updateCodexToml(existing, topLines, providerBlock), 0o600)
  ok(`已写入 ${configPath}（模型 ${model}${previousProvider && previousProvider !== 'OpenAI' ? `，原来用的是 ${previousProvider}，那段配置保留未删` : ''}）`)
  if (isFile(path.join(codexHome, 'auth.json'))) {
    info('auth.json（ChatGPT 账号登录 / 旧 Key）保留未动：现在走 HiveGPT，想切回 ChatGPT 账号运行 --restore 即可。')
  }
  info('如果 Codex App 或 IDE 扩展正开着，请完全退出后重新打开，再新建一个任务，配置才会生效。')
}

// ---------- main ----------

function parseArgs(argv) {
  const opts = { tools: process.env.HIVEGPT_TOOLS || '', restore: truthy(process.env.HIVEGPT_RESTORE) || /^\d{4}-/.test(process.env.HIVEGPT_RESTORE || '') }
  for (const arg of argv) {
    if (arg === '--restore') opts.restore = true
    else if (arg === '--yes' || arg === '-y') process.env.HIVEGPT_YES = '1'
    else if (arg === '--no-install') process.env.HIVEGPT_NO_INSTALL = '1'
    else if (arg.startsWith('--tools=')) opts.tools = arg.slice(8)
    else if (arg.startsWith('--key=')) process.env.HIVEGPT_KEY = arg.slice(6)
    else if (arg === '--help' || arg === '-h') opts.help = true
  }
  return opts
}

function printHelp() {
  console.log(`HiveGPT 一键安装与配置

  --tools=claude|codex|all   只配置某一类工具（默认按 Key 所在分组自动推荐）
  --yes                      不提问，全部用推荐选项
  --no-install               只写配置，不安装 Claude Code / Codex
  --restore                  恢复到最近一次运行之前的配置

环境变量：HIVEGPT_KEY、HIVEGPT_BASE_URL（默认 https://hivegpt.cn）、HIVEGPT_NPM_REGISTRY`)
}

async function main() {
  const opts = parseArgs(process.argv.slice(2))
  if (opts.help) return printHelp()
  console.log(bold(`\nHiveGPT 一键安装与配置  ${dim(BASE)}`))
  if (opts.restore) return restore()

  let key = (process.env.HIVEGPT_KEY || '').trim().replace(/^["']|["']$/g, '')
  // A key that came with the command (the site's copy button or the Windows installer) means
  // "just do it": no questions, the recommended tools are set up.
  const keyFromCommand = Boolean(key)
  if (!key) {
    if (!process.stdin.isTTY) fail('没有拿到 Key。请设置 HIVEGPT_KEY 环境变量后重新运行（在「API 密钥」页点「使用密钥」可复制带 Key 的命令）。')
    key = (await readLine(`请粘贴你的 HiveGPT Key（在 ${BASE}/keys 复制，输入时显示为 *）: `, { secret: true })).trim()
  }
  if (!key) fail('Key 不能为空。')
  if (/\s/.test(key)) fail('Key 里有空格或换行，请重新复制完整的 Key。')

  step('验证 Key')
  const modelIds = await fetchModels(key)
  ok(`Key ${maskKey(key)} 可用，分组里有 ${modelIds.length} 个模型`)
  const hasClaude = modelIds.some((id) => /^claude-/i.test(id))
  const hasGpt = modelIds.some((id) => /^(gpt-|codex-|o\d)/i.test(id))
  const recommended = hasClaude && !hasGpt ? 'claude' : hasGpt && !hasClaude ? 'codex' : 'all'

  let tools = String(opts.tools || '').toLowerCase()
  if (!['claude', 'codex', 'all'].includes(tools) && keyFromCommand) {
    tools = recommended
    info(`将配置：${{ claude: 'Claude Code + Claude 桌面 App', codex: 'Codex 命令行 + Codex App', all: 'Claude Code、Claude 桌面 App、Codex 命令行和 Codex App' }[tools]}${tools === 'all' ? '（只想配其中一种，可以加 --tools=claude 或 --tools=codex 重新运行）' : ''}`)
  }
  if (!['claude', 'codex', 'all'].includes(tools)) {
    const labels = { claude: '1', codex: '2', all: '3' }
    console.log(`\n要配置哪些工具？
  1) Claude Code 命令行 + Claude 桌面 App
  2) Codex 命令行 + Codex App
  3) 全部${hasClaude || hasGpt ? dim(`\n  （这个 Key 的分组${hasClaude ? '有 Claude 模型' : ''}${hasClaude && hasGpt ? '，也' : ''}${hasGpt ? '有 GPT 模型' : ''}，推荐 ${labels[recommended]}）`) : ''}`)
    const answer = await ask('请选择', labels[recommended])
    tools = { 1: 'claude', 2: 'codex', 3: 'all' }[answer.trim()] || recommended
  }

  const failures = []
  const attempt = async (label, fn) => {
    try {
      await fn()
    } catch (error) {
      failures.push(label)
      console.log(`${red('✗')} ${label}：${error instanceof SetupError ? error.message : error.stack || error}`)
    }
  }

  if (tools === 'claude' || tools === 'all') {
    await attempt('Claude Code', () => setupClaudeCode(key, modelIds))
    if (hasClaude) await attempt('Claude 桌面 App', () => setupClaudeDesktop(key))
    else info('这个 Key 的分组没有 Claude 模型，Claude 桌面 App 用不了它，已跳过（Claude Code 可以正常使用）。')
  }
  if (tools === 'codex' || tools === 'all') {
    if (hasClaude && !hasGpt) warn('这个 Key 是 Claude 分组：Claude 分组一般只允许 Claude Code 和 Claude 桌面 App 使用，Codex 请换一个 GPT 分组的 Key 再配置（运行时加 --tools=codex）。')
    await attempt('Codex', () => setupCodex(key, modelIds))
  }

  const pathDirs = [...pathDirsToPersist]
  const nodeDir = nodeBinDirToPersist()
  if (pathDirs.length && nodeDir) pathDirs.push(nodeDir)
  if (pathDirs.length) await attempt('PATH', () => persistPath(unique(pathDirs)))

  console.log('')
  if (manifest.entries.length) info(`原配置已备份到 ${backupDir}`)
  if (failures.length || installFailures.length) {
    if (failures.length) console.log(`${yellow('!')} ${failures.join('、')} 没有配置成功，见上面的提示。其它部分已经配好。`)
    if (installFailures.length) console.log(`${yellow('!')} ${installFailures.join('、')} 没有装上，见上面的提示；装好后不用重新配置。`)
    process.exitCode = 1
  } else {
    ok(bold('全部完成！'))
  }
  console.log(`
下一步：
  ${bold('用桌面 App')}：完全退出 Claude / Codex App 再重新打开，就会用 HiveGPT 了。
  ${bold('用命令行')}：
  1. ${bold('新开一个终端窗口')}（让新的 PATH 和配置生效）
  ${tools !== 'codex' ? `2. 进入项目文件夹，运行 ${cyan('claude')}` : `2. 进入项目文件夹，运行 ${cyan('codex')}`}${tools === 'all' ? `，或 ${cyan('codex')}` : ''}
  3. 想恢复原来的配置（DeepSeek、官方账号等）：重新运行安装命令并加 ${cyan('--restore')}
`)
}

main().catch((error) => {
  console.log(`\n${red('✗')} ${error instanceof SetupError ? error.message : error.stack || error}`)
  if (manifest.entries.length) console.log(`  已改动的文件都备份在 ${backupDir}，可以加 --restore 恢复。`)
  process.exitCode = 1
})
