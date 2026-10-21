# HiveGPT 一键安装与配置（Windows）：Claude Code、Claude 桌面 App、Codex CLI、Codex App
#
# 在 PowerShell 里运行：
#   irm https://hivegpt.cn/install.ps1 | iex
#   $env:HIVEGPT_KEY='sk-xxx'; irm https://hivegpt.cn/install.ps1 | iex
#   $env:HIVEGPT_RESTORE='1'; irm https://hivegpt.cn/install.ps1 | iex     # 恢复原配置
#   $env:HIVEGPT_TOOLS='codex'; irm https://hivegpt.cn/install.ps1 | iex   # 只配 Codex
#
# 这个脚本只负责准备 Node.js（没有 22+ 版本就从国内镜像下载免安装版到 %USERPROFILE%\.hivegpt\node，
# 不需要管理员权限），然后运行 hivegpt-setup.mjs 完成安装和配置。

& {
  $ErrorActionPreference = 'Stop'
  $ProgressPreference = 'SilentlyContinue'
  try {
    [Net.ServicePointManager]::SecurityProtocol = [Net.ServicePointManager]::SecurityProtocol -bor [Net.SecurityProtocolType]::Tls12
  } catch {}

  $base = if ($env:HIVEGPT_BASE_URL) { $env:HIVEGPT_BASE_URL.Trim().TrimEnd('/') } else { 'https://hivegpt.cn' }
  if ($base -match '/v1$') { $base = $base.Substring(0, $base.Length - 3) }
  $hiveDir = if ($env:HIVEGPT_HOME) { $env:HIVEGPT_HOME } else { Join-Path $env:USERPROFILE '.hivegpt' }

  function Say($msg) { Write-Host "> $msg" -ForegroundColor Cyan }
  function Fail($msg) { Write-Host "x $msg" -ForegroundColor Red }

  function Test-NodeOk($exe) {
    if (-not $exe -or -not (Test-Path $exe)) { return $false }
    try {
      & $exe -e "process.exit(+process.versions.node.split('.')[0] >= 22 ? 0 : 1)" 2>$null | Out-Null
      return $LASTEXITCODE -eq 0
    } catch { return $false }
  }

  $is64 = [Environment]::Is64BitOperatingSystem
  $arch = if ($env:PROCESSOR_ARCHITECTURE -eq 'ARM64' -or $env:PROCESSOR_ARCHITEW6432 -eq 'ARM64') { 'arm64' } else { 'x64' }
  if (-not $is64) { Fail '不支持 32 位 Windows，Claude Code 和 Codex 都需要 64 位系统。'; return }

  $node = $null
  $sys = Get-Command node -ErrorAction SilentlyContinue | Select-Object -First 1
  $ours = Join-Path $hiveDir 'node\node.exe'
  if ($sys -and (Test-NodeOk $sys.Source)) { $node = $sys.Source }
  elseif (Test-NodeOk $ours) { $node = $ours }

  $tmp = Join-Path ([IO.Path]::GetTempPath()) ("hivegpt-" + [Guid]::NewGuid().ToString('N'))
  New-Item -ItemType Directory -Path $tmp -Force | Out-Null
  try {
    if (-not $node) {
      Say "没有找到 Node.js 22+，从国内镜像下载免安装版到 $hiveDir\node ..."
      $mirrors = @('https://cdn.npmmirror.com/binaries/node', 'https://mirrors.cloud.tencent.com/nodejs-release', 'https://mirrors.huaweicloud.com/nodejs')
      foreach ($mirror in $mirrors) {
        try {
          $index = Invoke-RestMethod -Uri "$mirror/index.json" -TimeoutSec 20 -UseBasicParsing
          $version = ($index | Where-Object { $_.lts } | Select-Object -First 1).version
          if (-not $version) { continue }
          $name = "node-$version-win-$arch"
          Say "下载 Node.js $version（$mirror）..."
          $zip = Join-Path $tmp 'node.zip'
          Invoke-WebRequest -Uri "$mirror/$version/$name.zip" -OutFile $zip -UseBasicParsing -TimeoutSec 600
          Add-Type -AssemblyName System.IO.Compression.FileSystem
          $unzipped = Join-Path $tmp ([Guid]::NewGuid().ToString('N'))
          [IO.Compression.ZipFile]::ExtractToDirectory($zip, $unzipped)
          New-Item -ItemType Directory -Path $hiveDir -Force | Out-Null
          $target = Join-Path $hiveDir 'node'
          if (Test-Path $target) { Remove-Item $target -Recurse -Force }
          Move-Item (Join-Path $unzipped $name) $target
          $node = $ours
          break
        } catch {
          Say "从 $mirror 下载失败，换下一个镜像...（$($_.Exception.Message)）"
        }
      }
      if (-not (Test-NodeOk $node)) { Fail 'Node.js 下载失败，请检查网络后重试（npmmirror / 腾讯云 / 华为云镜像都连不上）。'; return }
    }

    $setup = Join-Path $tmp 'hivegpt-setup.mjs'
    try {
      Invoke-WebRequest -Uri "$base/hivegpt-setup.mjs" -OutFile $setup -UseBasicParsing -TimeoutSec 60
    } catch {
      Fail "下载配置脚本失败：$base/hivegpt-setup.mjs（$($_.Exception.Message)），请检查网络后重试。"
      return
    }

    $env:HIVEGPT_BASE_URL = $base
    $env:HIVEGPT_NODE_BIN = Split-Path $node -Parent
    & $node $setup
    # 让当前窗口也能直接用刚装好的命令
    $env:Path = [Environment]::GetEnvironmentVariable('Path', 'Machine') + ';' + [Environment]::GetEnvironmentVariable('Path', 'User')
  } finally {
    Remove-Item $tmp -Recurse -Force -ErrorAction SilentlyContinue
    Remove-Item Env:HIVEGPT_KEY -ErrorAction SilentlyContinue
    Remove-Item Env:HIVEGPT_RESTORE -ErrorAction SilentlyContinue
  }
}
