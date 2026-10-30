//! Where the app talks to and the device key. The key lives in the system credential store
//! (Windows Credential Manager / macOS Keychain), never in a plain file.
//!
//! Until WeChat login and POST /api/v1/desktop/session exist, the key is entered on the setup
//! screen. Developers can also set HIVEGPT_API_KEY / HIVEGPT_BASE_URL in the environment.

use std::path::PathBuf;

use tauri::AppHandle;

const KEYRING_SERVICE: &str = "cn.hivegpt.desktop";
const KEYRING_USER: &str = "device-key";
const DEFAULT_BASE_URL: &str = "https://hivegpt.cn";

pub fn base_url() -> String {
    std::env::var("HIVEGPT_BASE_URL")
        .ok()
        .map(|s| s.trim().trim_end_matches('/').to_string())
        .filter(|s| !s.is_empty())
        .unwrap_or_else(|| DEFAULT_BASE_URL.to_string())
}

fn entry() -> Result<keyring::Entry, String> {
    keyring::Entry::new(KEYRING_SERVICE, KEYRING_USER).map_err(|e| format!("无法访问系统凭据：{e}"))
}

pub fn api_key() -> Result<Option<String>, String> {
    if let Ok(k) = std::env::var("HIVEGPT_API_KEY") {
        if !k.trim().is_empty() {
            return Ok(Some(k.trim().to_string()));
        }
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
pub fn account_state() -> Result<AccountState, String> {
    let key = api_key()?;
    Ok(AccountState { signed_in: key.is_some(), base_url: base_url(), key_hint: key.as_deref().map(hint) })
}

#[tauri::command]
pub fn save_key(key: String) -> Result<AccountState, String> {
    let key = key.trim();
    if !key.starts_with("sk-") || key.len() < 20 {
        return Err("Key 格式不对，应该以 sk- 开头".into());
    }
    entry()?.set_password(key).map_err(|e| format!("保存失败：{e}"))?;
    account_state()
}

#[tauri::command]
pub fn sign_out() -> Result<AccountState, String> {
    match entry()?.delete_credential() {
        Ok(()) | Err(keyring::Error::NoEntry) => {}
        Err(e) => return Err(format!("退出失败：{e}")),
    }
    account_state()
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

/// 文档/HiveGPT 工作区, created on first use. The user can pick any other folder.
#[tauri::command]
pub fn default_workspace(_app: AppHandle) -> Result<String, String> {
    let docs = dirs::document_dir().or_else(dirs::home_dir).ok_or("找不到「文档」文件夹")?;
    let dir: PathBuf = docs.join("HiveGPT 工作区");
    std::fs::create_dir_all(&dir).map_err(|e| format!("创建工作区失败：{e}"))?;
    Ok(dir.display().to_string())
}
