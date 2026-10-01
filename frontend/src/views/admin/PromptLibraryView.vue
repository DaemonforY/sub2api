<template>
  <AppLayout>
    <div class="space-y-5" data-testid="prompt-library-admin">
      <div class="grid grid-cols-2 gap-3 lg:grid-cols-4">
        <div v-for="card in statCards" :key="card.key" class="card p-4">
          <div class="text-xs text-gray-500 dark:text-dark-400">{{ card.label }}</div>
          <div class="mt-1 text-2xl font-semibold text-gray-900 dark:text-white">{{ card.value.toLocaleString() }}</div>
        </div>
      </div>

      <div class="flex flex-wrap items-end justify-between gap-3 border-b border-gray-200 dark:border-dark-700">
        <div class="flex gap-1">
          <button
            v-for="item in tabs"
            :key="item.key"
            type="button"
            :data-testid="`prompt-tab-${item.key}`"
            :class="[
              '-mb-px border-b-2 px-4 py-2 text-sm font-medium',
              tab === item.key ? 'border-primary-500 text-primary-600 dark:text-primary-400' : 'border-transparent text-gray-500 hover:text-gray-800 dark:hover:text-gray-200'
            ]"
            @click="switchTab(item.key)"
          >
            {{ item.label }}
            <span v-if="item.badge" class="ml-1 rounded-full bg-amber-100 px-1.5 text-xs text-amber-700 dark:bg-amber-900/40 dark:text-amber-300">{{ item.badge }}</span>
          </button>
        </div>
        <button v-if="tab === 'items'" type="button" class="btn btn-primary mb-2" @click="openEditor(null)">
          <Icon name="plus" size="sm" />
          <span>{{ t('admin.promptLibrary.actions.create') }}</span>
        </button>
      </div>

      <!-- Entries / review -->
      <section v-if="tab !== 'sources'" class="space-y-4">
        <div class="flex flex-wrap items-end gap-3">
          <div class="min-w-[220px] flex-1">
            <input v-model="filters.q" type="search" class="input" :placeholder="t('admin.promptLibrary.filters.search')" data-testid="prompt-search" @keydown.enter="reload" />
          </div>
          <div v-if="tab === 'items'" class="w-36">
            <Select v-model="filters.status" :options="statusOptions" @change="reload" />
          </div>
          <div class="w-44">
            <Select v-model="filters.source" :options="sourceOptions" @change="reload" />
          </div>
          <div class="w-40">
            <Select v-model="filters.scene" :options="sceneFilterOptions" @change="reload" />
          </div>
          <div v-if="tab === 'items'" class="w-36">
            <Select v-model="filters.curated" :options="curatedOptions" @change="reload" />
          </div>
          <div class="w-32">
            <Select v-model="filters.sort" :options="sortOptions" @change="reload" />
          </div>
          <button type="button" class="btn btn-secondary" @click="reload">
            <Icon name="search" size="sm" />
          </button>
        </div>

        <div
          v-if="selected.length"
          class="sticky top-0 z-10 flex flex-wrap items-center gap-2 rounded-xl border border-primary-200 bg-primary-50/95 px-4 py-2 text-sm backdrop-blur dark:border-primary-800 dark:bg-dark-800/95"
          data-testid="prompt-batch-bar"
        >
          <span class="font-medium text-primary-700 dark:text-primary-300">{{ t('admin.promptLibrary.batch.selected', { n: selected.length }) }}</span>
          <button type="button" class="btn btn-secondary btn-sm" @click="openBatchScenes">{{ t('admin.promptLibrary.batch.scenesTitle') }}</button>
          <button type="button" class="btn btn-secondary btn-sm" @click="openBatchTags">{{ t('admin.promptLibrary.batch.tagsTitle') }}</button>
          <button type="button" class="btn btn-secondary btn-sm" @click="runBatch({ action: 'mark_reviewed' })">{{ t('admin.promptLibrary.actions.markReviewed') }}</button>
          <button type="button" class="btn btn-secondary btn-sm" @click="runBatch({ action: 'set_featured', featured: true })">{{ t('admin.promptLibrary.actions.feature') }}</button>
          <button type="button" class="btn btn-secondary btn-sm" @click="runBatch({ action: 'set_status', status: 'active' })">
            {{ tab === 'review' ? t('admin.promptLibrary.actions.approve') : t('admin.promptLibrary.actions.show') }}
          </button>
          <button type="button" class="btn btn-secondary btn-sm text-red-600" @click="runBatch({ action: 'set_status', status: 'hidden' })">{{ t('admin.promptLibrary.actions.hide') }}</button>
          <span class="flex-1" />
          <button type="button" class="text-gray-500 hover:text-gray-800" @click="selected = []">{{ t('admin.promptLibrary.actions.clearSelection') }}</button>
        </div>

        <div class="flex items-center justify-between text-sm text-gray-500 dark:text-dark-400">
          <span>{{ t('pagination.of') }} {{ total.toLocaleString() }}</span>
          <button v-if="items.length" type="button" class="hover:text-primary-600" @click="toggleSelectPage">
            {{ allOnPageSelected ? t('admin.promptLibrary.actions.clearSelection') : t('admin.promptLibrary.actions.selectAll') }}
          </button>
        </div>

        <div v-if="loading && !items.length" class="card py-16 text-center text-sm text-gray-500">{{ t('common.loading') }}</div>
        <div v-else-if="!items.length" class="card py-16 text-center text-sm text-gray-500">
          {{ tab === 'review' ? t('admin.promptLibrary.review.empty') : t('admin.promptLibrary.empty') }}
        </div>
        <div v-else class="grid gap-4 sm:grid-cols-2 lg:grid-cols-3 2xl:grid-cols-4" data-testid="prompt-admin-grid">
          <article
            v-for="item in items"
            :key="item.id"
            :class="['card flex flex-col overflow-hidden transition', isSelected(item.id) ? 'ring-2 ring-primary-500' : '']"
            :data-testid="`prompt-admin-card-${item.id}`"
          >
            <div class="relative aspect-[4/3] bg-gray-100 dark:bg-dark-900">
              <img
                v-if="item.cover_url && !brokenCovers.has(item.id)"
                :src="coverSrc(item.cover_url)"
                alt=""
                loading="lazy"
                referrerpolicy="no-referrer"
                class="h-full w-full cursor-pointer object-cover"
                @click="openEditor(item)"
                @error="brokenCovers.add(item.id)"
              />
              <button v-else type="button" class="flex h-full w-full items-center justify-center text-xs text-gray-400" @click="openEditor(item)">
                {{ t('admin.promptLibrary.card.noCover') }}
              </button>
              <label class="absolute left-2 top-2 flex h-7 w-7 cursor-pointer items-center justify-center rounded-md bg-white/90 shadow dark:bg-dark-800/90">
                <input type="checkbox" class="h-4 w-4" :checked="isSelected(item.id)" @change="toggleSelected(item.id)" />
              </label>
              <span :class="['absolute right-2 top-2 rounded-full px-2 py-0.5 text-xs font-medium', statusBadge(item.status)]">{{ t(`admin.promptLibrary.status.${item.status}`) }}</span>
              <span v-if="item.featured" class="absolute bottom-2 left-2 rounded-full bg-amber-500 px-2 py-0.5 text-xs font-medium text-white">★ {{ t('admin.promptLibrary.card.featured') }}</span>
            </div>
            <div class="flex flex-1 flex-col gap-2 p-3">
              <button type="button" class="line-clamp-1 text-left font-medium text-gray-900 hover:text-primary-600 dark:text-white" :title="item.title" @click="openEditor(item)">{{ item.title }}</button>
              <p v-if="item.original_title" class="-mt-1.5 line-clamp-1 text-xs text-gray-400" :title="item.original_title">{{ item.original_title }}</p>
              <p class="line-clamp-2 text-xs leading-5 text-gray-500 dark:text-dark-400" :title="item.prompt">{{ item.prompt }}</p>
              <div class="flex flex-wrap gap-1">
                <span
                  v-for="(scene, i) in item.scenes"
                  :key="scene"
                  :class="['rounded px-1.5 py-0.5 text-xs', i === 0 ? 'bg-primary-100 font-medium text-primary-700 dark:bg-primary-900/40 dark:text-primary-300' : 'bg-gray-100 text-gray-600 dark:bg-dark-700 dark:text-dark-300']"
                >
                  {{ sceneLabel(scene) }}
                </span>
                <span v-for="tag in item.tags" :key="`t-${tag}`" class="rounded bg-emerald-50 px-1.5 py-0.5 text-xs text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-300">#{{ tag }}</span>
                <span v-for="flag in item.auto_flags || []" :key="`f-${flag}`" class="rounded bg-red-50 px-1.5 py-0.5 text-xs text-red-600 dark:bg-red-900/30 dark:text-red-300">{{ t(`admin.promptLibrary.flags.${flag}`) }}</span>
              </div>
              <div v-if="item.owner_email" class="truncate text-xs text-gray-500">{{ t('admin.promptLibrary.card.owner') }}：{{ item.owner_email }}</div>
              <div v-if="item.review_note && item.review_note !== 'source_disabled'" class="truncate text-xs text-red-500" :title="item.review_note">{{ item.review_note }}</div>
              <div class="mt-auto flex flex-wrap items-center gap-x-2 gap-y-1 text-xs text-gray-500 dark:text-dark-400">
                <span class="truncate">{{ item.source_name }}</span>
                <span>· {{ t('admin.promptLibrary.card.uses', { n: item.use_count }) }}</span>
                <span v-if="item.favorite_count">· {{ t('admin.promptLibrary.card.favorites', { n: item.favorite_count }) }}</span>
                <span v-if="item.visibility === 'private'">· {{ t('admin.promptLibrary.card.private') }}</span>
                <span v-if="item.curated" class="text-emerald-600">· ✓ {{ t('admin.promptLibrary.card.curated') }}</span>
              </div>
              <div class="flex flex-wrap gap-3 border-t border-gray-100 pt-2 text-xs font-medium dark:border-dark-700">
                <button type="button" class="text-primary-600 hover:text-primary-700" @click="openEditor(item)">{{ t('admin.promptLibrary.actions.edit') }}</button>
                <template v-if="item.status === 'pending'">
                  <button type="button" class="text-emerald-600" @click="quickStatus(item, 'active')">{{ t('admin.promptLibrary.actions.approve') }}</button>
                  <button type="button" class="text-red-500" @click="openReject(item)">{{ t('admin.promptLibrary.actions.reject') }}</button>
                </template>
                <button v-else-if="item.status === 'active'" type="button" class="text-gray-500 hover:text-red-600" @click="quickStatus(item, 'hidden')">{{ t('admin.promptLibrary.actions.hide') }}</button>
                <button v-else type="button" class="text-emerald-600" @click="quickStatus(item, 'active')">{{ t('admin.promptLibrary.actions.show') }}</button>
                <button type="button" class="text-amber-600" @click="quickFeature(item)">
                  {{ item.featured ? t('admin.promptLibrary.actions.unfeature') : t('admin.promptLibrary.actions.feature') }}
                </button>
                <button v-if="!item.curated" type="button" class="text-gray-500 hover:text-emerald-600" @click="runBatch({ action: 'mark_reviewed' }, [item.id])">
                  {{ t('admin.promptLibrary.actions.markReviewed') }}
                </button>
              </div>
            </div>
          </article>
        </div>

        <Pagination v-if="total > pageSize" :total="total" :page="page" :page-size="pageSize" @update:page="onPage" @update:page-size="onPageSize" />
      </section>

      <!-- Sources -->
      <section v-else class="space-y-3">
        <div class="card space-y-3 p-5" data-testid="prompt-translation">
          <div class="flex flex-wrap items-start justify-between gap-3">
            <div>
              <h3 class="font-semibold text-gray-900 dark:text-white">{{ t('admin.promptLibrary.translation.title') }}</h3>
              <p class="mt-1 max-w-3xl text-sm text-gray-500 dark:text-dark-400">{{ t('admin.promptLibrary.translation.hint') }}</p>
            </div>
            <div class="text-right text-sm">
              <div class="text-gray-900 dark:text-white">{{ t('admin.promptLibrary.translation.left', { n: (translation?.untranslated || 0).toLocaleString() }) }}</div>
              <div v-if="translation?.running" class="text-xs text-primary-600">{{ t('admin.promptLibrary.translation.running') }}</div>
              <div v-else-if="translation?.last_run_at" class="text-xs text-gray-500">
                {{ t('admin.promptLibrary.translation.lastRun', { time: formatTime(translation.last_run_at), n: translation.last_translated }) }}
              </div>
              <div v-if="translation?.last_error" class="mt-1 max-w-sm truncate text-xs text-red-500" :title="translation.last_error">{{ translation.last_error }}</div>
            </div>
          </div>
          <div class="grid gap-3 md:grid-cols-3">
            <div>
              <label class="input-label">{{ t('admin.promptLibrary.translation.baseUrl') }}</label>
              <input v-model.trim="translationForm.base_url" type="url" class="input" placeholder="http://127.0.0.1:8080/v1" />
            </div>
            <div>
              <label class="input-label">{{ t('admin.promptLibrary.translation.model') }}</label>
              <input v-model.trim="translationForm.model" type="text" class="input" placeholder="gpt-5.4-mini" />
            </div>
            <div>
              <label class="input-label">{{ t('admin.promptLibrary.translation.apiKey') }}</label>
              <input
                v-model.trim="translationForm.api_key"
                type="password"
                autocomplete="new-password"
                class="input"
                :placeholder="translation?.api_key_configured ? t('admin.promptLibrary.translation.apiKeyKeep') : t('admin.promptLibrary.translation.apiKeyPlaceholder')"
              />
            </div>
          </div>
          <div class="flex flex-wrap gap-2">
            <button type="button" class="btn btn-secondary" :disabled="translationSaving" @click="saveTranslation">{{ t('common.save') }}</button>
            <button
              type="button"
              class="btn btn-primary"
              :disabled="translation?.running || !translation?.api_key_configured || !translation?.untranslated"
              @click="runTranslation"
            >
              {{ t('admin.promptLibrary.translation.run') }}
            </button>
          </div>
        </div>

        <div class="flex items-center justify-between">
          <p class="text-sm text-gray-500 dark:text-dark-400">{{ t('admin.promptLibrary.sources.hint') }}</p>
          <button type="button" class="btn btn-secondary" @click="loadSources">
            <Icon name="refresh" size="sm" />
            <span>{{ t('admin.promptLibrary.actions.refresh') }}</span>
          </button>
        </div>
        <div class="card overflow-x-auto">
          <table class="w-full min-w-[760px] text-sm">
            <thead class="border-b border-gray-200 text-left text-xs text-gray-500 dark:border-dark-700">
              <tr>
                <th class="px-4 py-3">{{ t('admin.promptLibrary.sources.name') }}</th>
                <th class="px-4 py-3">{{ t('admin.promptLibrary.sources.enabled') }}</th>
                <th class="px-4 py-3 text-right">{{ t('admin.promptLibrary.sources.items') }}</th>
                <th class="px-4 py-3 text-right">{{ t('admin.promptLibrary.sources.active') }}</th>
                <th class="px-4 py-3">{{ t('admin.promptLibrary.sources.lastSynced') }}</th>
                <th class="px-4 py-3"></th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="src in sources" :key="src.id" class="border-b border-gray-100 last:border-0 dark:border-dark-700" :data-testid="`prompt-source-${src.id}`">
                <td class="px-4 py-3">
                  <a :href="src.homepage || src.url" target="_blank" rel="noopener noreferrer" class="font-medium text-gray-900 hover:text-primary-600 dark:text-white">{{ src.name }}</a>
                  <div class="text-xs text-gray-400">{{ src.id }} · {{ src.format }}</div>
                  <div v-if="src.last_error" class="mt-1 max-w-md truncate text-xs text-red-500" :title="src.last_error">{{ src.last_error }}</div>
                </td>
                <td class="px-4 py-3"><Toggle :model-value="src.enabled" @update:model-value="(v: boolean) => setSourceEnabled(src, v)" /></td>
                <td class="px-4 py-3 text-right tabular-nums">{{ src.item_count.toLocaleString() }}</td>
                <td class="px-4 py-3 text-right tabular-nums">{{ src.active_count.toLocaleString() }}</td>
                <td class="px-4 py-3 text-xs text-gray-500">{{ src.last_synced_at ? formatTime(src.last_synced_at) : t('admin.promptLibrary.sources.never') }}</td>
                <td class="px-4 py-3 text-right">
                  <button type="button" class="btn btn-secondary btn-sm" :disabled="src.syncing" @click="syncSource(src)">
                    {{ src.syncing ? t('admin.promptLibrary.actions.syncing') : t('admin.promptLibrary.actions.sync') }}
                  </button>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </section>
    </div>

    <!-- Editor -->
    <BaseDialog :show="editorOpen" :title="editingId ? t('admin.promptLibrary.editor.title') : t('admin.promptLibrary.editor.createTitle')" width="wide" @close="editorOpen = false">
      <div class="grid gap-5 md:grid-cols-[220px_minmax(0,1fr)]">
        <div class="space-y-3">
          <div class="aspect-[4/3] overflow-hidden rounded-lg bg-gray-100 dark:bg-dark-900">
            <img v-if="form.cover_url" :src="coverSrc(form.cover_url)" alt="" referrerpolicy="no-referrer" class="h-full w-full object-cover" />
          </div>
          <div>
            <label class="input-label">{{ t('admin.promptLibrary.editor.fieldCover') }}</label>
            <input v-model.trim="form.cover_url" type="url" class="input" />
          </div>
          <div v-if="editingItem" class="space-y-1 text-xs text-gray-500">
            <p>{{ t('admin.promptLibrary.editor.sourceInfo', { source: editingItem.source_name, tags: editingItem.source_tags.join('、') || '—' }) }}</p>
            <p>{{ t('admin.promptLibrary.card.uses', { n: editingItem.use_count }) }} · {{ t('admin.promptLibrary.card.favorites', { n: editingItem.favorite_count }) }}</p>
            <p v-if="editingItem.owner_email">{{ t('admin.promptLibrary.card.owner') }}：{{ editingItem.owner_email }}</p>
            <a v-if="editingItem.source_url" :href="editingItem.source_url" target="_blank" rel="noopener noreferrer" class="text-primary-600">{{ t('admin.promptLibrary.editor.openSource') }}</a>
          </div>
        </div>
        <div class="min-w-0 space-y-4">
          <div>
            <label class="input-label">{{ t('admin.promptLibrary.editor.fieldTitle') }}</label>
            <input v-model.trim="form.title" type="text" maxlength="80" class="input" data-testid="prompt-editor-title" />
            <p v-if="editingItem?.original_title" class="mt-1 text-xs text-gray-500">{{ t('admin.promptLibrary.editor.originalTitle', { title: editingItem.original_title }) }}</p>
          </div>
          <div>
            <label class="input-label">{{ t('admin.promptLibrary.editor.fieldScenes') }}</label>
            <div class="flex flex-wrap gap-1.5" data-testid="prompt-editor-scenes">
              <button
                v-for="scene in PROMPT_SCENES"
                :key="scene"
                type="button"
                :data-testid="`prompt-editor-scene-${scene}`"
                :class="[
                  'rounded-full border px-2.5 py-1 text-xs transition',
                  form.scenes.includes(scene)
                    ? form.scenes[0] === scene
                      ? 'border-primary-600 bg-primary-600 text-white'
                      : 'border-primary-300 bg-primary-50 text-primary-700 dark:bg-primary-900/30 dark:text-primary-300'
                    : 'border-gray-200 text-gray-600 hover:border-primary-300 dark:border-dark-600 dark:text-dark-300'
                ]"
                @click="toggleScene(scene)"
              >
                <span v-if="form.scenes.includes(scene)" class="mr-0.5 opacity-70">{{ form.scenes.indexOf(scene) + 1 }}</span>
                {{ sceneLabel(scene) }}
              </button>
            </div>
            <div v-if="form.scenes.length > 1" class="mt-1.5 flex flex-wrap items-center gap-2 text-xs text-gray-500">
              <span>{{ t('admin.promptLibrary.editor.fieldScenes').split('（')[0] }}：</span>
              <button v-for="scene in form.scenes.slice(1)" :key="`p-${scene}`" type="button" class="text-primary-600 hover:underline" @click="makePrimary(scene)">↑ {{ sceneLabel(scene) }}</button>
            </div>
          </div>
          <div>
            <label class="input-label">{{ t('admin.promptLibrary.editor.fieldTags') }}</label>
            <input v-model="tagsText" type="text" class="input" :placeholder="t('admin.promptLibrary.batch.tagsPlaceholder')" data-testid="prompt-editor-tags" />
            <div v-if="knownTags.length" class="mt-1.5 flex flex-wrap gap-1">
              <button v-for="tag in suggestedTags" :key="tag" type="button" class="rounded bg-gray-100 px-1.5 py-0.5 text-xs text-gray-600 hover:bg-emerald-50 hover:text-emerald-700 dark:bg-dark-700 dark:text-dark-300" @click="addTag(tag)">+{{ tag }}</button>
            </div>
          </div>
          <div class="grid gap-3 sm:grid-cols-3">
            <div>
              <label class="input-label">{{ t('admin.promptLibrary.editor.fieldStatus') }}</label>
              <Select v-model="form.status" :options="editorStatusOptions" />
            </div>
            <div>
              <label class="input-label">{{ t('admin.promptLibrary.editor.fieldModel') }}</label>
              <Select v-model="form.model" :options="modelOptions" />
            </div>
            <div>
              <label class="input-label">{{ t('admin.promptLibrary.editor.fieldKind') }}</label>
              <Select v-model="form.kind" :options="kindOptions" />
            </div>
          </div>
          <div class="flex flex-wrap gap-x-6 gap-y-2 text-sm">
            <label class="flex items-center gap-2"><input v-model="form.featured" type="checkbox" class="h-4 w-4" />{{ t('admin.promptLibrary.editor.fieldFeatured') }}</label>
            <label class="flex items-center gap-2"><input v-model="form.needs_reference" type="checkbox" class="h-4 w-4" />{{ t('admin.promptLibrary.editor.fieldNeedsReference') }}</label>
            <label v-if="editingItem?.source_id === 'user'" class="flex items-center gap-2">
              <input :checked="form.visibility === 'public'" type="checkbox" class="h-4 w-4" @change="form.visibility = form.visibility === 'public' ? 'private' : 'public'" />
              {{ t('admin.promptLibrary.editor.visibilityPublic') }}
            </label>
          </div>
          <div>
            <label class="input-label">{{ t('admin.promptLibrary.editor.fieldPrompt') }}</label>
            <textarea v-model="form.prompt" rows="8" class="input font-mono text-xs leading-5" />
          </div>
          <div>
            <label class="input-label">{{ t('admin.promptLibrary.editor.fieldDescription') }}</label>
            <textarea v-model="form.description" rows="2" class="input text-sm" />
          </div>
          <div v-if="form.status === 'rejected' || form.status === 'hidden' || editingItem?.source_id === 'user'">
            <label class="input-label">{{ t('admin.promptLibrary.editor.fieldReviewNote') }}</label>
            <input v-model.trim="form.review_note" type="text" maxlength="200" class="input" />
          </div>
          <p class="text-xs text-gray-500">{{ t('admin.promptLibrary.editor.curatedHint') }}</p>
        </div>
      </div>
      <template #footer>
        <div class="flex justify-between gap-3">
          <button
            v-if="editingItem && (editingItem.source_id === 'user' || editingItem.source_id === 'official')"
            type="button"
            class="btn btn-secondary text-red-600"
            @click="confirmDelete = editingItem"
          >
            {{ t('admin.promptLibrary.actions.delete') }}
          </button>
          <span v-else />
          <div class="flex gap-3">
            <button type="button" class="btn btn-secondary" @click="editorOpen = false">{{ t('common.cancel') }}</button>
            <button type="button" class="btn btn-primary" :disabled="saving" data-testid="prompt-editor-save" @click="save">{{ t('common.save') }}</button>
          </div>
        </div>
      </template>
    </BaseDialog>

    <!-- Batch scenes -->
    <BaseDialog :show="batchScenesOpen" :title="t('admin.promptLibrary.batch.scenesTitle')" @close="batchScenesOpen = false">
      <div class="space-y-4">
        <div class="flex gap-2">
          <button
            v-for="mode in sceneModes"
            :key="mode.value"
            type="button"
            :class="['rounded-lg border px-3 py-1.5 text-sm', batchSceneMode === mode.value ? 'border-primary-500 bg-primary-50 text-primary-700 dark:bg-primary-900/30' : 'border-gray-200 dark:border-dark-600']"
            @click="batchSceneMode = mode.value"
          >
            {{ mode.label }}
          </button>
        </div>
        <div class="flex flex-wrap gap-1.5">
          <button
            v-for="scene in PROMPT_SCENES"
            :key="scene"
            type="button"
            :class="['rounded-full border px-2.5 py-1 text-xs', batchScenes.includes(scene) ? 'border-primary-600 bg-primary-600 text-white' : 'border-gray-200 text-gray-600 dark:border-dark-600 dark:text-dark-300']"
            @click="batchScenes = batchScenes.includes(scene) ? batchScenes.filter((s) => s !== scene) : [...batchScenes, scene]"
          >
            {{ sceneLabel(scene) }}
          </button>
        </div>
      </div>
      <template #footer>
        <div class="flex justify-end gap-3">
          <button type="button" class="btn btn-secondary" @click="batchScenesOpen = false">{{ t('common.cancel') }}</button>
          <button type="button" class="btn btn-primary" :disabled="!batchScenes.length" @click="applyBatchScenes">{{ t('common.confirm') }}</button>
        </div>
      </template>
    </BaseDialog>

    <!-- Batch tags -->
    <BaseDialog :show="batchTagsOpen" :title="t('admin.promptLibrary.batch.tagsTitle')" @close="batchTagsOpen = false">
      <div class="space-y-4">
        <div class="flex gap-2">
          <button
            v-for="mode in tagModes"
            :key="mode.value"
            type="button"
            :class="['rounded-lg border px-3 py-1.5 text-sm', batchTagMode === mode.value ? 'border-primary-500 bg-primary-50 text-primary-700 dark:bg-primary-900/30' : 'border-gray-200 dark:border-dark-600']"
            @click="batchTagMode = mode.value"
          >
            {{ mode.label }}
          </button>
        </div>
        <input v-model="batchTagsText" type="text" class="input" :placeholder="t('admin.promptLibrary.batch.tagsPlaceholder')" />
        <div v-if="knownTags.length" class="flex flex-wrap gap-1">
          <button
            v-for="tag in knownTags.slice(0, 30)"
            :key="tag.tag"
            type="button"
            class="rounded bg-gray-100 px-1.5 py-0.5 text-xs text-gray-600 hover:bg-emerald-50 dark:bg-dark-700 dark:text-dark-300"
            @click="batchTagsText = [...parseTags(batchTagsText), tag.tag].join(' ')"
          >
            {{ tag.tag }} <span class="opacity-50">{{ tag.count }}</span>
          </button>
        </div>
      </div>
      <template #footer>
        <div class="flex justify-end gap-3">
          <button type="button" class="btn btn-secondary" @click="batchTagsOpen = false">{{ t('common.cancel') }}</button>
          <button type="button" class="btn btn-primary" :disabled="!parseTags(batchTagsText).length" @click="applyBatchTags">{{ t('common.confirm') }}</button>
        </div>
      </template>
    </BaseDialog>

    <!-- Reject -->
    <BaseDialog :show="!!rejecting" :title="t('admin.promptLibrary.review.rejectTitle')" @close="rejecting = null">
      <input v-model.trim="rejectNote" type="text" maxlength="200" class="input" :placeholder="t('admin.promptLibrary.review.rejectPlaceholder')" />
      <template #footer>
        <div class="flex justify-end gap-3">
          <button type="button" class="btn btn-secondary" @click="rejecting = null">{{ t('common.cancel') }}</button>
          <button type="button" class="btn btn-primary" @click="confirmReject">{{ t('admin.promptLibrary.actions.reject') }}</button>
        </div>
      </template>
    </BaseDialog>

    <ConfirmDialog
      :show="!!confirmDelete"
      :title="t('admin.promptLibrary.actions.delete')"
      :message="t('admin.promptLibrary.deleteConfirm')"
      danger
      @confirm="runDelete"
      @cancel="confirmDelete = null"
    />
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import Select from '@/components/common/Select.vue'
import BaseDialog from '@/components/common/BaseDialog.vue'
import ConfirmDialog from '@/components/common/ConfirmDialog.vue'
import Pagination from '@/components/common/Pagination.vue'
import Toggle from '@/components/common/Toggle.vue'
import Icon from '@/components/icons/Icon.vue'
import { useAppStore } from '@/stores'
import { adminAPI } from '@/api/admin'
import { canvasProxiedImageUrl } from '@/constants/crossSites'
import {
  PROMPT_MODELS,
  PROMPT_SCENES,
  PROMPT_STATUSES,
  parseTags,
  toAdminInput,
  type PromptAdminInput,
  type PromptBatchInput,
  type PromptItem,
  type PromptLibraryStats,
  type PromptSource,
  type PromptStatus,
  type PromptTranslationStatus
} from '@/api/admin/promptLibrary'

type Tab = 'items' | 'review' | 'sources'

const { t, te } = useI18n()
const appStore = useAppStore()

const tab = ref<Tab>('items')
const loading = ref(false)
const items = ref<PromptItem[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(24)
const stats = ref<PromptLibraryStats | null>(null)
const sources = ref<PromptSource[]>([])
const knownTags = ref<{ tag: string; count: number }[]>([])
const brokenCovers = reactive(new Set<number>())
const selected = ref<number[]>([])
const filters = reactive({ q: '', status: '', source: '', scene: '', curated: '', sort: 'latest' })

const tabs = computed(() => [
  { key: 'items' as Tab, label: t('admin.promptLibrary.tabs.items'), badge: 0 },
  { key: 'review' as Tab, label: t('admin.promptLibrary.tabs.review'), badge: stats.value?.pending_user || 0 },
  { key: 'sources' as Tab, label: t('admin.promptLibrary.tabs.sources'), badge: 0 }
])

const statCards = computed(() => [
  { key: 'active', label: t('admin.promptLibrary.stats.active'), value: stats.value?.status_counts?.active || 0 },
  { key: 'pending', label: t('admin.promptLibrary.stats.pending'), value: stats.value?.pending_user || 0 },
  { key: 'uncurated', label: t('admin.promptLibrary.stats.uncurated'), value: stats.value?.uncurated || 0 },
  { key: 'uses', label: t('admin.promptLibrary.stats.uses'), value: stats.value?.total_uses || 0 }
])

function sceneLabel(scene: string) {
  return te(`admin.promptLibrary.scenes.${scene}`) ? t(`admin.promptLibrary.scenes.${scene}`) : scene
}

const statusOptions = computed(() => [{ value: '', label: t('admin.promptLibrary.filters.allStatus') }, ...PROMPT_STATUSES.map((s) => ({ value: s, label: t(`admin.promptLibrary.status.${s}`) }))])
const editorStatusOptions = computed(() => PROMPT_STATUSES.map((s) => ({ value: s, label: t(`admin.promptLibrary.status.${s}`) })))
const sourceOptions = computed(() => [
  { value: '', label: t('admin.promptLibrary.filters.allSources') },
  ...sources.value.map((s) => ({ value: s.id, label: s.name })),
  { value: 'user', label: '社区分享' },
  { value: 'official', label: 'HiveGPT 精选' }
])
const sceneFilterOptions = computed(() => [{ value: '', label: t('admin.promptLibrary.filters.allScenes') }, ...PROMPT_SCENES.map((s) => ({ value: s, label: sceneLabel(s) }))])
const curatedOptions = computed(() => [
  { value: '', label: t('admin.promptLibrary.filters.allCurated') },
  { value: 'true', label: t('admin.promptLibrary.filters.curatedYes') },
  { value: 'false', label: t('admin.promptLibrary.filters.curatedNo') }
])
const sortOptions = computed(() => [
  { value: 'latest', label: t('admin.promptLibrary.filters.sortLatest') },
  { value: 'popular', label: t('admin.promptLibrary.filters.sortPopular') },
  { value: 'recommended', label: t('admin.promptLibrary.filters.sortRecommended') }
])
const modelOptions = computed(() => PROMPT_MODELS.map((m) => ({ value: m, label: t(`admin.promptLibrary.models.${m}`) })))
const kindOptions = computed(() => [
  { value: 'image', label: t('admin.promptLibrary.editor.kindImage') },
  { value: 'video', label: t('admin.promptLibrary.editor.kindVideo') }
])

function statusBadge(status: PromptStatus) {
  switch (status) {
    case 'active':
      return 'bg-emerald-500/90 text-white'
    case 'pending':
      return 'bg-amber-500/90 text-white'
    case 'rejected':
      return 'bg-red-500/90 text-white'
    default:
      return 'bg-gray-700/80 text-white'
  }
}

function formatTime(value: string) {
  const d = new Date(value)
  return Number.isNaN(d.getTime()) ? '' : d.toLocaleString()
}

function errorMessage(err: unknown, fallback: string) {
  return (err as { message?: string })?.message || fallback
}

// ---- listing ----
async function loadItems() {
  loading.value = true
  try {
    const res = await adminAPI.promptLibrary.list({
      q: filters.q.trim() || undefined,
      status: tab.value === 'review' ? 'pending' : filters.status || undefined,
      source: filters.source || undefined,
      scene: filters.scene || undefined,
      curated: tab.value === 'review' ? undefined : filters.curated || undefined,
      sort: filters.sort,
      page: page.value,
      page_size: pageSize.value
    })
    items.value = res.items
    total.value = res.total
  } catch (err) {
    appStore.showError(errorMessage(err, t('admin.promptLibrary.loadFailed')))
  } finally {
    loading.value = false
  }
}

async function loadStats() {
  try {
    stats.value = await adminAPI.promptLibrary.stats()
  } catch {
    // stats are decorative
  }
}

async function loadSources() {
  try {
    sources.value = await adminAPI.promptLibrary.sources()
  } catch (err) {
    appStore.showError(errorMessage(err, t('admin.promptLibrary.loadFailed')))
  }
  scheduleSourcePoll()
}

async function loadTags() {
  try {
    knownTags.value = await adminAPI.promptLibrary.tags()
  } catch {
    knownTags.value = []
  }
}

function reload() {
  page.value = 1
  selected.value = []
  void loadItems()
}

function onPage(p: number) {
  page.value = p
  void loadItems()
}

function onPageSize(size: number) {
  pageSize.value = size
  reload()
}

function switchTab(next: Tab) {
  tab.value = next
  if (next === 'sources') {
    void loadSources()
    void loadTranslation()
    return
  }
  reload()
}

// ---- selection ----
function isSelected(id: number) {
  return selected.value.includes(id)
}
function toggleSelected(id: number) {
  selected.value = isSelected(id) ? selected.value.filter((x) => x !== id) : [...selected.value, id]
}
const allOnPageSelected = computed(() => items.value.length > 0 && items.value.every((item) => isSelected(item.id)))
function toggleSelectPage() {
  const ids = items.value.map((item) => item.id)
  selected.value = allOnPageSelected.value ? selected.value.filter((id) => !ids.includes(id)) : Array.from(new Set([...selected.value, ...ids]))
}

async function runBatch(op: Omit<PromptBatchInput, 'ids'>, ids = selected.value) {
  if (!ids.length) return
  try {
    const res = await adminAPI.promptLibrary.batch({ ...op, ids })
    appStore.showSuccess(t('admin.promptLibrary.batch.done', { n: res.updated }))
    if (ids === selected.value) selected.value = []
    await Promise.all([loadItems(), loadStats(), op.action.endsWith('tags') ? loadTags() : Promise.resolve()])
  } catch (err) {
    appStore.showError(errorMessage(err, t('common.error')))
  }
}

function quickStatus(item: PromptItem, status: PromptStatus) {
  void runBatch({ action: 'set_status', status, note: '' }, [item.id]).then(() => {
    if (status === 'active' && item.status === 'pending') appStore.showSuccess(t('admin.promptLibrary.review.approved'))
  })
}

function quickFeature(item: PromptItem) {
  void runBatch({ action: 'set_featured', featured: !item.featured }, [item.id])
}

const batchScenesOpen = ref(false)
const batchSceneMode = ref<'add_scenes' | 'remove_scenes' | 'set_scenes'>('add_scenes')
const batchScenes = ref<string[]>([])
const sceneModes = computed(() => [
  { value: 'add_scenes' as const, label: t('admin.promptLibrary.batch.addScenes') },
  { value: 'remove_scenes' as const, label: t('admin.promptLibrary.batch.removeScenes') },
  { value: 'set_scenes' as const, label: t('admin.promptLibrary.batch.setScenes') }
])
function openBatchScenes() {
  batchScenes.value = []
  batchScenesOpen.value = true
}
async function applyBatchScenes() {
  batchScenesOpen.value = false
  await runBatch({ action: batchSceneMode.value, scenes: batchScenes.value })
}

const batchTagsOpen = ref(false)
const batchTagMode = ref<'add_tags' | 'remove_tags'>('add_tags')
const batchTagsText = ref('')
const tagModes = computed(() => [
  { value: 'add_tags' as const, label: t('admin.promptLibrary.batch.addTags') },
  { value: 'remove_tags' as const, label: t('admin.promptLibrary.batch.removeTags') }
])
function openBatchTags() {
  batchTagsText.value = ''
  batchTagsOpen.value = true
}
async function applyBatchTags() {
  batchTagsOpen.value = false
  await runBatch({ action: batchTagMode.value, tags: parseTags(batchTagsText.value) })
}

// ---- review ----
const rejecting = ref<PromptItem | null>(null)
const rejectNote = ref('')
function openReject(item: PromptItem) {
  rejecting.value = item
  rejectNote.value = ''
}
async function confirmReject() {
  const item = rejecting.value
  rejecting.value = null
  if (!item) return
  await runBatch({ action: 'set_status', status: 'rejected', note: rejectNote.value }, [item.id])
}

// ---- editor ----
const editorOpen = ref(false)
const saving = ref(false)
const editingId = ref(0)
const editingItem = ref<PromptItem | null>(null)
const tagsText = ref('')
const blankForm = (): PromptAdminInput => ({
  title: '', prompt: '', description: '', cover_url: '', kind: 'image', scenes: [], tags: [], model: 'gpt-image-2',
  needs_reference: false, status: 'active', visibility: 'public', featured: true, review_note: ''
})
const form = reactive<PromptAdminInput>(blankForm())

const suggestedTags = computed(() => {
  const current = parseTags(tagsText.value)
  return knownTags.value.map((tag) => tag.tag).filter((tag) => !current.includes(tag)).slice(0, 16)
})

function openEditor(item: PromptItem | null) {
  editingId.value = item?.id || 0
  editingItem.value = item
  Object.assign(form, item ? toAdminInput(item) : blankForm())
  tagsText.value = form.tags.join(' ')
  editorOpen.value = true
}

function toggleScene(scene: string) {
  if (form.scenes.includes(scene)) form.scenes = form.scenes.filter((s) => s !== scene)
  else if (form.scenes.length < 4) form.scenes = [...form.scenes, scene]
}
function makePrimary(scene: string) {
  form.scenes = [scene, ...form.scenes.filter((s) => s !== scene)]
}
function addTag(tag: string) {
  tagsText.value = [...parseTags(tagsText.value), tag].join(' ')
}

async function save() {
  const payload: PromptAdminInput = { ...form, scenes: [...form.scenes], tags: parseTags(tagsText.value) }
  saving.value = true
  try {
    if (editingId.value) await adminAPI.promptLibrary.update(editingId.value, payload)
    else await adminAPI.promptLibrary.create(payload)
    appStore.showSuccess(t('admin.promptLibrary.editor.saved'))
    editorOpen.value = false
    await Promise.all([loadItems(), loadStats(), loadTags()])
  } catch (err) {
    appStore.showError(errorMessage(err, t('common.error')))
  } finally {
    saving.value = false
  }
}

const confirmDelete = ref<PromptItem | null>(null)
async function runDelete() {
  const item = confirmDelete.value
  confirmDelete.value = null
  if (!item) return
  try {
    await adminAPI.promptLibrary.remove(item.id)
    editorOpen.value = false
    await Promise.all([loadItems(), loadStats()])
  } catch (err) {
    appStore.showError(errorMessage(err, t('common.error')))
  }
}

// ---- covers & translation ----
function coverSrc(url: string) {
  return canvasProxiedImageUrl(url)
}

const translation = ref<PromptTranslationStatus | null>(null)
const translationForm = reactive({ base_url: '', model: '', api_key: '' })
const translationSaving = ref(false)
let translationPoll: ReturnType<typeof setTimeout> | undefined

async function loadTranslation() {
  try {
    translation.value = await adminAPI.promptLibrary.translationStatus()
    if (!translationForm.base_url) translationForm.base_url = translation.value.base_url
    if (!translationForm.model) translationForm.model = translation.value.model
  } catch {
    translation.value = null
  }
  clearTimeout(translationPoll)
  if (tab.value === 'sources' && translation.value?.running) translationPoll = setTimeout(() => void loadTranslation(), 4000)
}

async function saveTranslation() {
  translationSaving.value = true
  try {
    translation.value = await adminAPI.promptLibrary.saveTranslation({ ...translationForm })
    translationForm.api_key = ''
    appStore.showSuccess(t('admin.promptLibrary.editor.saved'))
  } catch (err) {
    appStore.showError(errorMessage(err, t('common.error')))
  } finally {
    translationSaving.value = false
  }
}

async function runTranslation() {
  try {
    await adminAPI.promptLibrary.runTranslation()
    appStore.showSuccess(t('admin.promptLibrary.translation.started'))
    setTimeout(() => void loadTranslation(), 1500)
  } catch (err) {
    appStore.showError(errorMessage(err, t('common.error')))
  }
}

// ---- sources ----
let sourcePoll: ReturnType<typeof setTimeout> | undefined
function scheduleSourcePoll() {
  clearTimeout(sourcePoll)
  if (tab.value === 'sources' && sources.value.some((s) => s.syncing)) sourcePoll = setTimeout(() => void loadSources(), 3000)
}

async function setSourceEnabled(src: PromptSource, enabled: boolean) {
  try {
    await adminAPI.promptLibrary.setSourceEnabled(src.id, enabled)
    await Promise.all([loadSources(), loadStats()])
  } catch (err) {
    appStore.showError(errorMessage(err, t('common.error')))
  }
}

async function syncSource(src: PromptSource) {
  try {
    await adminAPI.promptLibrary.syncSource(src.id)
    appStore.showSuccess(t('admin.promptLibrary.sources.syncStarted'))
    src.syncing = true
    setTimeout(() => void loadSources(), 1500)
  } catch (err) {
    appStore.showError(errorMessage(err, t('common.error')))
  }
}

onMounted(() => {
  void loadItems()
  void loadStats()
  void loadSources()
  void loadTags()
})
onBeforeUnmount(() => {
  clearTimeout(sourcePoll)
  clearTimeout(translationPoll)
})
</script>
