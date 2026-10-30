//! Runs `codex app-server` as a child process and relays its newline-delimited JSON-RPC.
//! The protocol itself (request ids, approvals, streaming) is handled in the web UI
//! (src/lib/codex.ts); this side only spawns, writes lines, and emits every stdout line
//! as an `engine-message` event.

use std::path::PathBuf;
use std::process::Stdio;

use tauri::{AppHandle, Emitter, Manager, State};
use tokio::io::{AsyncBufReadExt, AsyncWriteExt, BufReader};
use tokio::process::{Child, ChildStdin, Command};
use tokio::sync::Mutex;

use crate::engine;
use crate::settings;

#[derive(Default)]
pub struct Bridge {
    inner: Mutex<Option<Running>>,
}

struct Running {
    child: Child,
    stdin: ChildStdin,
}

/// Instructions every thread gets, written to the private CODEX_HOME as AGENTS.md.
const AGENTS_MD: &str = r#"# 给 HiveGPT 桌面助手的工作约定

使用这个助手的人大多不是程序员，请这样工作：

- 始终用简体中文回答，少用技术术语；必须用的时候顺带一句话解释。
- 动手之前先用一两句话说明打算怎么做、会改哪些文件。
- 不删除、不覆盖用户的原始文件。生成的新文件放在当前文件夹下的「输出」子文件夹里。
- 完成后告诉用户结果在哪里、怎么打开，以及有没有没做完或拿不准的地方。
- 能用已有工具完成的，不要去安装新软件；确实需要时先说明原因。
"#;

fn codex_home(app: &AppHandle) -> Result<PathBuf, String> {
    let base = app.path().app_data_dir().map_err(|e| format!("找不到应用数据目录：{e}"))?;
    Ok(base.join("codex-home"))
}

fn toml_string(s: &str) -> String {
    serde_json::to_string(s).unwrap_or_else(|_| "\"\"".into())
}

/// Writes the private config every start, so a changed base URL or engine version applies.
fn write_codex_home(home: &PathBuf, base_url: &str) -> Result<(), String> {
    std::fs::create_dir_all(home).map_err(|e| format!("创建配置目录失败：{e}"))?;
    let config = format!(
        r#"# 由 HiveGPT 桌面助手生成，每次启动都会重写。
model = "gpt-6.1-sol"
model_provider = "hivegpt"

[model_providers.hivegpt]
name = "HiveGPT"
base_url = {base}
env_key = "HIVEGPT_API_KEY"
wire_api = "responses"
"#,
        base = toml_string(&format!("{}/v1", base_url.trim_end_matches('/')))
    );
    std::fs::write(home.join("config.toml"), config).map_err(|e| format!("写入配置失败：{e}"))?;
    std::fs::write(home.join("AGENTS.md"), AGENTS_MD).map_err(|e| format!("写入配置失败：{e}"))?;
    Ok(())
}

#[tauri::command]
pub async fn engine_start(app: AppHandle, bridge: State<'_, Bridge>) -> Result<(), String> {
    let mut guard = bridge.inner.lock().await;
    if let Some(running) = guard.as_mut() {
        if running.child.try_wait().map_err(|e| e.to_string())?.is_none() {
            return Ok(());
        }
    }

    let bin = engine::binary_path(&app)?;
    if !bin.is_file() {
        return Err("引擎还没有准备好".into());
    }
    let key = settings::api_key()?.ok_or("还没有登录")?;
    let base_url = settings::base_url();
    let home = codex_home(&app)?;
    write_codex_home(&home, &base_url)?;

    // Codex looks for its bundled ripgrep next to the binary (vendor/<triple>/codex-path).
    let vendor = engine::vendor_dir(&app)?;
    let mut path_dirs = vec![vendor.join("codex-path")];
    if let Some(existing) = std::env::var_os("PATH") {
        path_dirs.extend(std::env::split_paths(&existing));
    }
    let path = std::env::join_paths(path_dirs).map_err(|e| e.to_string())?;

    let mut cmd = Command::new(&bin);
    cmd.arg("app-server")
        .env("CODEX_HOME", &home)
        .env("HIVEGPT_API_KEY", key)
        .env("PATH", path)
        .env_remove("OPENAI_API_KEY")
        .env_remove("OPENAI_BASE_URL")
        .stdin(Stdio::piped())
        .stdout(Stdio::piped())
        .stderr(Stdio::piped())
        .kill_on_drop(true);
    #[cfg(windows)]
    {
        const CREATE_NO_WINDOW: u32 = 0x0800_0000;
        cmd.creation_flags(CREATE_NO_WINDOW);
    }
    let mut child = cmd.spawn().map_err(|e| format!("启动引擎失败：{e}"))?;
    let stdin = child.stdin.take().ok_or("引擎没有输入管道")?;
    let stdout = child.stdout.take().ok_or("引擎没有输出管道")?;
    let stderr = child.stderr.take().ok_or("引擎没有错误输出管道")?;

    let out_app = app.clone();
    tauri::async_runtime::spawn(async move {
        let mut lines = BufReader::new(stdout).lines();
        while let Ok(Some(line)) = lines.next_line().await {
            let _ = out_app.emit("engine-message", line);
        }
        let _ = out_app.emit("engine-exit", ());
    });

    let log_path = home.join("engine-stderr.log");
    tauri::async_runtime::spawn(async move {
        let mut log = tokio::fs::OpenOptions::new().create(true).append(true).open(&log_path).await.ok();
        let mut lines = BufReader::new(stderr).lines();
        while let Ok(Some(line)) = lines.next_line().await {
            if let Some(f) = log.as_mut() {
                let _ = f.write_all(format!("{line}\n").as_bytes()).await;
            }
        }
    });

    *guard = Some(Running { child, stdin });
    Ok(())
}

#[tauri::command]
pub async fn engine_send(line: String, bridge: State<'_, Bridge>) -> Result<(), String> {
    let mut guard = bridge.inner.lock().await;
    let running = guard.as_mut().ok_or("引擎没有在运行")?;
    let mut data = line.into_bytes();
    data.push(b'\n');
    running.stdin.write_all(&data).await.map_err(|e| format!("发送失败：{e}"))?;
    running.stdin.flush().await.map_err(|e| e.to_string())
}

#[tauri::command]
pub async fn engine_stop(bridge: State<'_, Bridge>) -> Result<(), String> {
    if let Some(mut running) = bridge.inner.lock().await.take() {
        let _ = running.child.kill().await;
    }
    Ok(())
}
