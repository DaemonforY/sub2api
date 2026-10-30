//! 办公工具包: a private Python with the libraries Codex needs for Excel / Word / PowerPoint / PDF
//! work. Office PCs rarely have Python, and without it most office tasks fail.
//!
//! Python is python-build-standalone from the npmmirror mirror, pinned by sha256: hashes taken from
//! the release's SHA256SUMS and cross-checked against the upstream GitHub release. The libraries are
//! installed with pip from domestic PyPI mirrors. Everything lives under <app data>/office; the
//! user's own Python, if any, is untouched.

use std::path::PathBuf;
use std::process::Stdio;
use std::sync::OnceLock;

use serde::Serialize;
use tauri::{AppHandle, Manager};
use tokio::process::Command;
use tokio::sync::Mutex;

use crate::download;

const PY_RELEASE: &str = "20261020";
const PY_VERSION: &str = "3.12.15";
/// Bump when PACKAGES changes so existing installs add the new ones.
const KIT_VERSION: &str = "1";
const PROGRESS_EVENT: &str = "office-install-progress";

const PY_MIRROR: &str = "https://registry.npmmirror.com/-/binary/python-build-standalone";

pub const PIP_MIRRORS: &[&str] = &[
    "https://pypi.tuna.tsinghua.edu.cn/simple",
    "https://mirrors.aliyun.com/pypi/simple",
    "https://repo.huaweicloud.com/repository/pypi/simple",
];

/// What the task templates rely on: spreadsheets (old .xls too), Word, PowerPoint, reading and writing
/// PDF, images, and guessing the encoding of Chinese CSV files.
const PACKAGES: &[&str] = &["openpyxl", "xlrd", "pandas", "python-docx", "python-pptx", "pdfplumber", "fpdf2", "pillow", "chardet"];

fn python_asset() -> Result<(&'static str, &'static str), String> {
    // (target triple, sha256 of cpython-<ver>+<release>-<triple>-install_only.tar.gz)
    let a = match (std::env::consts::OS, std::env::consts::ARCH) {
        ("windows", "x86_64") => ("x86_64-pc-windows-msvc", "5e7d27ae6181fcf1f992c5a221b8fa7a320e7fd576c476434886292ccf2e20fb"),
        ("windows", "aarch64") => ("aarch64-pc-windows-msvc", "2b1b432b7276d997f938b616e99d418a532402c824bca7e6530dadc77b38cf3d"),
        ("macos", "aarch64") => ("aarch64-apple-darwin", "921fc822416bbad146b026d2051dbfe08338ce4144ced64c56afd899a4351752"),
        ("macos", "x86_64") => ("x86_64-apple-darwin", "f4aa54b04f24bb1aa597d60c077755c731c3090f822da167e1cef97207111ae9"),
        (os, arch) => return Err(format!("办公工具包暂不支持这个系统（{os} / {arch}）")),
    };
    Ok(a)
}

fn root(app: &AppHandle) -> Result<PathBuf, String> {
    let base = app.path().app_data_dir().map_err(|e| format!("找不到应用数据目录：{e}"))?;
    Ok(base.join("office").join(format!("py-{PY_VERSION}")))
}

fn python_home(app: &AppHandle) -> Result<PathBuf, String> {
    Ok(root(app)?.join("python"))
}

pub fn python_exe(app: &AppHandle) -> Result<PathBuf, String> {
    let home = python_home(app)?;
    Ok(if cfg!(windows) { home.join("python.exe") } else { home.join("bin").join("python3") })
}

/// The command name Codex should type: Windows has python.exe, macOS python3.
pub fn python_command() -> &'static str {
    if cfg!(windows) {
        "python"
    } else {
        "python3"
    }
}

fn marker(app: &AppHandle) -> Result<PathBuf, String> {
    Ok(root(app)?.join("kit-ready"))
}

pub fn is_ready(app: &AppHandle) -> bool {
    marker(app).ok().and_then(|m| std::fs::read_to_string(m).ok()).map(|v| v.trim() == KIT_VERSION).unwrap_or(false)
}

/// Directories to put in front of PATH so `python` / `pip` resolve to the kit.
pub fn path_dirs(app: &AppHandle) -> Vec<PathBuf> {
    let Ok(home) = python_home(app) else { return Vec::new() };
    if cfg!(windows) {
        vec![home.clone(), home.join("Scripts")]
    } else {
        vec![home.join("bin")]
    }
}

#[derive(Serialize)]
pub struct OfficeStatus {
    ready: bool,
    python: String,
    packages: Vec<&'static str>,
}

#[tauri::command]
pub fn office_status(app: AppHandle) -> Result<OfficeStatus, String> {
    Ok(OfficeStatus { ready: is_ready(&app), python: PY_VERSION.to_string(), packages: PACKAGES.to_vec() })
}

fn install_lock() -> &'static Mutex<()> {
    static LOCK: OnceLock<Mutex<()>> = OnceLock::new();
    LOCK.get_or_init(|| Mutex::new(()))
}

/// Downloads Python if needed, then pip-installs the packages. Emits `office-install-progress`
/// (download → unpack → packages → done). Safe to call twice; the second call waits for the first.
#[tauri::command]
pub async fn office_install(app: AppHandle) -> Result<OfficeStatus, String> {
    let _guard = install_lock().lock().await;
    if is_ready(&app) {
        return office_status(app);
    }
    let root = root(&app)?;
    tokio::fs::create_dir_all(&root).await.map_err(|e| format!("创建目录失败：{e}"))?;

    if !python_exe(&app)?.is_file() {
        let (triple, sha256) = python_asset()?;
        let name = format!("cpython-{PY_VERSION}+{PY_RELEASE}-{triple}-install_only.tar.gz");
        let url = format!("{PY_MIRROR}/{PY_RELEASE}/{}", name.replace('+', "%2B"));
        let tgz = root.join(format!("{name}.part"));
        let client = download::client()?;
        let digests = download::fetch(&app, &client, &url, &tgz, None, PROGRESS_EVENT, "npmmirror").await?;
        if digests.sha256_hex != sha256 {
            let _ = tokio::fs::remove_file(&tgz).await;
            return Err("Python 安装包校验不通过（可能被篡改或下载不完整），请重试".into());
        }
        download::emit(&app, PROGRESS_EVENT, "unpack", 0, 0, "npmmirror");
        // The archive's top-level folder is python/, so unpacking into root gives root/python.
        let staging = root.join("staging");
        let (tgz2, staging2) = (tgz.clone(), staging.clone());
        tokio::task::spawn_blocking(move || download::unpack_tgz(&tgz2, &staging2))
            .await
            .map_err(|e| e.to_string())??;
        let _ = tokio::fs::remove_file(&tgz).await;
        let home = python_home(&app)?;
        if home.exists() {
            let _ = tokio::fs::remove_dir_all(&home).await;
        }
        tokio::fs::rename(staging.join("python"), &home).await.map_err(|e| format!("安装 Python 失败：{e}"))?;
        let _ = tokio::fs::remove_dir_all(&staging).await;
    }

    download::emit(&app, PROGRESS_EVENT, "packages", 0, 0, "");
    let python = python_exe(&app)?;
    let mut errors = Vec::new();
    let mut installed = false;
    for mirror in PIP_MIRRORS {
        match pip_install(&python, mirror).await {
            Ok(()) => {
                installed = true;
                break;
            }
            Err(e) => errors.push(format!("{mirror}：{e}")),
        }
    }
    if !installed {
        return Err(format!("办公工具包安装失败，请检查网络后重试。\n{}", errors.join("\n")));
    }
    tokio::fs::write(marker(&app)?, KIT_VERSION).await.map_err(|e| e.to_string())?;
    download::emit(&app, PROGRESS_EVENT, "done", 0, 0, "");
    office_status(app)
}

async fn pip_install(python: &PathBuf, index: &str) -> Result<(), String> {
    let mut cmd = Command::new(python);
    cmd.args(["-m", "pip", "install", "--disable-pip-version-check", "--no-warn-script-location", "--only-binary=:all:", "--timeout", "30", "--index-url", index])
        .args(PACKAGES)
        .env("PYTHONUTF8", "1")
        .env_remove("PIP_INDEX_URL")
        .stdin(Stdio::null())
        .stdout(Stdio::null())
        .stderr(Stdio::piped())
        .kill_on_drop(true);
    #[cfg(windows)]
    {
        const CREATE_NO_WINDOW: u32 = 0x0800_0000;
        cmd.creation_flags(CREATE_NO_WINDOW);
    }
    let out = cmd.output().await.map_err(|e| format!("无法运行 pip：{e}"))?;
    if out.status.success() {
        return Ok(());
    }
    let err = String::from_utf8_lossy(&out.stderr);
    let tail: Vec<&str> = err.lines().rev().filter(|l| !l.trim().is_empty()).take(3).collect();
    Err(tail.into_iter().rev().collect::<Vec<_>>().join(" / "))
}
