// 技能库: cards for the empty chat come from the skills published in 后台 → 桌面助手, synced by the
// Rust side (src-tauri/src/skills.rs), which also writes each skill's SKILL.md for Codex. Offline the
// last synced list is used; with nothing at all, the built-in templates.

import { invoke } from '@tauri-apps/api/core'
import { TEMPLATES, type TaskTemplate } from './templates'

interface SyncedSkill {
  slug: string
  title: string
  icon: string
  summary: string
  prepare: string
  prompt: string
  sample_prompt: string
  needs_office: boolean
}

export function toTemplate(s: SyncedSkill): TaskTemplate {
  return {
    id: s.slug,
    icon: s.icon || '🧩',
    title: s.title,
    desc: s.summary,
    prepare: s.prepare,
    prompt: s.prompt,
    samplePrompt: s.sample_prompt,
    office: s.needs_office
  }
}

/** Must run before the engine starts, so Codex finds the SKILL.md files. Never throws. */
export async function syncTemplates(): Promise<TaskTemplate[]> {
  const res = await invoke<{ skills: SyncedSkill[]; source: string }>('skills_sync').catch(() => null)
  const list = (res?.skills ?? []).filter((s) => s.slug && s.title && s.prompt).map(toTemplate)
  return list.length ? list : TEMPLATES
}
