//! 技能库: the skills operators publish in 后台 → 桌面助手 (server: backend/internal/service/desktop_skills.go).
//!
//! On every launch `skills_sync` fetches GET /api/v1/desktop/skills and writes each skill's SKILL.md
//! into the private CODEX_HOME (`skills/hivegpt-<slug>/SKILL.md`), where Codex picks it up as a user
//! skill — so it also follows a skill when the user only describes the task. A skill is rewritten only
//! when its version changed, and folders of skills that were taken down are removed. The cards for the
//! empty chat come back to the web side; offline, the last list is used, and with none at all the web
//! side falls back to the built-in templates.

use std::collections::HashMap;
use std::path::{Path, PathBuf};

use serde::{Deserialize, Serialize};
use tauri::{AppHandle, Manager};

use crate::{bridge, login, settings};

/// Folders we own inside CODEX_HOME/skills (so we never touch anything else there).
const SKILL_PREFIX: &str = "hivegpt-";

#[derive(Deserialize, Serialize, Clone)]
pub struct Skill {
    slug: String,
    title: String,
    #[serde(default)]
    icon: String,
    #[serde(default)]
    summary: String,
    #[serde(default)]
    prepare: String,
    #[serde(default)]
    prompt: String,
    #[serde(default)]
    sample_prompt: String,
    #[serde(default)]
    needs_office: bool,
    #[serde(default)]
    version: i64,
    skill_name: String,
    /// Not sent to the web view (it only needs the card).
    #[serde(default, skip_serializing)]
    skill_md: String,
}

#[derive(Deserialize)]
struct Envelope {
    code: i64,
    #[serde(default)]
    message: String,
    data: Option<Payload>,
}

#[derive(Deserialize)]
struct Payload {
    skills: Vec<Skill>,
}

#[derive(Serialize)]
pub struct SkillsState {
    skills: Vec<Skill>,
    /// "server" (fresh), "cache" (last good list, the site was unreachable) or "none".
    source: &'static str,
}

fn app_dir(app: &AppHandle) -> Result<PathBuf, String> {
    app.path().app_data_dir().map_err(|e| format!("找不到应用数据目录：{e}"))
}

fn valid_name(name: &str) -> bool {
    name.starts_with(SKILL_PREFIX)
        && name.len() <= 64
        && name[SKILL_PREFIX.len()..].chars().all(|c| c.is_ascii_lowercase() || c.is_ascii_digit() || c == '-')
}

/// Writes changed skills and removes ours that are gone. `versions` is what is on disk now.
fn apply(skills_dir: &Path, skills: &[Skill], versions: &mut HashMap<String, i64>) -> Result<(), String> {
    std::fs::create_dir_all(skills_dir).map_err(|e| format!("创建技能目录失败：{e}"))?;
    for s in skills {
        if !valid_name(&s.skill_name) || s.skill_md.trim().is_empty() {
            continue;
        }
        let file = skills_dir.join(&s.skill_name).join("SKILL.md");
        if versions.get(&s.skill_name) == Some(&s.version) && file.is_file() {
            continue;
        }
        std::fs::create_dir_all(file.parent().unwrap()).map_err(|e| format!("写入技能失败：{e}"))?;
        std::fs::write(&file, &s.skill_md).map_err(|e| format!("写入技能失败：{e}"))?;
        versions.insert(s.skill_name.clone(), s.version);
    }
    let keep: Vec<&str> = skills.iter().map(|s| s.skill_name.as_str()).collect();
    for entry in std::fs::read_dir(skills_dir).into_iter().flatten().flatten() {
        let name = entry.file_name().to_string_lossy().to_string();
        if valid_name(&name) && !keep.contains(&name.as_str()) {
            let _ = std::fs::remove_dir_all(entry.path());
            versions.remove(&name);
        }
    }
    Ok(())
}

async fn fetch() -> Result<Vec<Skill>, String> {
    let resp = login::client()?
        .get(format!("{}/api/v1/desktop/skills", settings::base_url()))
        .timeout(std::time::Duration::from_secs(8))
        .send()
        .await
        .map_err(|e| e.to_string())?;
    let status = resp.status();
    let env: Envelope = resp.json().await.map_err(|e| format!("HTTP {status}: {e}"))?;
    if env.code != 0 {
        return Err(env.message);
    }
    Ok(env.data.map(|d| d.skills).unwrap_or_default())
}

#[tauri::command]
pub async fn skills_sync(app: AppHandle) -> Result<SkillsState, String> {
    let dir = app_dir(&app)?;
    let catalog_file = dir.join("skills-catalog.json");
    let versions_file = dir.join("skills-versions.json");
    let skills_dir = bridge::codex_home(&app)?.join("skills");

    match fetch().await {
        Ok(skills) => {
            let mut versions: HashMap<String, i64> = std::fs::read_to_string(&versions_file)
                .ok()
                .and_then(|s| serde_json::from_str(&s).ok())
                .unwrap_or_default();
            apply(&skills_dir, &skills, &mut versions)?;
            let _ = std::fs::write(&versions_file, serde_json::to_string(&versions).unwrap_or_default());
            // The cache keeps the cards (skill_md is not serialized; it is already on disk).
            let _ = std::fs::write(&catalog_file, serde_json::to_string(&skills).unwrap_or_default());
            Ok(SkillsState { skills, source: "server" })
        }
        Err(_) => {
            let cached: Option<Vec<Skill>> =
                std::fs::read_to_string(&catalog_file).ok().and_then(|s| serde_json::from_str(&s).ok());
            Ok(match cached {
                Some(skills) => SkillsState { skills, source: "cache" },
                None => SkillsState { skills: Vec::new(), source: "none" },
            })
        }
    }
}

/// `[[skills.config]]` entries that switch off every skill we didn't install: the user's own global
/// ones (~/.agents/skills, e.g. a developer's design or Feishu skills) and Codex's bundled system skills
/// (skill-creator, plugin-creator, openai-docs, imagegen). They cost context and, for non-developers,
/// only lead Codex astray. Rebuilt on every start, since config.toml is rewritten then.
pub fn disabled_skills_toml(codex_home: &Path) -> String {
    let mut roots: Vec<PathBuf> = vec![codex_home.join("skills").join(".system")];
    if let Some(home) = dirs::home_dir() {
        roots.push(home.join(".agents").join("skills"));
    }
    let mut out = String::new();
    for root in roots {
        for entry in std::fs::read_dir(&root).into_iter().flatten().flatten() {
            let file = entry.path().join("SKILL.md");
            if !file.is_file() {
                continue;
            }
            let mut paths = vec![file.clone()];
            if let Ok(real) = std::fs::canonicalize(&file) {
                if real != file {
                    paths.push(real);
                }
            }
            for p in paths {
                let quoted = serde_json::to_string(&p.display().to_string()).unwrap_or_default();
                out.push_str(&format!("\n[[skills.config]]\npath = {quoted}\nenabled = false\n"));
            }
        }
    }
    out
}

#[cfg(test)]
mod tests {
    use super::*;

    fn skill(slug: &str, version: i64) -> Skill {
        Skill {
            slug: slug.into(),
            title: slug.into(),
            icon: String::new(),
            summary: String::new(),
            prepare: String::new(),
            prompt: "p".into(),
            sample_prompt: String::new(),
            needs_office: false,
            version,
            skill_name: format!("hivegpt-{slug}"),
            skill_md: format!("---\nname: hivegpt-{slug}\ndescription: \"v{version}\"\n---\n"),
        }
    }

    #[test]
    fn writes_changed_skills_and_removes_taken_down_ones() {
        let dir = std::env::temp_dir().join(format!("hivegpt-skills-test-{}", uuid::Uuid::new_v4().simple()));
        std::fs::create_dir_all(dir.join(".system/skill-creator")).unwrap();
        std::fs::create_dir_all(dir.join("someone-elses")).unwrap();
        let mut versions = HashMap::new();

        apply(&dir, &[skill("invoices", 1), skill("rename", 1)], &mut versions).unwrap();
        assert!(dir.join("hivegpt-invoices/SKILL.md").is_file());
        assert_eq!(versions.get("hivegpt-rename"), Some(&1));

        // invoices v2 is rewritten, rename was taken down.
        apply(&dir, &[skill("invoices", 2)], &mut versions).unwrap();
        assert!(std::fs::read_to_string(dir.join("hivegpt-invoices/SKILL.md")).unwrap().contains("v2"));
        assert!(!dir.join("hivegpt-rename").exists());
        assert!(!versions.contains_key("hivegpt-rename"));
        // Folders we don't own are left alone.
        assert!(dir.join(".system/skill-creator").exists());
        assert!(dir.join("someone-elses").exists());

        // A bad name from the server never becomes a path.
        let mut bad = skill("x", 1);
        bad.skill_name = "../evil".into();
        apply(&dir, &[bad], &mut versions).unwrap();
        assert!(!dir.parent().unwrap().join("evil").exists());
    }

    #[test]
    fn disables_system_skills() {
        let home = std::env::temp_dir().join(format!("hivegpt-home-test-{}", uuid::Uuid::new_v4().simple()));
        let sys = home.join("skills/.system/skill-creator");
        std::fs::create_dir_all(&sys).unwrap();
        std::fs::write(sys.join("SKILL.md"), "x").unwrap();
        let toml = disabled_skills_toml(&home);
        assert!(toml.contains("skill-creator/SKILL.md"));
        assert!(toml.contains("enabled = false"));
    }
}
