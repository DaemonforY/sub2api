mod bridge;
mod engine;
mod login;
mod settings;

#[cfg_attr(mobile, tauri::mobile_entry_point)]
pub fn run() {
    tauri::Builder::default()
        .plugin(tauri_plugin_dialog::init())
        .plugin(tauri_plugin_opener::init())
        .manage(bridge::Bridge::default())
        .manage(login::LoginState::default())
        .invoke_handler(tauri::generate_handler![
            engine::engine_status,
            engine::engine_install,
            bridge::engine_start,
            bridge::engine_send,
            bridge::engine_stop,
            login::login_start,
            login::login_poll,
            settings::account_state,
            settings::save_key,
            settings::sign_out,
            settings::default_workspace,
            settings::dev_autorun,
        ])
        .run(tauri::generate_context!())
        .expect("error while running HiveGPT 桌面助手");
}
