//! Where the app talks to and the device key. The key lives in the system credential store
//! (Windows Credential Manager / macOS Keychain), never in a plain file.
//!
//! The key normally comes from the WeChat sign-in (login.rs). In debug builds developers can paste one
//! on the setup screen or set HIVEGPT_API_KEY / HIVEGPT_BASE_URL in the environment. Release builds
//! ignore both variables, so something that only changes the user's environment can't point the app
//! at another server (which would then serve it skills) or swap its key.

use std::path::PathBuf;

use tauri::AppHandle;

use crate::login;

const KEYRING_SERVICE: &str = "cn.hivegpt.desktop";
const KEYRING_USER: &str = "device-key";
const DEFAULT_BASE_URL: &str = "https://hivegpt.cn";

/// A development override, read only in debug builds.
fn dev_env(name: &str) -> Option<String> {
    if !cfg!(debug_assertions) {
        return None;
    }
    std::env::var(name).ok().map(|v| v.trim().to_string()).filter(|v| !v.is_empty())
}

pub fn base_url() -> String {
    dev_env("HIVEGPT_BASE_URL")
        .map(|s| s.trim_end_matches('/').to_string())
        .filter(|s| !s.is_empty())
        .unwrap_or_else(|| DEFAULT_BASE_URL.to_string())
}

fn entry() -> Result<keyring::Entry, String> {
    keyring::Entry::new(KEYRING_SERVICE, KEYRING_USER).map_err(|e| format!("无法访问系统凭据：{e}"))
}

pub fn api_key() -> Result<Option<String>, String> {
    if let Some(k) = dev_env("HIVEGPT_API_KEY") {
        return Ok(Some(k));
    }
    match entry()?.get_password() {
        Ok(k) => Ok(Some(k)),
        Err(keyring::Error::NoEntry) => Ok(None),
        Err(e) => Err(format!("读取系统凭据失败：{e}")),
    }
}

#[derive(serde::Serialize)]
pub struct AccountState {
    signed_in: bool,
    base_url: String,
    /// sk-ab…wxyz, for display only.
    key_hint: Option<String>,
    /// From the WeChat sign-in (empty after pasting a key by hand).
    user_name: String,
    group_name: String,
}

fn hint(key: &str) -> String {
    let chars: Vec<char> = key.chars().collect();
    if chars.len() <= 10 {
        return "••••".into();
    }
    let head: String = chars[..5].iter().collect();
    let tail: String = chars[chars.len() - 4..].iter().collect();
    format!("{head}…{tail}")
}

#[tauri::command]
pub fn account_state(app: AppHandle) -> Result<AccountState, String> {
    let key = api_key()?;
    let info = if key.is_some() { login::read_account(&app) } else { login::AccountInfo::default() };
    Ok(AccountState {
        signed_in: key.is_some(),
        base_url: base_url(),
        key_hint: key.as_deref().map(hint),
        user_name: info.user_name,
        group_name: info.group_name,
    })
}

pub fn store_key(key: &str) -> Result<(), String> {
    let key = key.trim();
    if !key.starts_with("sk-") || key.len() < 20 {
        return Err("Key 格式不对，应该以 sk- 开头".into());
    }
    entry()?.set_password(key).map_err(|e| format!("保存失败：{e}"))
}

/// Development fallback for when the site has no desktop sign-in yet (e.g. a local sub2api).
#[tauri::command]
pub fn save_key(app: AppHandle, key: String) -> Result<AccountState, String> {
    store_key(&key)?;
    login::clear_account(&app);
    account_state(app)
}

#[tauri::command]
pub fn sign_out(app: AppHandle) -> Result<AccountState, String> {
    match entry()?.delete_credential() {
        Ok(()) | Err(keyring::Error::NoEntry) => {}
        Err(e) => return Err(format!("退出失败：{e}")),
    }
    login::clear_account(&app);
    account_state(app)
}

/// Debug builds only: HIVEGPT_DEV_PROMPT makes the app finish setup with the default folder and
/// send this prompt, so the whole chain can be smoke-tested without clicking.
#[tauri::command]
pub fn dev_autorun() -> Option<String> {
    if !cfg!(debug_assertions) {
        return None;
    }
    std::env::var("HIVEGPT_DEV_PROMPT").ok().filter(|s| !s.trim().is_empty())
}

/// Debug builds only: with HIVEGPT_DEV_UNDO set, the chat undoes the HIVEGPT_DEV_PROMPT turn once it
/// finishes, so 撤销这一轮 (snapshot restore + thread/rollback) can be tested without clicking.
#[tauri::command]
pub fn dev_undo() -> bool {
    cfg!(debug_assertions) && std::env::var("HIVEGPT_DEV_UNDO").map(|v| !v.trim().is_empty()).unwrap_or(false)
}

/// 文档/HiveGPT 工作区, created on first use. The user can pick any other folder.
/// Debug builds: HIVEGPT_DEV_WORKSPACE overrides it (used to run each template in its own folder).
#[tauri::command]
pub fn default_workspace(_app: AppHandle) -> Result<String, String> {
    if cfg!(debug_assertions) {
        if let Ok(dir) = std::env::var("HIVEGPT_DEV_WORKSPACE") {
            if !dir.trim().is_empty() {
                std::fs::create_dir_all(&dir).map_err(|e| format!("创建工作区失败：{e}"))?;
                return Ok(dir);
            }
        }
    }
    let docs = dirs::document_dir().or_else(dirs::home_dir).ok_or("找不到「文档」文件夹")?;
    let dir: PathBuf = docs.join("HiveGPT 工作区");
    std::fs::create_dir_all(&dir).map_err(|e| format!("创建工作区失败：{e}"))?;
    Ok(dir.display().to_string())
}
