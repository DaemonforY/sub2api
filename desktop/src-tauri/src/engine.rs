//! The Codex engine: a pinned `@openai/codex` platform build, downloaded on first run from
//! domestic npm mirrors (the installer stays small; hivegpt.cn's Seoul server is too slow for a
//! ~120 MB download), checked against the registry's sha512 integrity and unpacked into the app's
//! data directory. Nothing here touches the user's own `~/.codex` or any Codex they installed.

use std::path::{Path, PathBuf};
use std::time::Instant;

use base64::Engine as _;
use futures_util::StreamExt;
use serde::{Deserialize, Serialize};
use sha2::{Digest, Sha512};
use tauri::{AppHandle, Emitter, Manager};
use tokio::io::AsyncWriteExt;

/// Pinned engine version. The app-server protocol is experimental, so upgrades are deliberate
/// (later this comes from GET /api/v1/desktop/version together with per-platform hashes).
pub const CODEX_VERSION: &str = "0.142.3";

/// Mirrors tried in order. All serve the npm registry API, so `<mirror>/@openai%2fcodex/<ver>`
/// returns the version document with `dist.tarball` and `dist.integrity`.
const MIRRORS: &[&str] = &[
    "https://registry.npmmirror.com",
    "https://mirrors.cloud.tencent.com/npm",
    "https://repo.huaweicloud.com/repository/npm",
];

pub struct Platform {
    /// Suffix of the npm version, e.g. `0.142.3-win32-x64`.
    npm_suffix: &'static str,
    /// Rust target triple used as the directory name inside the package.
    triple: &'static str,
}

pub fn platform() -> Result<Platform, String> {
    let p = match (std::env::consts::OS, std::env::consts::ARCH) {
        ("windows", "x86_64") => Platform { npm_suffix: "win32-x64", triple: "x86_64-pc-windows-msvc" },
        ("windows", "aarch64") => Platform { npm_suffix: "win32-arm64", triple: "aarch64-pc-windows-msvc" },
        ("macos", "aarch64") => Platform { npm_suffix: "darwin-arm64", triple: "aarch64-apple-darwin" },
        ("macos", "x86_64") => Platform { npm_suffix: "darwin-x64", triple: "x86_64-apple-darwin" },
        ("linux", "x86_64") => Platform { npm_suffix: "linux-x64", triple: "x86_64-unknown-linux-musl" },
        ("linux", "aarch64") => Platform { npm_suffix: "linux-arm64", triple: "aarch64-unknown-linux-musl" },
        (os, arch) => return Err(format!("暂不支持这个系统（{os} / {arch}）")),
    };
    Ok(p)
}

pub fn engine_root(app: &AppHandle) -> Result<PathBuf, String> {
    let base = app.path().app_data_dir().map_err(|e| format!("找不到应用数据目录：{e}"))?;
    Ok(base.join("engine").join(CODEX_VERSION))
}

/// Directory of the unpacked package's vendor folder for this platform.
pub fn vendor_dir(app: &AppHandle) -> Result<PathBuf, String> {
    let p = platform()?;
    Ok(engine_root(app)?.join("package").join("vendor").join(p.triple))
}

pub fn binary_path(app: &AppHandle) -> Result<PathBuf, String> {
    let exe = if cfg!(windows) { "codex.exe" } else { "codex" };
    Ok(vendor_dir(app)?.join("bin").join(exe))
}

#[derive(Serialize)]
pub struct EngineStatus {
    ready: bool,
    version: &'static str,
    path: String,
}

#[tauri::command]
pub fn engine_status(app: AppHandle) -> Result<EngineStatus, String> {
    let bin = binary_path(&app)?;
    Ok(EngineStatus { ready: bin.is_file(), version: CODEX_VERSION, path: bin.display().to_string() })
}

#[derive(Deserialize)]
struct VersionDoc {
    dist: Dist,
}

#[derive(Deserialize)]
struct Dist {
    tarball: String,
    integrity: String,
    #[serde(default)]
    size: Option<u64>,
}

#[derive(Clone, Serialize)]
struct Progress {
    stage: &'static str,
    downloaded: u64,
    total: u64,
    mirror: String,
}

/// Downloads, verifies and unpacks the engine. Emits `engine-install-progress` while working.
#[tauri::command]
pub async fn engine_install(app: AppHandle) -> Result<EngineStatus, String> {
    if binary_path(&app)?.is_file() {
        return engine_status(app);
    }
    let plat = platform()?;
    let version = format!("{CODEX_VERSION}-{}", plat.npm_suffix);
    let client = reqwest::Client::builder()
        .user_agent(concat!("hivegpt-desktop/", env!("CARGO_PKG_VERSION")))
        .connect_timeout(std::time::Duration::from_secs(15))
        .build()
        .map_err(|e| e.to_string())?;

    let root = engine_root(&app)?;
    let parent = root.parent().ok_or("应用数据目录异常")?.to_path_buf();
    tokio::fs::create_dir_all(&parent).await.map_err(|e| format!("创建目录失败：{e}"))?;

    let mut errors = Vec::new();
    for mirror in MIRRORS {
        match install_from(&app, &client, mirror, &version, &parent, &root).await {
            Ok(()) => return engine_status(app),
            Err(e) => errors.push(format!("{mirror}：{e}")),
        }
    }
    Err(format!("引擎下载失败，请检查网络后重试。\n{}", errors.join("\n")))
}

async fn install_from(
    app: &AppHandle,
    client: &reqwest::Client,
    mirror: &str,
    version: &str,
    parent: &Path,
    root: &Path,
) -> Result<(), String> {
    let meta_url = format!("{mirror}/@openai%2fcodex/{version}");
    let doc: VersionDoc = client
        .get(&meta_url)
        .send()
        .await
        .and_then(|r| r.error_for_status())
        .map_err(|e| format!("读取版本信息失败：{e}"))?
        .json()
        .await
        .map_err(|e| format!("版本信息格式不对：{e}"))?;
    let expected = doc
        .dist
        .integrity
        .strip_prefix("sha512-")
        .ok_or("版本信息里没有 sha512 校验值")?
        .to_string();

    let resp = client
        .get(&doc.dist.tarball)
        .send()
        .await
        .and_then(|r| r.error_for_status())
        .map_err(|e| format!("下载失败：{e}"))?;
    let total = resp.content_length().or(doc.dist.size).unwrap_or(0);

    let tgz = parent.join(format!("{version}.tgz.part"));
    let mut file = tokio::fs::File::create(&tgz).await.map_err(|e| format!("写入失败：{e}"))?;
    let mut hasher = Sha512::new();
    let mut downloaded = 0u64;
    let mut last_emit = Instant::now();
    let mut stream = resp.bytes_stream();
    while let Some(chunk) = stream.next().await {
        let chunk = chunk.map_err(|e| format!("下载中断：{e}"))?;
        hasher.update(&chunk);
        file.write_all(&chunk).await.map_err(|e| format!("写入失败：{e}"))?;
        downloaded += chunk.len() as u64;
        if last_emit.elapsed().as_millis() >= 200 {
            last_emit = Instant::now();
            let _ = app.emit(
                "engine-install-progress",
                Progress { stage: "download", downloaded, total, mirror: mirror.to_string() },
            );
        }
    }
    file.flush().await.map_err(|e| e.to_string())?;
    drop(file);

    let actual = base64::engine::general_purpose::STANDARD.encode(hasher.finalize());
    if actual != expected {
        let _ = tokio::fs::remove_file(&tgz).await;
        return Err("文件校验不通过（可能被篡改或下载不完整）".into());
    }

    let _ = app.emit(
        "engine-install-progress",
        Progress { stage: "unpack", downloaded, total, mirror: mirror.to_string() },
    );
    let staging = parent.join(format!("{version}.staging"));
    let tgz_for_unpack = tgz.clone();
    let staging_for_unpack = staging.clone();
    tokio::task::spawn_blocking(move || unpack(&tgz_for_unpack, &staging_for_unpack))
        .await
        .map_err(|e| e.to_string())??;
    let _ = tokio::fs::remove_file(&tgz).await;

    if root.exists() {
        let _ = tokio::fs::remove_dir_all(root).await;
    }
    tokio::fs::rename(&staging, root).await.map_err(|e| format!("安装失败：{e}"))?;
    let _ = app.emit("engine-install-progress", Progress { stage: "done", downloaded, total, mirror: mirror.to_string() });
    Ok(())
}

fn unpack(tgz: &Path, dest: &Path) -> Result<(), String> {
    if dest.exists() {
        std::fs::remove_dir_all(dest).map_err(|e| e.to_string())?;
    }
    std::fs::create_dir_all(dest).map_err(|e| e.to_string())?;
    let f = std::fs::File::open(tgz).map_err(|e| e.to_string())?;
    let mut archive = tar::Archive::new(flate2::read::GzDecoder::new(f));
    archive.set_preserve_permissions(true);
    // `unpack` refuses entries that would escape `dest`.
    archive.unpack(dest).map_err(|e| format!("解压失败：{e}"))?;
    Ok(())
}
