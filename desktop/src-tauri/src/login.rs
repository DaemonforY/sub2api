//! WeChat sign-in, device-authorization style (server side: backend/internal/service/desktop_login.go).
//!
//! `login_start` asks the site for a ticket and returns the URL the QR shows
//! (<site>/desktop-login?t=…). The phone scans it with WeChat, signs in on the site and confirms.
//! `login_poll` checks every few seconds with the secret device code, which stays in this process
//! and never goes to the web view. When the user confirms, the site returns this device's key, which is
//! saved to the system credential store. The account name and group are kept in account.json for display.

use std::path::PathBuf;

use serde::{Deserialize, Serialize};
use tauri::{AppHandle, Manager, State};
use tokio::sync::Mutex;

use crate::settings;

#[derive(Default)]
pub struct LoginState {
    pending: Mutex<Option<Pending>>,
}

struct Pending {
    ticket_id: String,
    device_code: String,
}

#[derive(Deserialize)]
struct Envelope<T> {
    code: i64,
    #[serde(default)]
    message: String,
    data: Option<T>,
}

#[derive(Deserialize)]
struct Started {
    ticket_id: String,
    device_code: String,
    login_path: String,
    expires_in: u64,
    poll_interval: u64,
}

#[derive(Deserialize)]
struct Polled {
    status: String,
    #[serde(default)]
    api_key: String,
    #[serde(default)]
    group_name: String,
    #[serde(default)]
    user_name: String,
}

#[derive(Serialize)]
pub struct LoginStarted {
    login_url: String,
    expires_in: u64,
    poll_interval: u64,
}

#[derive(Serialize)]
pub struct LoginPoll {
    /// pending / approved / rejected / expired / done
    status: String,
}

/// Shown on the setup screen and in the chat header; never contains the key.
#[derive(Serialize, Deserialize, Default, Clone)]
pub struct AccountInfo {
    #[serde(default)]
    pub user_name: String,
    #[serde(default)]
    pub group_name: String,
}

fn client() -> Result<reqwest::Client, String> {
    reqwest::Client::builder()
        .user_agent(concat!("hivegpt-desktop/", env!("CARGO_PKG_VERSION")))
        .connect_timeout(std::time::Duration::from_secs(10))
        .timeout(std::time::Duration::from_secs(30))
        .build()
        .map_err(|e| e.to_string())
}

async fn post<T: for<'de> Deserialize<'de>>(path: &str, body: serde_json::Value) -> Result<T, String> {
    let url = format!("{}{}", settings::base_url(), path);
    let resp = client()?
        .post(&url)
        .json(&body)
        .send()
        .await
        .map_err(|e| format!("连接 HiveGPT 失败，请检查网络后重试（{e}）"))?;
    let status = resp.status();
    let env: Envelope<T> = resp
        .json()
        .await
        .map_err(|e| format!("服务器返回了无法识别的内容（HTTP {status}，{e}）"))?;
    if env.code != 0 {
        return Err(if env.message.is_empty() { format!("请求失败（HTTP {status}）") } else { env.message });
    }
    env.data.ok_or_else(|| "服务器没有返回数据".to_string())
}

fn account_file(app: &AppHandle) -> Result<PathBuf, String> {
    let dir = app.path().app_data_dir().map_err(|e| format!("找不到应用数据目录：{e}"))?;
    std::fs::create_dir_all(&dir).map_err(|e| e.to_string())?;
    Ok(dir.join("account.json"))
}

pub fn read_account(app: &AppHandle) -> AccountInfo {
    account_file(app)
        .ok()
        .and_then(|p| std::fs::read_to_string(p).ok())
        .and_then(|s| serde_json::from_str(&s).ok())
        .unwrap_or_default()
}

pub fn clear_account(app: &AppHandle) {
    if let Ok(p) = account_file(app) {
        let _ = std::fs::remove_file(p);
    }
}

/// A random id made on first run and kept in the app data dir, so signing in again on this computer
/// reuses its key instead of piling up new ones.
fn device_id(app: &AppHandle) -> Result<String, String> {
    let dir = app.path().app_data_dir().map_err(|e| format!("找不到应用数据目录：{e}"))?;
    std::fs::create_dir_all(&dir).map_err(|e| e.to_string())?;
    let path = dir.join("device-id");
    if let Ok(id) = std::fs::read_to_string(&path) {
        let id = id.trim().to_string();
        if id.len() >= 8 {
            return Ok(id);
        }
    }
    let id = uuid::Uuid::new_v4().to_string();
    std::fs::write(&path, &id).map_err(|e| format!("保存设备编号失败：{e}"))?;
    Ok(id)
}

fn device_name() -> String {
    let name = whoami::devicename();
    if name.trim().is_empty() {
        "我的电脑".to_string()
    } else {
        name
    }
}

#[tauri::command]
pub async fn login_start(app: AppHandle, login: State<'_, LoginState>) -> Result<LoginStarted, String> {
    let body = serde_json::json!({ "device_id": device_id(&app)?, "device_name": device_name() });
    let started: Started = post("/api/v1/auth/desktop/start", body).await?;
    *login.pending.lock().await = Some(Pending { ticket_id: started.ticket_id, device_code: started.device_code });
    Ok(LoginStarted {
        login_url: format!("{}{}", settings::base_url(), started.login_path),
        expires_in: started.expires_in,
        poll_interval: started.poll_interval.max(1),
    })
}

#[tauri::command]
pub async fn login_poll(app: AppHandle, login: State<'_, LoginState>) -> Result<LoginPoll, String> {
    let (ticket_id, device_code) = {
        let guard = login.pending.lock().await;
        let p = guard.as_ref().ok_or("没有进行中的登录，请刷新二维码")?;
        (p.ticket_id.clone(), p.device_code.clone())
    };
    let body = serde_json::json!({ "ticket_id": ticket_id, "device_code": device_code });
    let polled: Polled = post("/api/v1/auth/desktop/poll", body).await?;
    if polled.status == "done" {
        if polled.api_key.is_empty() {
            return Err("服务器没有返回 Key，请重新扫码".into());
        }
        settings::store_key(&polled.api_key)?;
        let info = AccountInfo { user_name: polled.user_name, group_name: polled.group_name };
        if let Ok(p) = account_file(&app) {
            let _ = std::fs::write(p, serde_json::to_string(&info).unwrap_or_default());
        }
    }
    if matches!(polled.status.as_str(), "done" | "rejected" | "expired") {
        *login.pending.lock().await = None;
    }
    Ok(LoginPoll { status: polled.status })
}

#[derive(Serialize)]
pub struct KeyCheck {
    /// ok / invalid (401: deleted, disabled or wrong key) / unreachable (network or server trouble)
    status: &'static str,
    message: String,
}

/// Asks the gateway whether the saved key still works (GET /v1/models), so a revoked key sends the
/// user back to sign-in at once instead of after Codex's ~26 s of retries.
#[tauri::command]
pub async fn check_key() -> Result<KeyCheck, String> {
    let Some(key) = settings::api_key()? else {
        return Ok(KeyCheck { status: "invalid", message: "还没有登录".into() });
    };
    let url = format!("{}/v1/models", settings::base_url());
    let resp = match client()?.get(&url).bearer_auth(key).send().await {
        Ok(r) => r,
        Err(e) => return Ok(KeyCheck { status: "unreachable", message: format!("连接 HiveGPT 失败：{e}") }),
    };
    let code = resp.status().as_u16();
    if code == 401 {
        return Ok(KeyCheck { status: "invalid", message: "这台电脑的 Key 已失效（可能被停用或删除）".into() });
    }
    if resp.status().is_success() {
        return Ok(KeyCheck { status: "ok", message: String::new() });
    }
    // Anything else (403 balance / group problems, 5xx) is left to the conversation's own messages.
    Ok(KeyCheck { status: "ok", message: format!("HTTP {code}") })
}
