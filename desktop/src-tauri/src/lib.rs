mod bridge;
mod download;
mod engine;
mod login;
mod office;
mod settings;
mod skills;
mod snapshot;
mod track;

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
            office::office_status,
            office::office_install,
            bridge::engine_start,
            bridge::engine_send,
            bridge::engine_stop,
            login::login_start,
            login::login_poll,
            login::check_key,
            login::usage_summary,
            skills::skills_sync,
            snapshot::snapshot_take,
            snapshot::snapshot_list,
            snapshot::snapshot_restore,
            snapshot::snapshot_discard,
            track::track,
            track::track_launch,
            settings::account_state,
            settings::save_key,
            settings::sign_out,
            settings::default_workspace,
            settings::dev_autorun,
            settings::dev_undo,
        ])
        .run(tauri::generate_context!())
        .expect("error while running HiveGPT 桌面助手");
}
