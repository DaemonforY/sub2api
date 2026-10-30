//! 埋点: small events posted to the site's collector (POST /api/v1/events, app "desktop"), so we can
//! see where people stop — download, sign-in, first task, coming back the next day.
//!
//! Sent from Rust rather than the web view: no CORS, and the device id stays on this side. Nothing
//! personal goes out: no prompts, no file names, only the event name and a few short props.
//! Debug builds send nothing unless HIVEGPT_DEV_TRACK is set, so development doesn't pollute the data.

use std::sync::OnceLock;

use serde_json::{json, Map, Value};
use tauri::{AppHandle, Manager};

use crate::{login, settings};

fn session_id() -> &'static str {
    static ID: OnceLock<String> = OnceLock::new();
    ID.get_or_init(|| uuid::Uuid::new_v4().simple().to_string())
}

fn enabled() -> bool {
    !cfg!(debug_assertions) || std::env::var("HIVEGPT_DEV_TRACK").map(|v| !v.trim().is_empty()).unwrap_or(false)
}

fn send(app: &AppHandle, name: &str, props: Map<String, Value>) {
    if !enabled() {
        return;
    }
    let Ok(visitor) = login::device_id(app) else { return };
    let mut props = props;
    props.insert("v".into(), json!(env!("CARGO_PKG_VERSION")));
    props.insert("os".into(), json!(std::env::consts::OS));
    let body = json!({
        "visitor_id": visitor,
        "session_id": session_id(),
        "app": "desktop",
        "events": [{ "name": name, "path": "/desktop", "props": props }],
    });
    tauri::async_runtime::spawn(async move {
        let Ok(client) = login::client() else { return };
        // Fire and forget: losing an event is fine, slowing the app down is not.
        let _ = client.post(format!("{}/api/v1/events", settings::base_url())).json(&body).send().await;
    });
}

#[tauri::command]
pub fn track(app: AppHandle, name: String, props: Option<Map<String, Value>>) {
    if !name.starts_with("desktop_") {
        return;
    }
    send(&app, &name, props.unwrap_or_default());
}

/// Called once per launch: `desktop_installed` the very first time, then `desktop_open` every time
/// (coming back on day 2 is read from these).
#[tauri::command]
pub fn track_launch(app: AppHandle) {
    if let Ok(dir) = app.path().app_data_dir() {
        let marker = dir.join("installed");
        if !marker.exists() {
            let _ = std::fs::create_dir_all(&dir);
            let _ = std::fs::write(&marker, "1");
            send(&app, "desktop_installed", Map::new());
        }
    }
    send(&app, "desktop_open", Map::new());
}
