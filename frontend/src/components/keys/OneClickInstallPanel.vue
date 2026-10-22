<template>
  <section
    class="rounded-lg border border-emerald-200 bg-emerald-50/50 px-4 py-3 dark:border-emerald-900 dark:bg-emerald-950/20"
    data-testid="use-key-one-click-install"
  >
    <div class="flex flex-wrap items-center justify-between gap-2">
      <p class="text-sm font-medium text-gray-900 dark:text-white">⚡ {{ t('keys.useKeyModal.oneClick.title') }}</p>
      <div class="flex gap-1 rounded-lg bg-white/80 p-0.5 dark:bg-dark-800" role="tablist">
        <button
          v-for="o in OS_TABS"
          :key="o.id"
          type="button"
          role="tab"
          :aria-selected="os === o.id"
          :data-testid="`one-click-os-${o.id}`"
          :class="[
            'rounded-md px-2.5 py-1 text-xs font-medium transition-colors',
            os === o.id ? 'bg-emerald-600 text-white' : 'text-gray-600 hover:text-gray-900 dark:text-dark-300 dark:hover:text-white'
          ]"
          @click="os = o.id"
        >
          {{ o.label }}
        </button>
      </div>
    </div>
    <p class="mt-1 text-xs leading-5 text-gray-600 dark:text-gray-400">{{ t('keys.useKeyModal.oneClick.description') }}</p>

    <!-- Paste one line: PowerShell on Windows (works under Smart App Control), Terminal elsewhere -->
    <ol class="mt-3 space-y-2 text-sm text-gray-700 dark:text-dark-200">
      <li class="flex gap-2">
        <span class="font-semibold text-emerald-600">1</span>
        <span>{{ os === 'mac' ? t('keys.useKeyModal.oneClick.macOpen') : os === 'windows' ? t('keys.useKeyModal.oneClick.windowsOpen') : t('keys.useKeyModal.oneClick.linuxOpen') }}</span>
      </li>
      <li class="flex gap-2">
        <span class="font-semibold text-emerald-600">2</span>
        <span class="min-w-0 flex-1 space-y-1">
          <span class="block">{{ os === 'mac' ? t('keys.useKeyModal.oneClick.macPaste') : os === 'windows' ? t('keys.useKeyModal.oneClick.windowsPaste') : t('keys.useKeyModal.oneClick.otherPaste') }}</span>
          <CommandLine :command="command" @copy="copyCommand" :copied="copied" />
        </span>
      </li>
      <li class="flex gap-2"><span class="font-semibold text-emerald-600">3</span><span>{{ t('keys.useKeyModal.oneClick.waitDone') }}</span></li>
    </ol>

    <!-- Windows fallback: a double-click installer. Smart App Control blocks unsigned downloaded
         scripts with no override, so it comes second and says how to unblock it. -->
    <details v-if="os === 'windows' && windowsScript" class="mt-2 text-xs text-gray-500 dark:text-gray-400" data-testid="one-click-cmd-fallback">
      <summary class="cursor-pointer hover:text-gray-700 dark:hover:text-gray-200">{{ t('keys.useKeyModal.oneClick.cmdFallback') }}</summary>
      <ol class="mt-2 space-y-1.5 text-gray-600 dark:text-dark-300">
        <li class="flex flex-wrap items-center gap-2">
          <span class="font-semibold text-emerald-600">1</span>
          <button type="button" class="btn btn-secondary btn-sm" data-testid="one-click-download" @click="downloadInstaller">
            ⬇ {{ t('keys.useKeyModal.oneClick.windowsDownload') }}
          </button>
        </li>
        <li class="flex gap-2"><span class="font-semibold text-emerald-600">2</span><span>{{ t('keys.useKeyModal.oneClick.windowsRun', { name: WINDOWS_INSTALLER_NAME }) }}</span></li>
        <li class="flex gap-2"><span class="font-semibold text-emerald-600">3</span><span>{{ t('keys.useKeyModal.oneClick.windowsBlocked') }}</span></li>
        <li class="flex gap-2"><span class="font-semibold text-emerald-600">4</span><span>{{ t('keys.useKeyModal.oneClick.windowsDone') }}</span></li>
      </ol>
    </details>

    <p class="mt-2 text-xs leading-5 text-gray-500 dark:text-gray-400">
      {{ t('keys.useKeyModal.oneClick.keyWarning') }}
      {{ t('keys.useKeyModal.oneClick.restore') }}
      <code class="break-all rounded bg-gray-100 px-1 font-mono text-[11px] text-gray-700 dark:bg-dark-700 dark:text-gray-300">{{ restoreCommand }}</code>
    </p>
  </section>
</template>

<script setup lang="ts">
import { computed, defineComponent, h, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useClipboard } from '@/composables/useClipboard'
import { detectDesktopOS, downloadTextFile, type DesktopOS } from '@/utils/codexStarterConfig'
import {
  oneClickInstallCommand,
  oneClickRestoreCommand,
  windowsInstallerScript,
  WINDOWS_INSTALLER_NAME
} from '@/utils/oneClickInstall'
import { track } from '@/utils/analytics'

const props = defineProps<{ apiKey: string; baseUrl: string; source: string }>()
const emit = defineEmits<{ (e: 'used'): void }>()

const { t } = useI18n()
const { copyToClipboard } = useClipboard()

const OS_TABS: { id: DesktopOS; label: string }[] = [
  { id: 'windows', label: 'Windows' },
  { id: 'mac', label: 'macOS' },
  { id: 'linux', label: 'Linux' }
]
const os = ref<DesktopOS>(detectDesktopOS(typeof navigator === 'undefined' ? '' : navigator.userAgent))
const copied = ref(false)

const base = computed(() => props.baseUrl || window.location.origin)
const command = computed(() => oneClickInstallCommand(os.value === 'windows' ? 'windows' : 'unix', base.value, props.apiKey))
const restoreCommand = computed(() => oneClickRestoreCommand(os.value === 'windows' ? 'windows' : 'unix', base.value))
const windowsScript = computed(() => windowsInstallerScript(base.value, props.apiKey))

function downloadInstaller(): void {
  if (!windowsScript.value) return
  downloadTextFile(WINDOWS_INSTALLER_NAME, windowsScript.value)
  track('key_config_copied', { client: 'one-click-install', os: 'windows', via: 'download', source: props.source })
  emit('used')
}

async function copyCommand(): Promise<void> {
  if (!(await copyToClipboard(command.value, t('keys.copied')))) return
  track('key_config_copied', { client: 'one-click-install', os: os.value, source: props.source })
  emit('used')
  copied.value = true
  setTimeout(() => (copied.value = false), 2000)
}

const CommandLine = defineComponent({
  props: { command: { type: String, required: true }, copied: Boolean },
  emits: ['copy'],
  setup(p, { emit: emitCopy }) {
    return () =>
      h('div', { class: 'flex items-stretch gap-2' }, [
        h('code', {
          class: 'min-w-0 flex-1 overflow-x-auto whitespace-nowrap rounded-md bg-gray-900 px-3 py-2 font-mono text-xs text-gray-100',
          'data-testid': 'one-click-command'
        }, p.command),
        h('button', {
          type: 'button',
          class: 'btn btn-primary btn-sm flex-shrink-0',
          'data-testid': 'one-click-copy',
          onClick: () => emitCopy('copy')
        }, p.copied ? t('keys.useKeyModal.copied') : t('keys.useKeyModal.copy'))
      ])
  }
})
</script>
