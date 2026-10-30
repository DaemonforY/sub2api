// 埋点 helper: forwards to the Rust side (src-tauri/src/track.rs), which posts to the site. Never throws.

import { invoke } from '@tauri-apps/api/core'

export type TrackProps = Record<string, string | number | boolean>

export function track(name: `desktop_${string}`, props?: TrackProps): void {
  invoke('track', { name, props: props ?? null }).catch(() => {})
}
