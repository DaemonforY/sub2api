<template>
  <div class="space-y-4" data-testid="wechat-template-notices">
    <div>
      <h4 class="font-medium text-gray-900 dark:text-white">
        {{ t('admin.settings.wechatConnect.templateNotices.title') }}
      </h4>
      <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
        {{ t('admin.settings.wechatConnect.templateNotices.description') }}
      </p>
    </div>

    <ul
      class="list-disc space-y-1 rounded-lg border border-sky-200 bg-sky-50 py-3 pl-8 pr-4 text-xs leading-5 text-sky-800 dark:border-sky-900/40 dark:bg-sky-900/10 dark:text-sky-200"
    >
      <li>{{ t('admin.settings.wechatConnect.templateNotices.helpOpen') }}</li>
      <li>{{ helpKeysText }}</li>
      <li>{{ t('admin.settings.wechatConnect.templateNotices.helpNoMarketing') }}</li>
      <li>{{ t('admin.settings.wechatConnect.templateNotices.helpTest') }}</li>
    </ul>

    <div
      v-for="form in forms"
      :key="form.event"
      class="space-y-3 rounded-lg border border-gray-200 p-4 dark:border-dark-700"
      :data-testid="`wechat-template-notice-${form.event}`"
    >
      <div class="flex items-center justify-between gap-4">
        <p class="text-sm font-medium text-gray-900 dark:text-white">
          {{ t(`admin.settings.wechatConnect.templateNotices.events.${form.event}`) }}
        </p>
        <Toggle v-model="form.enabled" :data-testid="`wechat-template-notice-${form.event}-enabled`" />
      </div>

      <div class="grid grid-cols-1 gap-3 lg:grid-cols-2">
        <div>
          <label class="mb-1 block text-xs font-medium text-gray-700 dark:text-gray-300">
            {{ t('admin.settings.wechatConnect.templateNotices.templateId') }}
          </label>
          <input
            v-model="form.template_id"
            type="text"
            class="input font-mono text-sm"
            :data-testid="`wechat-template-notice-${form.event}-template-id`"
            :placeholder="t('admin.settings.wechatConnect.templateNotices.templateIdPlaceholder')"
          />
        </div>
        <div>
          <label class="mb-1 block text-xs font-medium text-gray-700 dark:text-gray-300">
            {{ t('admin.settings.wechatConnect.templateNotices.urlPath') }}
          </label>
          <input
            v-model="form.url_path"
            type="text"
            class="input font-mono text-sm"
            :class="{ 'input-error': urlPathInvalid(form) }"
            :data-testid="`wechat-template-notice-${form.event}-url`"
            :placeholder="defaultUrlPath(form.event)"
          />
          <p v-if="urlPathInvalid(form)" class="mt-1 text-xs text-red-600 dark:text-red-400">
            {{ t('admin.settings.wechatConnect.templateNotices.urlPathInvalid') }}
          </p>
        </div>
      </div>

      <div class="space-y-2">
        <p class="text-xs font-medium text-gray-700 dark:text-gray-300">
          {{ t('admin.settings.wechatConnect.templateNotices.fields') }}
        </p>
        <div
          v-for="(row, index) in form.fields"
          :key="index"
          class="grid grid-cols-[minmax(0,8rem)_minmax(0,1fr)_auto] gap-2"
        >
          <div>
            <input
              v-model="row.key"
              type="text"
              class="input font-mono text-sm"
              :class="{ 'input-error': isTemplateFieldKeyInvalid(row.key) }"
              placeholder="thing1"
              :data-testid="`wechat-template-notice-${form.event}-field-key-${index}`"
            />
          </div>
          <input
            v-model="row.value"
            type="text"
            class="input text-sm"
            :placeholder="t('admin.settings.wechatConnect.templateNotices.fieldValuePlaceholder')"
            :data-testid="`wechat-template-notice-${form.event}-field-value-${index}`"
          />
          <button type="button" class="btn btn-secondary btn-sm" @click="form.fields.splice(index, 1)">
            {{ t('admin.settings.wechatConnect.templateNotices.removeField') }}
          </button>
        </div>
        <p
          v-if="form.fields.some((row) => isTemplateFieldKeyInvalid(row.key))"
          class="text-xs text-red-600 dark:text-red-400"
          :data-testid="`wechat-template-notice-${form.event}-key-error`"
        >
          {{ t('admin.settings.wechatConnect.templateNotices.fieldKeyInvalid') }}
        </p>
        <button
          type="button"
          class="btn btn-secondary btn-sm"
          :data-testid="`wechat-template-notice-${form.event}-add-field`"
          @click="form.fields.push({ key: '', value: '' })"
        >
          {{ t('admin.settings.wechatConnect.templateNotices.addField') }}
        </button>
        <p class="text-xs leading-5 text-gray-500 dark:text-gray-400">
          {{ t('admin.settings.wechatConnect.templateNotices.placeholders') }}
          <code
            v-for="name in placeholdersFor(form.event)"
            :key="name"
            class="mx-0.5 rounded bg-gray-100 px-1 font-mono text-[11px] text-gray-700 dark:bg-dark-800 dark:text-gray-300"
          >{{ placeholderLabel(name) }}</code>
        </p>
      </div>

      <div class="flex flex-wrap items-center gap-3">
        <button
          type="button"
          class="btn btn-secondary btn-sm"
          :disabled="testing === form.event"
          :data-testid="`wechat-template-notice-${form.event}-test`"
          @click="sendTest(form.event)"
        >
          {{
            testing === form.event
              ? t('admin.settings.wechatConnect.templateNotices.testing')
              : t('admin.settings.wechatConnect.templateNotices.test')
          }}
        </button>
        <span
          v-if="testResults[form.event]"
          :class="[
            'text-xs',
            testResults[form.event]?.ok ? 'text-emerald-600 dark:text-emerald-400' : 'text-red-600 dark:text-red-400',
          ]"
          :data-testid="`wechat-template-notice-${form.event}-test-result`"
        >
          {{ testResults[form.event]?.message }}
        </span>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import Toggle from '@/components/common/Toggle.vue'
import { testWeChatTemplateNotice, type WeChatTemplateNoticeEvent } from '@/api/admin/settings'
import { extractApiErrorMessage } from '@/utils/apiError'
import {
  COMMON_TEMPLATE_PLACEHOLDERS,
  WECHAT_TEMPLATE_NOTICE_EVENTS,
  isSafeSiteUrlPath,
  isTemplateFieldKeyInvalid,
  type WeChatTemplateNoticeForm,
} from '@/utils/wechatTemplateNotices'

/** 微信提醒: one card per event (switch, 模板 ID, 跳转页面, field rows, test send). */
const forms = defineModel<WeChatTemplateNoticeForm[]>({ required: true })

const { t } = useI18n()
const testing = ref<WeChatTemplateNoticeEvent | null>(null)
const testResults = reactive<Partial<Record<WeChatTemplateNoticeEvent, { ok: boolean; message: string }>>>({})

// Built here: mustache-like text can't sit inside template interpolations.
const helpKeysText = computed(() =>
  t('admin.settings.wechatConnect.templateNotices.helpKeys', { example: '{{thing1.DATA}}', key: 'thing1' }),
)

function placeholderLabel(name: string): string {
  return '{' + name + '}'
}

function defaultUrlPath(event: WeChatTemplateNoticeEvent): string {
  return WECHAT_TEMPLATE_NOTICE_EVENTS.find((info) => info.event === event)?.defaultUrlPath || '/'
}

function placeholdersFor(event: WeChatTemplateNoticeEvent): string[] {
  const info = WECHAT_TEMPLATE_NOTICE_EVENTS.find((item) => item.event === event)
  return [...COMMON_TEMPLATE_PLACEHOLDERS, ...(info?.placeholders || [])]
}

function urlPathInvalid(form: WeChatTemplateNoticeForm): boolean {
  return form.url_path.trim() !== '' && !isSafeSiteUrlPath(form.url_path)
}

async function sendTest(event: WeChatTemplateNoticeEvent): Promise<void> {
  testing.value = event
  delete testResults[event]
  try {
    const result = await testWeChatTemplateNotice(event)
    const ok = !result.errcode
    testResults[event] = {
      ok,
      message:
        result.message?.trim() ||
        (ok
          ? t('admin.settings.wechatConnect.templateNotices.testSent')
          : `${t('admin.settings.wechatConnect.templateNotices.testFailed')} (${result.errcode} ${result.errmsg || ''})`),
    }
  } catch (error: unknown) {
    testResults[event] = {
      ok: false,
      message: extractApiErrorMessage(error, t('admin.settings.wechatConnect.templateNotices.testFailed')),
    }
  } finally {
    testing.value = null
  }
}
</script>
