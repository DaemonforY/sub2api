//! 撤销这一轮: before each turn the working folder is copied into the app data dir, and undo puts
//! it back. Files that appeared after the snapshot are not deleted — they are moved into
//! 撤销回收站 (app data), so undo can never lose anything.
//!
//! Layout: <app data>/snapshots/<thread id>/<id>/{meta.json, files/…}. `turn_index` is how many
//! turns the conversation had before this one, so the web side can tell whether the newest snapshot
//! really belongs to the turn it is about to roll back.

use std::path::{Path, PathBuf};
use std::time::{Duration, SystemTime, UNIX_EPOCH};

use serde::{Deserialize, Serialize};
use tauri::{AppHandle, Manager};

/// Bigger folders are not copied: the turn still runs, it just can't be undone on disk.
const MAX_BYTES: u64 = 200 * 1024 * 1024;
const MAX_FILES: usize = 5000;
/// Snapshots kept per conversation (older ones are dropped).
const KEEP_PER_THREAD: usize = 10;
/// Conversations untouched this long lose their snapshots.
const MAX_AGE: Duration = Duration::from_secs(14 * 24 * 3600);

#[derive(Serialize, Deserialize, Clone)]
pub struct Meta {
    id: String,
    cwd: String,
    turn_index: u32,
    created_at: u64,
    /// Why the folder was not copied (too big, unreadable…); None when the copy is complete.
    skipped: Option<String>,
}

#[derive(Serialize)]
pub struct Restored {
    /// Files written back from the snapshot.
    restored: usize,
    /// Files that appeared after the snapshot, moved into 撤销回收站.
    moved_aside: usize,
    /// Where they went (empty when nothing was moved).
    trash: String,
}

fn now_secs() -> u64 {
    SystemTime::now().duration_since(UNIX_EPOCH).map(|d| d.as_secs()).unwrap_or(0)
}

fn app_dir(app: &AppHandle) -> Result<PathBuf, String> {
    app.path().app_data_dir().map_err(|e| format!("找不到应用数据目录：{e}"))
}

/// Thread ids come from Codex (UUIDs); refuse anything that could step out of the snapshot dir.
fn thread_dir(base: &Path, thread_id: &str) -> Result<PathBuf, String> {
    if thread_id.is_empty() || !thread_id.chars().all(|c| c.is_ascii_alphanumeric() || c == '-' || c == '_') {
        return Err("会话编号不正确".into());
    }
    Ok(base.join("snapshots").join(thread_id))
}

fn read_meta(dir: &Path) -> Option<Meta> {
    std::fs::read_to_string(dir.join("meta.json")).ok().and_then(|s| serde_json::from_str(&s).ok())
}

/// Snapshots of one conversation, oldest first.
fn list_metas(dir: &Path) -> Vec<(PathBuf, Meta)> {
    let mut out: Vec<(PathBuf, Meta)> = std::fs::read_dir(dir)
        .into_iter()
        .flatten()
        .flatten()
        .filter_map(|e| {
            let p = e.path();
            read_meta(&p).map(|m| (p, m))
        })
        .collect();
    out.sort_by(|a, b| (a.1.turn_index, a.1.created_at).cmp(&(b.1.turn_index, b.1.created_at)));
    out
}

/// Regular files under `root`, as paths relative to it. Symlinks are skipped (never followed).
fn walk(root: &Path, rel: &Path, out: &mut Vec<(PathBuf, u64)>, limit: usize) -> std::io::Result<()> {
    for entry in std::fs::read_dir(root.join(rel))? {
        let entry = entry?;
        let ty = entry.file_type()?;
        let rel_path = rel.join(entry.file_name());
        if ty.is_symlink() {
            continue;
        }
        if ty.is_dir() {
            walk(root, &rel_path, out, limit)?;
        } else if ty.is_file() {
            out.push((rel_path, entry.metadata()?.len()));
        }
        if out.len() > limit {
            return Ok(());
        }
    }
    Ok(())
}

fn take_in(base: &Path, thread_id: &str, cwd: &str, turn_index: u32) -> Result<Meta, String> {
    let tdir = thread_dir(base, thread_id)?;
    std::fs::create_dir_all(&tdir).map_err(|e| format!("创建快照目录失败：{e}"))?;
    // Snapshots at or after this turn are left over from turns that were undone; drop them.
    for (p, m) in list_metas(&tdir) {
        if m.turn_index >= turn_index {
            let _ = std::fs::remove_dir_all(p);
        }
    }

    let created_at = now_secs();
    let id = format!("{turn_index:04}-{}", uuid::Uuid::new_v4().simple());
    let sdir = tdir.join(&id);
    let files_dir = sdir.join("files");
    std::fs::create_dir_all(&files_dir).map_err(|e| format!("创建快照目录失败：{e}"))?;

    let root = PathBuf::from(cwd);
    let mut files = Vec::new();
    let skipped = match walk(&root, Path::new(""), &mut files, MAX_FILES) {
        Err(e) => Some(format!("读取文件夹失败：{e}")),
        Ok(()) if files.len() > MAX_FILES => Some(format!("文件超过 {MAX_FILES} 个")),
        Ok(()) => {
            let total: u64 = files.iter().map(|(_, n)| n).sum();
            if total > MAX_BYTES {
                Some(format!("文件夹超过 {} MB", MAX_BYTES / 1024 / 1024))
            } else {
                let mut err = None;
                for (rel, _) in &files {
                    let dst = files_dir.join(rel);
                    if let Some(parent) = dst.parent() {
                        let _ = std::fs::create_dir_all(parent);
                    }
                    if let Err(e) = std::fs::copy(root.join(rel), &dst) {
                        err = Some(format!("复制「{}」失败：{e}", rel.display()));
                        break;
                    }
                }
                err
            }
        }
    };
    if skipped.is_some() {
        let _ = std::fs::remove_dir_all(&files_dir);
    }

    let meta = Meta { id, cwd: cwd.to_string(), turn_index, created_at, skipped };
    std::fs::write(sdir.join("meta.json"), serde_json::to_string(&meta).unwrap_or_default())
        .map_err(|e| format!("保存快照失败：{e}"))?;

    // Keep the newest few for this conversation, and drop conversations nobody opened for a while.
    let metas = list_metas(&tdir);
    if metas.len() > KEEP_PER_THREAD {
        for (p, _) in &metas[..metas.len() - KEEP_PER_THREAD] {
            let _ = std::fs::remove_dir_all(p);
        }
    }
    prune_old(base, thread_id);
    Ok(meta)
}

fn prune_old(base: &Path, current: &str) {
    let base = base.join("snapshots");
    let cutoff = now_secs().saturating_sub(MAX_AGE.as_secs());
    for entry in std::fs::read_dir(base).into_iter().flatten().flatten() {
        if entry.file_name().to_string_lossy() == current {
            continue;
        }
        let newest = list_metas(&entry.path()).iter().map(|(_, m)| m.created_at).max().unwrap_or(0);
        if newest < cutoff {
            let _ = std::fs::remove_dir_all(entry.path());
        }
    }
}

/// Moves `path` (file or folder) into the trash, keeping its place relative to the working folder.
fn move_aside(path: &Path, rel: &Path, trash: &Path) -> std::io::Result<()> {
    let dst = trash.join(rel);
    if let Some(parent) = dst.parent() {
        std::fs::create_dir_all(parent)?;
    }
    if std::fs::rename(path, &dst).is_ok() {
        return Ok(());
    }
    // Different volume: copy, then remove.
    if path.is_dir() {
        copy_dir(path, &dst)?;
        std::fs::remove_dir_all(path)
    } else {
        std::fs::copy(path, &dst)?;
        std::fs::remove_file(path)
    }
}

fn copy_dir(src: &Path, dst: &Path) -> std::io::Result<()> {
    std::fs::create_dir_all(dst)?;
    for entry in std::fs::read_dir(src)? {
        let entry = entry?;
        let ty = entry.file_type()?;
        if ty.is_dir() {
            copy_dir(&entry.path(), &dst.join(entry.file_name()))?;
        } else if ty.is_file() {
            std::fs::copy(entry.path(), dst.join(entry.file_name()))?;
        }
    }
    Ok(())
}

/// Removes directories under `root` that ended up empty (created during the undone turn).
fn remove_empty_dirs(root: &Path, keep: &std::collections::HashSet<PathBuf>, rel: &Path) {
    let Ok(entries) = std::fs::read_dir(root.join(rel)) else { return };
    for entry in entries.flatten() {
        if entry.file_type().map(|t| t.is_dir()).unwrap_or(false) {
            let child = rel.join(entry.file_name());
            remove_empty_dirs(root, keep, &child);
            if !keep.contains(&child) {
                let _ = std::fs::remove_dir(root.join(&child)); // only succeeds when empty
            }
        }
    }
}

fn restore_in(base: &Path, thread_id: &str, id: &str) -> Result<Restored, String> {
    let tdir = thread_dir(base, thread_id)?;
    let metas = list_metas(&tdir);
    let (sdir, meta) = metas.iter().find(|(_, m)| m.id == id).cloned().ok_or("找不到这一轮的快照")?;
    if let Some(reason) = &meta.skipped {
        return Err(format!("这一轮开始前没有保存文件（{reason}），文件无法还原"));
    }
    let root = PathBuf::from(&meta.cwd);
    let files_dir = sdir.join("files");

    let mut saved = Vec::new();
    walk(&files_dir, Path::new(""), &mut saved, usize::MAX).map_err(|e| format!("读取快照失败：{e}"))?;
    let saved_set: std::collections::HashSet<PathBuf> = saved.iter().map(|(p, _)| p.clone()).collect();
    let mut saved_dirs = std::collections::HashSet::new();
    for p in &saved_set {
        let mut cur = p.parent();
        while let Some(d) = cur {
            if d.as_os_str().is_empty() {
                break;
            }
            saved_dirs.insert(d.to_path_buf());
            cur = d.parent();
        }
    }

    let stamp = chrono_like_stamp();
    let trash = base.join("撤销回收站").join(&stamp);
    let mut moved = 0usize;

    // 1) Files that are in the folder now but were not before the turn: move them aside.
    let mut current = Vec::new();
    walk(&root, Path::new(""), &mut current, usize::MAX).map_err(|e| format!("读取文件夹失败：{e}"))?;
    for (rel, _) in &current {
        if !saved_set.contains(rel) {
            move_aside(&root.join(rel), rel, &trash).map_err(|e| format!("挪开「{}」失败：{e}", rel.display()))?;
            moved += 1;
        }
    }

    // 2) Put every saved file back; anything now standing in its way (a folder with that name) is moved aside.
    let mut restored = 0usize;
    for rel in &saved_set {
        let dst = root.join(rel);
        if dst.is_dir() {
            move_aside(&dst, rel, &trash).map_err(|e| format!("挪开「{}」失败：{e}", rel.display()))?;
            moved += 1;
        }
        if let Some(parent) = dst.parent() {
            if parent.is_file() {
                let prel = parent.strip_prefix(&root).unwrap_or(parent).to_path_buf();
                move_aside(parent, &prel, &trash).map_err(|e| format!("挪开「{}」失败：{e}", prel.display()))?;
                moved += 1;
            }
            std::fs::create_dir_all(parent).map_err(|e| format!("创建文件夹失败：{e}"))?;
        }
        let src = files_dir.join(rel);
        if same_file_content(&src, &dst) {
            continue;
        }
        std::fs::copy(&src, &dst).map_err(|e| format!("还原「{}」失败：{e}", rel.display()))?;
        restored += 1;
    }

    remove_empty_dirs(&root, &saved_dirs, Path::new(""));

    // This snapshot and any later ones are used up.
    for (p, m) in &metas {
        if m.turn_index >= meta.turn_index {
            let _ = std::fs::remove_dir_all(p);
        }
    }
    Ok(Restored {
        restored,
        moved_aside: moved,
        trash: if moved > 0 { trash.display().to_string() } else { String::new() },
    })
}

fn same_file_content(a: &Path, b: &Path) -> bool {
    let (Ok(ma), Ok(mb)) = (std::fs::metadata(a), std::fs::metadata(b)) else { return false };
    if !mb.is_file() || ma.len() != mb.len() {
        return false;
    }
    match (std::fs::read(a), std::fs::read(b)) {
        (Ok(x), Ok(y)) => x == y,
        _ => false,
    }
}

/// 2026-10-31_14-05-09 in local time: the 撤销回收站 folder for one undo.
fn chrono_like_stamp() -> String {
    chrono::Local::now().format("%Y-%m-%d_%H-%M-%S").to_string()
}

#[tauri::command]
pub async fn snapshot_take(app: AppHandle, thread_id: String, cwd: String, turn_index: u32) -> Result<Meta, String> {
    let base = app_dir(&app)?;
    tauri::async_runtime::spawn_blocking(move || take_in(&base, &thread_id, &cwd, turn_index))
        .await
        .map_err(|e| e.to_string())?
}

#[tauri::command]
pub async fn snapshot_list(app: AppHandle, thread_id: String) -> Result<Vec<Meta>, String> {
    let dir = thread_dir(&app_dir(&app)?, &thread_id)?;
    Ok(list_metas(&dir).into_iter().map(|(_, m)| m).collect())
}

#[tauri::command]
pub async fn snapshot_restore(app: AppHandle, thread_id: String, id: String) -> Result<Restored, String> {
    let base = app_dir(&app)?;
    tauri::async_runtime::spawn_blocking(move || restore_in(&base, &thread_id, &id))
        .await
        .map_err(|e| e.to_string())?
}

/// A turn that never started (turn/start failed) must not leave its snapshot behind.
#[tauri::command]
pub async fn snapshot_discard(app: AppHandle, thread_id: String, id: String) -> Result<(), String> {
    let dir = thread_dir(&app_dir(&app)?, &thread_id)?;
    for (p, m) in list_metas(&dir) {
        if m.id == id {
            let _ = std::fs::remove_dir_all(p);
        }
    }
    Ok(())
}

#[cfg(test)]
mod tests {
    use super::*;

    fn write(p: &Path, s: &str) {
        std::fs::create_dir_all(p.parent().unwrap()).unwrap();
        std::fs::write(p, s).unwrap();
    }

    fn tmp(name: &str) -> PathBuf {
        let d = std::env::temp_dir().join(format!("hivegpt-snap-test-{name}-{}", uuid::Uuid::new_v4().simple()));
        std::fs::create_dir_all(&d).unwrap();
        d
    }

    #[test]
    fn undo_restores_edits_and_moves_new_files_aside() {
        let base = tmp("base");
        let ws = tmp("ws");
        write(&ws.join("a.txt"), "original");
        write(&ws.join("子目录/b.xlsx"), "bbb");
        let meta = take_in(&base, "t1", ws.to_str().unwrap(), 0).unwrap();
        assert!(meta.skipped.is_none());

        // The turn edits a, deletes b, creates 输出/new.docx.
        write(&ws.join("a.txt"), "changed by the agent");
        std::fs::remove_file(ws.join("子目录/b.xlsx")).unwrap();
        write(&ws.join("输出/new.docx"), "new");

        let r = restore_in(&base, "t1", &meta.id).unwrap();
        assert_eq!(std::fs::read_to_string(ws.join("a.txt")).unwrap(), "original");
        assert_eq!(std::fs::read_to_string(ws.join("子目录/b.xlsx")).unwrap(), "bbb");
        assert!(!ws.join("输出").exists(), "empty folder created by the turn is removed");
        assert_eq!(r.restored, 2);
        assert_eq!(r.moved_aside, 1);
        assert_eq!(std::fs::read_to_string(Path::new(&r.trash).join("输出/new.docx")).unwrap(), "new");
        // Used up: the snapshot is gone.
        assert!(list_metas(&thread_dir(&base, "t1").unwrap()).is_empty());
    }

    #[test]
    fn taking_a_snapshot_drops_ones_left_from_undone_turns() {
        let base = tmp("base2");
        let ws = tmp("ws2");
        write(&ws.join("a.txt"), "x");
        take_in(&base, "t2", ws.to_str().unwrap(), 0).unwrap();
        take_in(&base, "t2", ws.to_str().unwrap(), 1).unwrap();
        // Turn 1 was undone in the conversation only; the next turn is index 1 again.
        take_in(&base, "t2", ws.to_str().unwrap(), 1).unwrap();
        let idx: Vec<u32> = list_metas(&thread_dir(&base, "t2").unwrap()).iter().map(|(_, m)| m.turn_index).collect();
        assert_eq!(idx, vec![0, 1]);
    }

    #[test]
    fn rejects_thread_ids_that_could_escape() {
        let base = tmp("base3");
        assert!(thread_dir(&base, "../evil").is_err());
        assert!(thread_dir(&base, "").is_err());
        assert!(thread_dir(&base, "019a-ok_id").is_ok());
    }

    #[test]
    fn stamp_looks_like_a_date() {
        let s = chrono_like_stamp();
        assert_eq!(s.len(), 19);
        assert!(s.starts_with("20"));
    }
}
