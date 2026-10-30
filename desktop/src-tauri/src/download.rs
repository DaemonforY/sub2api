//! Shared download helper: streams a URL to a file, hashing as it goes and emitting progress events,
//! and unpacks .tar.gz archives. Used for the Codex engine and the office kit's Python.

use std::path::Path;
use std::time::Instant;

use base64::Engine as _;
use futures_util::StreamExt;
use serde::Serialize;
use sha2::{Digest, Sha256, Sha512};
use tauri::{AppHandle, Emitter};
use tokio::io::AsyncWriteExt;

#[derive(Clone, Serialize)]
pub struct Progress {
    pub stage: &'static str,
    pub downloaded: u64,
    pub total: u64,
    pub source: String,
}

pub struct Digests {
    pub sha256_hex: String,
    pub sha512_b64: String,
}

pub fn client() -> Result<reqwest::Client, String> {
    reqwest::Client::builder()
        .user_agent(concat!("hivegpt-desktop/", env!("CARGO_PKG_VERSION")))
        .connect_timeout(std::time::Duration::from_secs(15))
        .build()
        .map_err(|e| e.to_string())
}

pub fn emit(app: &AppHandle, event: &str, stage: &'static str, downloaded: u64, total: u64, source: &str) {
    let _ = app.emit(event, Progress { stage, downloaded, total, source: source.to_string() });
}

/// Downloads `url` into `dest`, emitting `event` with stage "download" at most every 200 ms.
pub async fn fetch(
    app: &AppHandle,
    client: &reqwest::Client,
    url: &str,
    dest: &Path,
    size_hint: Option<u64>,
    event: &str,
    source: &str,
) -> Result<Digests, String> {
    let resp = client
        .get(url)
        .send()
        .await
        .and_then(|r| r.error_for_status())
        .map_err(|e| format!("下载失败：{e}"))?;
    let total = resp.content_length().or(size_hint).unwrap_or(0);
    let mut file = tokio::fs::File::create(dest).await.map_err(|e| format!("写入失败：{e}"))?;
    let mut h256 = Sha256::new();
    let mut h512 = Sha512::new();
    let mut downloaded = 0u64;
    let mut last = Instant::now();
    let mut stream = resp.bytes_stream();
    while let Some(chunk) = stream.next().await {
        let chunk = chunk.map_err(|e| format!("下载中断：{e}"))?;
        h256.update(&chunk);
        h512.update(&chunk);
        file.write_all(&chunk).await.map_err(|e| format!("写入失败：{e}"))?;
        downloaded += chunk.len() as u64;
        if last.elapsed().as_millis() >= 200 {
            last = Instant::now();
            emit(app, event, "download", downloaded, total, source);
        }
    }
    file.flush().await.map_err(|e| e.to_string())?;
    Ok(Digests {
        sha256_hex: format!("{:x}", h256.finalize()),
        sha512_b64: base64::engine::general_purpose::STANDARD.encode(h512.finalize()),
    })
}

/// Unpacks a .tar.gz into `dest` (replaced if it exists). Entries that would escape `dest` are refused.
pub fn unpack_tgz(tgz: &Path, dest: &Path) -> Result<(), String> {
    if dest.exists() {
        std::fs::remove_dir_all(dest).map_err(|e| e.to_string())?;
    }
    std::fs::create_dir_all(dest).map_err(|e| e.to_string())?;
    let f = std::fs::File::open(tgz).map_err(|e| e.to_string())?;
    let mut archive = tar::Archive::new(flate2::read::GzDecoder::new(f));
    archive.set_preserve_permissions(true);
    archive.unpack(dest).map_err(|e| format!("解压失败：{e}"))?;
    Ok(())
}
