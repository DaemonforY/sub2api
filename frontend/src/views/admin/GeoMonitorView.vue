<template>
  <AppLayout>
    <div class="space-y-4" data-testid="geo-view">
      <div class="card space-y-1 p-4">
        <p class="text-sm text-gray-600 dark:text-dark-300">{{ t('admin.geo.intro') }}</p>
        <p class="text-xs text-gray-500 dark:text-dark-400">{{ t('admin.geo.introManual') }}</p>
      </div>

      <div class="flex flex-wrap gap-2" role="tablist">
        <button
          v-for="tb in TABS"
          :key="tb"
          type="button"
          role="tab"
          :data-testid="`geo-tab-${tb}`"
          :class="['btn btn-sm', tab === tb ? 'btn-primary' : 'btn-secondary']"
          @click="tab = tb"
        >
          {{ t(`admin.geo.tabs.${tb}`) }}
        </button>
      </div>

      <!-- 概览 -->
      <template v-if="tab === 'overview'">
        <div class="card flex flex-wrap items-center justify-between gap-3 p-4">
          <div class="text-sm text-gray-600 dark:text-dark-300">
            <span v-if="status.last_run_at">{{ t('admin.geo.overview.lastRun', { time: fmtTime(status.last_run_at) }) }}</span>
            <span v-else>{{ t('admin.geo.overview.neverRun') }}</span>
          </div>
          <div class="flex items-center gap-2">
            <button type="button" class="btn btn-secondary btn-sm" data-testid="geo-manual-open" @click="openManual()">{{ t('admin.geo.overview.manual') }}</button>
            <button type="button" class="btn btn-primary btn-sm" data-testid="geo-run" :disabled="status.running || starting" @click="run">
              {{ status.running ? t('admin.geo.overview.running', { done: status.done, total: status.total }) : t('admin.geo.overview.run') }}
            </button>
          </div>
        </div>

        <div v-if="summary" class="grid gap-3 sm:grid-cols-2 lg:grid-cols-4" data-testid="geo-engine-cards">
          <div class="card p-4">
            <div class="text-xs text-gray-500">{{ t('admin.geo.overview.totalRate') }}</div>
            <div class="text-2xl font-semibold text-gray-900 dark:text-white">{{ pct(summary.totals.mention_rate) }}</div>
            <div class="text-xs text-gray-400">{{ t('admin.geo.overview.totalHint', { mentioned: summary.totals.mentioned, answered: summary.totals.answered }) }}</div>
          </div>
          <div v-for="e in summary.engines" :key="e.name" class="card p-4">
            <div class="flex items-center gap-1 text-xs text-gray-500">
              <span class="font-medium text-gray-700 dark:text-dark-200">{{ e.name }}</span>
              <span v-if="!e.auto" class="rounded bg-gray-100 px-1 dark:bg-dark-700">{{ t('admin.geo.overview.manualOnly') }}</span>
              <span v-else-if="!e.enabled" class="rounded bg-gray-100 px-1 dark:bg-dark-700">{{ t('admin.geo.overview.engineOff') }}</span>
            </div>
            <div class="text-2xl font-semibold text-gray-900 dark:text-white">{{ e.answered ? pct(e.mention_rate) : '—' }}</div>
            <div class="text-xs text-gray-400">{{ t('admin.geo.overview.engineRate', { mentioned: e.mentioned, answered: e.answered }) }}</div>
          </div>
        </div>

        <div class="card p-4">
          <h3 class="font-semibold text-gray-900 dark:text-white">{{ t('admin.geo.overview.matrixTitle') }}</h3>
          <p class="mb-2 text-xs text-gray-500">{{ t('admin.geo.overview.matrixHint') }}</p>
          <p v-if="summary && !summary.engines.length" class="py-4 text-center text-sm text-gray-500">{{ t('admin.geo.overview.noEngines') }}</p>
          <div v-else-if="summary" class="overflow-x-auto">
            <table class="w-full text-sm" data-testid="geo-matrix">
              <thead>
                <tr class="text-left text-xs text-gray-500">
                  <th class="py-1 pr-3">{{ t('admin.geo.overview.question') }}</th>
                  <th v-for="e in summary.engines" :key="e.name" class="whitespace-nowrap px-2 text-center">{{ e.name }}</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="q in matrixQuestions" :key="q.id" class="border-t border-gray-100 dark:border-dark-700">
                  <td class="max-w-[360px] py-1.5 pr-3">
                    <span class="text-gray-900 dark:text-white">{{ q.question }}</span>
                    <span v-if="q.category" class="ml-1 text-xs text-gray-400">{{ q.category }}</span>
                  </td>
                  <td v-for="e in summary.engines" :key="e.name" class="px-2 text-center">
                    <button
                      v-if="cell(q.id, e.name)"
                      type="button"
                      :data-testid="`geo-cell-${q.id}-${e.name}`"
                      :class="['whitespace-nowrap text-xs hover:underline', cell(q.id, e.name)!.mentioned ? 'text-green-600 dark:text-green-400' : 'text-red-500']"
                      @click="detail = cell(q.id, e.name)"
                    >
                      {{ cell(q.id, e.name)!.mentioned ? t('admin.geo.overview.yes') : t('admin.geo.overview.no') }}
                    </button>
                    <span v-else class="text-gray-300">{{ t('admin.geo.overview.none') }}</span>
                  </td>
                </tr>
              </tbody>
            </table>
          </div>
        </div>

        <div class="card p-4">
          <h3 class="font-semibold text-gray-900 dark:text-white">{{ t('admin.geo.overview.trendTitle') }}</h3>
          <p class="mb-2 text-xs text-gray-500">{{ t('admin.geo.overview.trendHint') }}</p>
          <p v-if="!trendWeeks.length" class="py-2 text-sm text-gray-500">{{ t('admin.geo.overview.noTrend') }}</p>
          <div v-else class="overflow-x-auto">
            <table class="w-full text-sm" data-testid="geo-trend">
              <thead>
                <tr class="text-left text-xs text-gray-500">
                  <th class="py-1 pr-3">{{ t('admin.geo.overview.week') }}</th>
                  <th v-for="n in trendEngines" :key="n" class="whitespace-nowrap px-2 text-right">{{ n }}</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="w in trendWeeks" :key="w.week" class="border-t border-gray-100 dark:border-dark-700">
                  <td class="whitespace-nowrap py-1 pr-3">{{ w.week }} <span class="text-xs text-gray-400">{{ w.start }}</span></td>
                  <td v-for="n in trendEngines" :key="n" class="whitespace-nowrap px-2 text-right tabular-nums">
                    <template v-if="trend(w.week, n)">{{ pct(trend(w.week, n)!.rate) }} <span class="text-xs text-gray-400">{{ trend(w.week, n)!.mentioned }}/{{ trend(w.week, n)!.total }}</span></template>
                    <span v-else class="text-gray-300">—</span>
                  </td>
                </tr>
              </tbody>
            </table>
          </div>
        </div>
      </template>

      <!-- 问题 -->
      <template v-if="tab === 'questions'">
        <div class="flex justify-end">
          <button type="button" class="btn btn-primary btn-sm" data-testid="geo-question-new" @click="editQuestion(null)">{{ t('admin.geo.questions.add') }}</button>
        </div>
        <form v-if="qForm" class="card space-y-3 p-4" data-testid="geo-question-form" @submit.prevent="saveQuestion">
          <h3 class="font-semibold text-gray-900 dark:text-white">{{ qEditingId ? t('admin.geo.questions.editTitle') : t('admin.geo.questions.addTitle') }}</h3>
          <div>
            <label class="input-label" for="geo-q-text">{{ t('admin.geo.questions.question') }}</label>
            <input id="geo-q-text" v-model="qForm.question" class="input" maxlength="500" required :placeholder="t('admin.geo.questions.questionPlaceholder')" />
          </div>
          <div class="grid gap-3 md:grid-cols-3">
            <div>
              <label class="input-label" for="geo-q-cat">{{ t('admin.geo.questions.category') }}</label>
              <input id="geo-q-cat" v-model="qForm.category" class="input" maxlength="32" :placeholder="t('admin.geo.questions.categoryPlaceholder')" />
            </div>
            <div>
              <label class="input-label" for="geo-q-sort">{{ t('admin.geo.questions.sort') }}</label>
              <input id="geo-q-sort" v-model.number="qForm.sort" type="number" class="input" />
            </div>
            <label class="flex items-center gap-2 pt-6 text-sm">
              <input v-model="qForm.enabled" type="checkbox" />{{ t('admin.geo.questions.enabled') }}
            </label>
          </div>
          <div class="flex justify-end gap-2">
            <button type="button" class="btn btn-secondary btn-sm" @click="qForm = null">{{ t('admin.geo.common.cancel') }}</button>
            <button type="submit" class="btn btn-primary btn-sm" :disabled="saving">{{ t('admin.geo.common.save') }}</button>
          </div>
        </form>
        <div class="card overflow-x-auto p-4">
          <table class="w-full text-sm" data-testid="geo-questions-table">
            <thead>
              <tr class="text-left text-xs text-gray-500">
                <th class="py-1">{{ t('admin.geo.questions.question') }}</th>
                <th class="px-2">{{ t('admin.geo.questions.category') }}</th>
                <th class="px-2">{{ t('admin.geo.questions.enabled') }}</th>
                <th class="px-2 text-right">{{ t('admin.geo.common.actions') }}</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="q in questions" :key="q.id" class="border-t border-gray-100 dark:border-dark-700">
                <td class="py-1.5 pr-3 text-gray-900 dark:text-white">{{ q.question }}</td>
                <td class="whitespace-nowrap px-2 text-gray-500">{{ q.category }}</td>
                <td class="px-2">
                  <button type="button" :class="['text-xs', q.enabled ? 'text-green-600' : 'text-gray-400']" :data-testid="`geo-question-toggle-${q.id}`" @click="toggleQuestion(q)">
                    {{ q.enabled ? t('admin.geo.common.enabled') : t('admin.geo.common.disabled') }}
                  </button>
                </td>
                <td class="whitespace-nowrap px-2 text-right">
                  <button type="button" class="btn btn-secondary btn-sm" @click="editQuestion(q)">{{ t('admin.geo.common.edit') }}</button>
                  <button type="button" class="btn btn-secondary btn-sm ml-1 text-red-600" @click="removeQuestion(q)">{{ t('admin.geo.common.delete') }}</button>
                </td>
              </tr>
              <tr v-if="!questions.length">
                <td colspan="4" class="py-4 text-center text-gray-500">{{ t('admin.geo.questions.empty') }}</td>
              </tr>
            </tbody>
          </table>
        </div>
      </template>

      <!-- 引擎 -->
      <template v-if="tab === 'engines'">
        <div class="flex items-center justify-between gap-2">
          <p class="text-xs text-gray-500">{{ t('admin.geo.engines.costNote') }}</p>
          <button type="button" class="btn btn-primary btn-sm" data-testid="geo-engine-new" @click="editEngine(null)">{{ t('admin.geo.engines.add') }}</button>
        </div>
        <form v-if="eForm" class="card space-y-3 p-4" data-testid="geo-engine-form" @submit.prevent="saveEngine">
          <h3 class="font-semibold text-gray-900 dark:text-white">{{ eEditingId ? t('admin.geo.engines.editTitle') : t('admin.geo.engines.addTitle') }}</h3>
          <div class="space-y-1">
            <p class="text-xs text-gray-500">{{ t('admin.geo.engines.presets') }}</p>
            <div class="flex flex-wrap gap-2">
              <button v-for="p in PRESETS" :key="p.key" type="button" class="btn btn-secondary btn-sm" :data-testid="`geo-preset-${p.key}`" @click="applyPreset(p)">
                {{ t(`admin.geo.engines.preset.${p.key}`) }}
              </button>
            </div>
            <p v-if="presetHint" class="text-xs text-amber-600 dark:text-amber-400">{{ presetHint }}</p>
          </div>
          <div class="grid gap-3 md:grid-cols-2">
            <div>
              <label class="input-label" for="geo-e-name">{{ t('admin.geo.engines.name') }}</label>
              <input id="geo-e-name" v-model="eForm.name" class="input" maxlength="64" required :placeholder="t('admin.geo.engines.namePlaceholder')" />
            </div>
            <div>
              <label class="input-label" for="geo-e-model">{{ t('admin.geo.engines.model') }}</label>
              <input id="geo-e-model" v-model="eForm.model" class="input font-mono" maxlength="128" required />
            </div>
            <div>
              <label class="input-label" for="geo-e-base">{{ t('admin.geo.engines.baseUrl') }}</label>
              <input id="geo-e-base" v-model="eForm.base_url" class="input font-mono" required placeholder="https://api.example.com/v1" />
              <p class="mt-1 text-xs text-gray-400">{{ t('admin.geo.engines.baseUrlHint') }}</p>
            </div>
            <div>
              <label class="input-label" for="geo-e-key">{{ t('admin.geo.engines.apiKey') }}</label>
              <input id="geo-e-key" v-model="eForm.api_key" type="password" autocomplete="new-password" class="input font-mono" :required="!eEditingId" :placeholder="t('admin.geo.engines.apiKeyPlaceholder')" />
              <p v-if="eEditingKeyMasked" class="mt-1 text-xs text-gray-400">{{ t('admin.geo.engines.apiKeyKeep', { masked: eEditingKeyMasked }) }}</p>
            </div>
            <div class="md:col-span-2">
              <label class="input-label" for="geo-e-extra">{{ t('admin.geo.engines.extraBody') }}</label>
              <textarea id="geo-e-extra" v-model="eForm.extra_body" rows="3" class="input font-mono text-xs" placeholder='{"enable_search": true}'></textarea>
              <p class="mt-1 text-xs text-gray-400">{{ t('admin.geo.engines.extraBodyHint') }}</p>
            </div>
            <label class="flex items-center gap-2 text-sm">
              <input v-model="eForm.enabled" type="checkbox" />{{ t('admin.geo.engines.enabled') }}
            </label>
          </div>
          <div class="flex justify-end gap-2">
            <button type="button" class="btn btn-secondary btn-sm" @click="eForm = null">{{ t('admin.geo.common.cancel') }}</button>
            <button type="submit" class="btn btn-primary btn-sm" :disabled="saving" data-testid="geo-engine-save">{{ t('admin.geo.common.save') }}</button>
          </div>
        </form>
        <div class="card overflow-x-auto p-4">
          <table class="w-full text-sm" data-testid="geo-engines-table">
            <thead>
              <tr class="text-left text-xs text-gray-500">
                <th class="py-1">{{ t('admin.geo.engines.name') }}</th>
                <th class="px-2">{{ t('admin.geo.engines.baseUrl') }} / {{ t('admin.geo.engines.model') }}</th>
                <th class="px-2">{{ t('admin.geo.engines.key') }}</th>
                <th class="px-2">{{ t('admin.geo.engines.enabled') }}</th>
                <th class="px-2 text-right">{{ t('admin.geo.common.actions') }}</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="e in engines" :key="e.id" class="border-t border-gray-100 align-top dark:border-dark-700">
                <td class="py-1.5 pr-3 font-medium text-gray-900 dark:text-white">{{ e.name }}</td>
                <td class="px-2">
                  <div class="font-mono text-xs">{{ e.base_url }}</div>
                  <div class="font-mono text-xs text-gray-500">{{ e.model }}<template v-if="e.extra_body"> · {{ JSON.stringify(e.extra_body) }}</template></div>
                  <div v-if="testResults[e.id]" :class="['mt-1 text-xs', testResults[e.id].ok ? 'text-green-600' : 'text-red-500']" :data-testid="`geo-engine-test-${e.id}`">
                    {{ testResults[e.id].ok
                      ? t('admin.geo.engines.testOk', { ms: testResults[e.id].latency_ms, answer: testResults[e.id].answer || '' })
                      : t('admin.geo.engines.testFailed', { error: testResults[e.id].error || '' }) }}
                  </div>
                </td>
                <td class="whitespace-nowrap px-2 font-mono text-xs">{{ e.has_key ? e.key_masked : t('admin.geo.engines.noKey') }}</td>
                <td class="px-2 text-xs">{{ e.enabled ? t('admin.geo.common.enabled') : t('admin.geo.common.disabled') }}</td>
                <td class="whitespace-nowrap px-2 text-right">
                  <button type="button" class="btn btn-secondary btn-sm" :disabled="testing[e.id]" @click="runTest(e)">
                    {{ testing[e.id] ? t('admin.geo.engines.testing') : t('admin.geo.engines.test') }}
                  </button>
                  <button type="button" class="btn btn-secondary btn-sm ml-1" @click="editEngine(e)">{{ t('admin.geo.common.edit') }}</button>
                  <button type="button" class="btn btn-secondary btn-sm ml-1 text-red-600" @click="removeEngine(e)">{{ t('admin.geo.common.delete') }}</button>
                </td>
              </tr>
              <tr v-if="!engines.length">
                <td colspan="5" class="py-4 text-center text-gray-500">{{ t('admin.geo.engines.empty') }}</td>
              </tr>
            </tbody>
          </table>
        </div>
      </template>

      <!-- 记录 -->
      <template v-if="tab === 'checks'">
        <div class="card flex flex-wrap items-center gap-2 p-4">
          <select v-model.number="filters.question_id" class="input w-auto max-w-xs" data-testid="geo-filter-question" @change="reloadChecks">
            <option :value="0">{{ t('admin.geo.checks.allQuestions') }}</option>
            <option v-for="q in questions" :key="q.id" :value="q.id">{{ q.question }}</option>
          </select>
          <input v-model="filters.engine" class="input w-40" list="geo-engine-names" :placeholder="t('admin.geo.checks.enginePlaceholder')" @change="reloadChecks" />
          <select v-model="filters.source" class="input w-auto" @change="reloadChecks">
            <option value="">{{ t('admin.geo.checks.allSources') }}</option>
            <option value="auto">{{ t('admin.geo.detail.source.auto') }}</option>
            <option value="manual">{{ t('admin.geo.detail.source.manual') }}</option>
          </select>
          <select v-model="filters.mentioned" class="input w-auto" @change="reloadChecks">
            <option value="">{{ t('admin.geo.checks.allResults') }}</option>
            <option value="true">{{ t('admin.geo.checks.mentioned') }}</option>
            <option value="false">{{ t('admin.geo.checks.notMentioned') }}</option>
          </select>
          <span class="ml-auto text-xs text-gray-500">{{ t('admin.geo.checks.total', { total: checks.total }) }}</span>
        </div>
        <div class="card overflow-x-auto p-4">
          <table class="w-full text-sm" data-testid="geo-checks-table">
            <thead>
              <tr class="text-left text-xs text-gray-500">
                <th class="py-1">{{ t('admin.geo.checks.time') }}</th>
                <th class="px-2">{{ t('admin.geo.checks.question') }}</th>
                <th class="px-2">{{ t('admin.geo.checks.engine') }}</th>
                <th class="px-2">{{ t('admin.geo.checks.source') }}</th>
                <th class="px-2">{{ t('admin.geo.checks.result') }}</th>
                <th class="px-2 text-right">{{ t('admin.geo.common.actions') }}</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="c in checks.items" :key="c.id" class="border-t border-gray-100 align-top dark:border-dark-700">
                <td class="whitespace-nowrap py-1.5 pr-2 text-xs text-gray-500">{{ fmtTime(c.created_at) }}</td>
                <td class="max-w-[320px] px-2">{{ c.question }}</td>
                <td class="whitespace-nowrap px-2">{{ c.engine_name }}</td>
                <td class="whitespace-nowrap px-2 text-xs">{{ t(`admin.geo.detail.source.${c.source}`) }}</td>
                <td class="max-w-[320px] px-2 text-xs">
                  <span v-if="c.error" class="text-red-500">{{ t('admin.geo.checks.failed') }}：{{ c.error }}</span>
                  <span v-else-if="c.mentioned" class="text-green-600">{{ t('admin.geo.checks.mentioned') }}</span>
                  <span v-else class="text-gray-500">{{ t('admin.geo.checks.notMentioned') }}</span>
                </td>
                <td class="whitespace-nowrap px-2 text-right">
                  <button v-if="!c.error" type="button" class="btn btn-secondary btn-sm" @click="detail = c">{{ t('admin.geo.checks.showAnswer') }}</button>
                  <button type="button" class="btn btn-secondary btn-sm ml-1 text-red-600" @click="removeCheck(c)">{{ t('admin.geo.common.delete') }}</button>
                </td>
              </tr>
              <tr v-if="!checks.items.length">
                <td colspan="6" class="py-4 text-center text-gray-500">{{ t('admin.geo.checks.empty') }}</td>
              </tr>
            </tbody>
          </table>
          <div class="mt-3 flex justify-end gap-2">
            <button type="button" class="btn btn-secondary btn-sm" :disabled="offset === 0" @click="page(-1)">{{ t('admin.geo.checks.prev') }}</button>
            <button type="button" class="btn btn-secondary btn-sm" :disabled="offset + PAGE_SIZE >= checks.total" @click="page(1)">{{ t('admin.geo.checks.next') }}</button>
          </div>
        </div>
      </template>

      <!-- 设置 -->
      <form v-if="tab === 'settings' && sForm" class="card max-w-3xl space-y-4 p-4" data-testid="geo-settings-form" @submit.prevent="saveSettingsForm">
        <p class="rounded bg-gray-50 p-3 text-xs text-gray-600 dark:bg-dark-800 dark:text-dark-300">{{ t('admin.geo.settings.about') }}</p>
        <div>
          <label class="input-label" for="geo-s-schedule">{{ t('admin.geo.settings.schedule') }}</label>
          <select id="geo-s-schedule" v-model="sForm.schedule" class="input w-auto">
            <option value="off">{{ t('admin.geo.settings.scheduleOff') }}</option>
            <option value="daily">{{ t('admin.geo.settings.scheduleDaily') }}</option>
            <option value="weekly">{{ t('admin.geo.settings.scheduleWeekly') }}</option>
          </select>
          <p class="mt-1 text-xs text-gray-400">{{ t('admin.geo.settings.scheduleHint') }}</p>
        </div>
        <div>
          <label class="input-label" for="geo-s-brand">{{ t('admin.geo.settings.brand') }}</label>
          <textarea id="geo-s-brand" v-model="sForm.brand" rows="3" class="input font-mono text-sm"></textarea>
          <p class="mt-1 text-xs text-gray-400">{{ t('admin.geo.settings.brandHint') }}</p>
        </div>
        <div>
          <label class="input-label" for="geo-s-comp">{{ t('admin.geo.settings.competitors') }}</label>
          <textarea id="geo-s-comp" v-model="sForm.competitors" rows="3" class="input font-mono text-sm"></textarea>
          <p class="mt-1 text-xs text-gray-400">{{ t('admin.geo.settings.competitorsHint') }}</p>
        </div>
        <div class="flex justify-end">
          <button type="submit" class="btn btn-primary btn-sm" :disabled="saving">{{ t('admin.geo.common.save') }}</button>
        </div>
      </form>

      <datalist id="geo-engine-names">
        <option v-for="n in engineNameSuggestions" :key="n" :value="n" />
      </datalist>

      <!-- 回答详情 -->
      <BaseDialog :show="!!detail" :title="detail ? t('admin.geo.detail.title', { engine: detail.engine_name }) : ''" width="wide" close-on-click-outside @close="detail = null">
        <div v-if="detail" class="space-y-3" data-testid="geo-detail">
          <div class="text-xs text-gray-500">
            {{ t('admin.geo.detail.time') }}：{{ fmtTime(detail.created_at) }} · {{ t(`admin.geo.detail.source.${detail.source}`) }}
          </div>
          <div>
            <div class="text-xs text-gray-500">{{ t('admin.geo.detail.question') }}</div>
            <div class="text-sm text-gray-900 dark:text-white">{{ detail.question }}</div>
          </div>
          <div>
            <div class="text-xs text-gray-500">{{ t('admin.geo.detail.answer') }}</div>
            <div class="whitespace-pre-wrap rounded bg-gray-50 p-3 text-sm dark:bg-dark-800" data-testid="geo-detail-answer">{{ detail.answer }}</div>
          </div>
          <div>
            <div class="text-xs text-gray-500">{{ t('admin.geo.detail.cited') }}</div>
            <ul v-if="detail.cited_urls.length" class="space-y-0.5 text-xs" data-testid="geo-detail-cited">
              <li v-for="u in detail.cited_urls" :key="u" class="break-all">
                <a :href="u" target="_blank" rel="noopener noreferrer nofollow" :class="detail.our_urls.includes(u) ? 'font-semibold text-green-600 dark:text-green-400' : 'text-primary-600 dark:text-primary-400'">{{ u }}</a>
                <span v-if="detail.our_urls.includes(u)" class="ml-1 rounded bg-green-100 px-1 text-green-700 dark:bg-green-900/40 dark:text-green-300">{{ t('admin.geo.detail.ours') }}</span>
              </li>
            </ul>
            <p v-else class="text-xs text-gray-400">{{ t('admin.geo.detail.noCited') }}</p>
          </div>
          <div>
            <div class="text-xs text-gray-500">{{ t('admin.geo.detail.competitors') }}</div>
            <p class="text-sm">{{ detail.competitors.length ? detail.competitors.join('、') : t('admin.geo.detail.noCompetitors') }}</p>
          </div>
          <div class="flex justify-end">
            <button type="button" class="btn btn-secondary btn-sm" @click="detail = null">{{ t('admin.geo.common.close') }}</button>
          </div>
        </div>
      </BaseDialog>

      <!-- 手动录入 -->
      <BaseDialog :show="!!mForm" :title="t('admin.geo.manual.title')" width="wide" @close="mForm = null">
        <form v-if="mForm" class="space-y-3" data-testid="geo-manual-form" @submit.prevent="saveManual">
          <div>
            <label class="input-label" for="geo-m-q">{{ t('admin.geo.manual.question') }}</label>
            <select id="geo-m-q" v-model.number="mForm.question_id" class="input" required>
              <option :value="0" disabled>{{ t('admin.geo.manual.pickQuestion') }}</option>
              <option v-for="q in questions" :key="q.id" :value="q.id">{{ q.question }}</option>
            </select>
          </div>
          <div>
            <label class="input-label" for="geo-m-engine">{{ t('admin.geo.manual.engine') }}</label>
            <input id="geo-m-engine" v-model="mForm.engine_name" class="input" maxlength="64" list="geo-engine-names" required :placeholder="t('admin.geo.manual.enginePlaceholder')" />
          </div>
          <div>
            <label class="input-label" for="geo-m-answer">{{ t('admin.geo.manual.answer') }}</label>
            <textarea id="geo-m-answer" v-model="mForm.answer" rows="8" class="input text-sm" required :placeholder="t('admin.geo.manual.answerPlaceholder')"></textarea>
          </div>
          <div>
            <label class="input-label" for="geo-m-cited">{{ t('admin.geo.manual.cited') }}</label>
            <textarea id="geo-m-cited" v-model="mForm.cited_urls" rows="3" class="input font-mono text-xs" :placeholder="t('admin.geo.manual.citedPlaceholder')"></textarea>
            <p class="mt-1 text-xs text-gray-400">{{ t('admin.geo.manual.hint') }}</p>
          </div>
          <div class="flex justify-end gap-2">
            <button type="button" class="btn btn-secondary btn-sm" @click="mForm = null">{{ t('admin.geo.common.cancel') }}</button>
            <button type="submit" class="btn btn-primary btn-sm" :disabled="saving" data-testid="geo-manual-save">{{ t('admin.geo.manual.submit') }}</button>
          </div>
        </form>
      </BaseDialog>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import BaseDialog from '@/components/common/BaseDialog.vue'
import {
  addManualCheck,
  createEngine,
  createQuestion,
  deleteCheck,
  deleteEngine,
  deleteQuestion,
  getSettings,
  getStatus,
  getSummary,
  listChecks,
  listEngines,
  listQuestions,
  saveSettings,
  startRun,
  testEngine,
  updateEngine,
  updateQuestion,
  type GeoCheck,
  type GeoCheckPage,
  type GeoEngine,
  type GeoEngineInput,
  type GeoEngineTestResult,
  type GeoManualInput,
  type GeoQuestion,
  type GeoQuestionInput,
  type GeoRunStatus,
  type GeoSettings,
  type GeoSummary,
  type GeoWeekRate
} from '@/api/admin/geo'
import { useAppStore } from '@/stores/app'
import { extractApiErrorMessage } from '@/utils/apiError'

const { t } = useI18n()
const appStore = useAppStore()

const TABS = ['overview', 'questions', 'engines', 'checks', 'settings'] as const
type Tab = (typeof TABS)[number]
const PAGE_SIZE = 20
const POLL_MS = 3000

interface Preset {
  key: 'perplexity' | 'glm' | 'qwen' | 'kimi' | 'deepseek'
  name: string
  base_url: string
  model: string
  extra_body: string
}
// Suggestions only: model names change, the admin checks each provider's docs.
const PRESETS: Preset[] = [
  { key: 'perplexity', name: 'Perplexity', base_url: 'https://api.perplexity.ai', model: 'sonar', extra_body: '' },
  {
    key: 'glm',
    name: '智谱 GLM',
    base_url: 'https://open.bigmodel.cn/api/paas/v4',
    model: 'glm-4-plus',
    extra_body: '{"tools": [{"type": "web_search", "web_search": {"enable": true}}]}'
  },
  { key: 'qwen', name: '通义千问', base_url: 'https://dashscope.aliyuncs.com/compatible-mode/v1', model: 'qwen-plus', extra_body: '{"enable_search": true}' },
  { key: 'kimi', name: 'Kimi', base_url: 'https://api.moonshot.cn/v1', model: 'moonshot-v1-8k', extra_body: '' },
  { key: 'deepseek', name: 'DeepSeek', base_url: 'https://api.deepseek.com', model: 'deepseek-chat', extra_body: '' }
]
const MANUAL_ENGINES = ['豆包', '腾讯元宝', 'Kimi', 'DeepSeek', '文心一言', '通义千问', 'ChatGPT', 'Perplexity']

const tab = ref<Tab>('overview')
const saving = ref(false)
const starting = ref(false)
const summary = ref<GeoSummary | null>(null)
const status = ref<GeoRunStatus>({ running: false, run_id: '', started_at: null, last_run_at: null, done: 0, total: 0 })
const questions = ref<GeoQuestion[]>([])
const engines = ref<GeoEngine[]>([])
const checks = ref<GeoCheckPage>({ items: [], total: 0 })
const offset = ref(0)
const filters = reactive<{ question_id: number; engine: string; source: '' | 'auto' | 'manual'; mentioned: '' | 'true' | 'false' }>({
  question_id: 0,
  engine: '',
  source: '',
  mentioned: ''
})
const detail = ref<GeoCheck | null>(null)
const testing = reactive<Record<number, boolean>>({})
const testResults = reactive<Record<number, GeoEngineTestResult>>({})

const qForm = ref<GeoQuestionInput | null>(null)
const qEditingId = ref<number | null>(null)
const eForm = ref<GeoEngineInput | null>(null)
const eEditingId = ref<number | null>(null)
const eEditingKeyMasked = ref('')
const presetHint = ref('')
const mForm = ref<GeoManualInput | null>(null)
const sForm = ref<{ schedule: GeoSettings['schedule']; brand: string; competitors: string } | null>(null)

let pollTimer: ReturnType<typeof setTimeout> | null = null

function fail(e: unknown): void {
  appStore.showError(extractApiErrorMessage(e, t('admin.geo.common.loadFailed')))
}

const pct = (r: number) => `${Math.round(r * 100)}%`
const fmtTime = (s: string | null) => (s ? new Date(s).toLocaleString() : '')

// ---- overview ----

const latestMap = computed(() => {
  const m = new Map<string, GeoCheck>()
  for (const c of summary.value?.latest ?? []) m.set(`${c.question_id}|${c.engine_name}`, c)
  return m
})
const cell = (qid: number, engine: string) => latestMap.value.get(`${qid}|${engine}`) ?? null

// Enabled questions, plus disabled ones that still have results.
const matrixQuestions = computed(() => {
  const withData = new Set((summary.value?.latest ?? []).map((c) => c.question_id))
  return (summary.value?.questions ?? []).filter((q) => q.enabled || withData.has(q.id))
})

const trendEngines = computed(() => {
  const names = new Set((summary.value?.weekly ?? []).map((w) => w.engine_name))
  const ordered = (summary.value?.engines ?? []).map((e) => e.name).filter((n) => names.has(n))
  for (const n of names) if (!ordered.includes(n)) ordered.push(n)
  return ordered
})
const trendWeeks = computed(() => {
  const m = new Map<string, string>()
  for (const w of summary.value?.weekly ?? []) m.set(w.week, w.week_start)
  return [...m.entries()].sort((a, b) => b[1].localeCompare(a[1])).map(([week, start]) => ({ week, start }))
})
const trendMap = computed(() => {
  const m = new Map<string, GeoWeekRate>()
  for (const w of summary.value?.weekly ?? []) m.set(`${w.week}|${w.engine_name}`, w)
  return m
})
const trend = (week: string, engine: string) => trendMap.value.get(`${week}|${engine}`) ?? null

const engineNameSuggestions = computed(() => {
  const names = new Set<string>(MANUAL_ENGINES)
  for (const e of engines.value) names.add(e.name)
  for (const e of summary.value?.engines ?? []) names.add(e.name)
  return [...names]
})

async function loadSummary(): Promise<void> {
  try {
    summary.value = await getSummary()
  } catch (e) {
    fail(e)
  }
}

function schedulePoll(): void {
  if (pollTimer) clearTimeout(pollTimer)
  pollTimer = setTimeout(pollStatus, POLL_MS)
}

async function pollStatus(): Promise<void> {
  pollTimer = null
  const wasRunning = status.value.running
  try {
    status.value = await getStatus()
  } catch {
    // Try again on the next tick.
  }
  if (status.value.running) {
    schedulePoll()
  } else if (wasRunning) {
    appStore.showSuccess(t('admin.geo.overview.runDone'))
    await loadSummary()
  }
}

async function run(): Promise<void> {
  if (starting.value || status.value.running) return
  starting.value = true
  try {
    await startRun()
    appStore.showSuccess(t('admin.geo.overview.runStarted'))
    status.value = await getStatus()
    if (status.value.running) schedulePoll()
    else await loadSummary()
  } catch (e) {
    fail(e)
  } finally {
    starting.value = false
  }
}

// ---- manual entry ----

function openManual(): void {
  mForm.value = { question_id: 0, engine_name: '', answer: '', cited_urls: '' }
}

async function saveManual(): Promise<void> {
  if (!mForm.value || saving.value) return
  saving.value = true
  try {
    await addManualCheck({ ...mForm.value, engine_name: mForm.value.engine_name.trim() })
    appStore.showSuccess(t('admin.geo.common.saved'))
    mForm.value = null
    await loadSummary()
    if (tab.value === 'checks') await loadChecks()
  } catch (e) {
    fail(e)
  } finally {
    saving.value = false
  }
}

// ---- questions ----

async function loadQuestions(): Promise<void> {
  try {
    questions.value = await listQuestions()
  } catch (e) {
    fail(e)
  }
}

function editQuestion(q: GeoQuestion | null): void {
  qEditingId.value = q?.id ?? null
  const nextSort = questions.value.reduce((m, x) => Math.max(m, x.sort), 0) + 10
  qForm.value = q
    ? { question: q.question, category: q.category, enabled: q.enabled, sort: q.sort }
    : { question: '', category: '', enabled: true, sort: nextSort }
}

async function saveQuestion(): Promise<void> {
  if (!qForm.value || saving.value) return
  saving.value = true
  try {
    if (qEditingId.value) await updateQuestion(qEditingId.value, qForm.value)
    else await createQuestion(qForm.value)
    appStore.showSuccess(t('admin.geo.common.saved'))
    qForm.value = null
    await Promise.all([loadQuestions(), loadSummary()])
  } catch (e) {
    fail(e)
  } finally {
    saving.value = false
  }
}

async function toggleQuestion(q: GeoQuestion): Promise<void> {
  try {
    await updateQuestion(q.id, { question: q.question, category: q.category, enabled: !q.enabled, sort: q.sort })
    await Promise.all([loadQuestions(), loadSummary()])
  } catch (e) {
    fail(e)
  }
}

async function removeQuestion(q: GeoQuestion): Promise<void> {
  if (!window.confirm(t('admin.geo.questions.confirmDelete', { q: q.question }))) return
  try {
    await deleteQuestion(q.id)
    appStore.showSuccess(t('admin.geo.common.deleted'))
    await Promise.all([loadQuestions(), loadSummary()])
  } catch (e) {
    fail(e)
  }
}

// ---- engines ----

async function loadEngines(): Promise<void> {
  try {
    engines.value = await listEngines()
  } catch (e) {
    fail(e)
  }
}

function editEngine(e: GeoEngine | null): void {
  eEditingId.value = e?.id ?? null
  eEditingKeyMasked.value = e?.has_key ? e.key_masked : ''
  presetHint.value = ''
  // The stored key is never sent to the page: the field starts empty and an empty value keeps it.
  eForm.value = e
    ? { name: e.name, base_url: e.base_url, api_key: '', model: e.model, extra_body: e.extra_body ? JSON.stringify(e.extra_body, null, 2) : '', enabled: e.enabled }
    : { name: '', base_url: '', api_key: '', model: '', extra_body: '', enabled: true }
}

function applyPreset(p: Preset): void {
  if (!eForm.value) return
  if (!eForm.value.name.trim()) eForm.value.name = p.name
  eForm.value.base_url = p.base_url
  eForm.value.model = p.model
  eForm.value.extra_body = p.extra_body
  presetHint.value = t(`admin.geo.engines.presetHint.${p.key}`)
}

async function saveEngine(): Promise<void> {
  if (!eForm.value || saving.value) return
  saving.value = true
  try {
    const input: GeoEngineInput = { ...eForm.value, api_key: eForm.value.api_key.trim() }
    if (eEditingId.value) await updateEngine(eEditingId.value, input)
    else await createEngine(input)
    appStore.showSuccess(t('admin.geo.common.saved'))
    eForm.value = null
    await Promise.all([loadEngines(), loadSummary()])
  } catch (e) {
    fail(e)
  } finally {
    saving.value = false
  }
}

async function runTest(e: GeoEngine): Promise<void> {
  testing[e.id] = true
  delete testResults[e.id]
  try {
    testResults[e.id] = await testEngine(e.id)
  } catch (err) {
    fail(err)
  } finally {
    testing[e.id] = false
  }
}

async function removeEngine(e: GeoEngine): Promise<void> {
  if (!window.confirm(t('admin.geo.engines.confirmDelete', { name: e.name }))) return
  try {
    await deleteEngine(e.id)
    appStore.showSuccess(t('admin.geo.common.deleted'))
    await Promise.all([loadEngines(), loadSummary()])
  } catch (err) {
    fail(err)
  }
}

// ---- checks ----

async function loadChecks(): Promise<void> {
  try {
    checks.value = await listChecks({ ...filters, engine: filters.engine.trim(), limit: PAGE_SIZE, offset: offset.value })
  } catch (e) {
    fail(e)
  }
}

function reloadChecks(): void {
  offset.value = 0
  void loadChecks()
}

function page(dir: number): void {
  offset.value = Math.max(0, offset.value + dir * PAGE_SIZE)
  void loadChecks()
}

async function removeCheck(c: GeoCheck): Promise<void> {
  if (!window.confirm(t('admin.geo.checks.confirmDelete'))) return
  try {
    await deleteCheck(c.id)
    appStore.showSuccess(t('admin.geo.common.deleted'))
    await Promise.all([loadChecks(), loadSummary()])
  } catch (e) {
    fail(e)
  }
}

// ---- settings ----

const splitLines = (s: string) =>
  s
    .split(/[\n,，]/)
    .map((x) => x.trim())
    .filter(Boolean)

async function loadSettings(): Promise<void> {
  try {
    const s = await getSettings()
    sForm.value = { schedule: s.schedule, brand: s.brand_keywords.join('\n'), competitors: s.competitor_keywords.join('\n') }
  } catch (e) {
    fail(e)
  }
}

async function saveSettingsForm(): Promise<void> {
  if (!sForm.value || saving.value) return
  saving.value = true
  try {
    const s = await saveSettings({
      schedule: sForm.value.schedule,
      brand_keywords: splitLines(sForm.value.brand),
      competitor_keywords: splitLines(sForm.value.competitors)
    })
    sForm.value = { schedule: s.schedule, brand: s.brand_keywords.join('\n'), competitors: s.competitor_keywords.join('\n') }
    appStore.showSuccess(t('admin.geo.common.saved'))
  } catch (e) {
    fail(e)
  } finally {
    saving.value = false
  }
}

watch(tab, (v) => {
  if (v === 'checks') void loadChecks()
  if (v === 'settings' && !sForm.value) void loadSettings()
  if (v === 'engines') void loadEngines()
})

onMounted(async () => {
  await Promise.all([loadSummary(), loadQuestions(), loadEngines()])
  try {
    status.value = await getStatus()
    if (status.value.running) schedulePoll()
  } catch {
    // The page still works without the run status.
  }
})

onBeforeUnmount(() => {
  if (pollTimer) clearTimeout(pollTimer)
})
</script>
