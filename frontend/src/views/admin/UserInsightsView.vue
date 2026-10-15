<template>
  <AppLayout>
    <div class="space-y-4" data-testid="user-insights-view">
      <!-- Tabs -->
      <div class="card flex flex-wrap items-center justify-between gap-3 p-4">
        <div class="flex flex-wrap items-center gap-2">
          <button
            v-for="tb in TABS"
            :key="tb"
            type="button"
            class="btn btn-sm"
            :class="tab === tb ? 'btn-primary' : 'btn-secondary'"
            :data-testid="`ui-tab-${tb}`"
            @click="selectTab(tb)"
          >
            {{ t(`admin.userInsights.tabs.${tb}`) }}
          </button>
        </div>
        <div v-if="tab === 'overview'" class="flex items-center gap-2">
          <button
            v-for="n in RANGES"
            :key="n"
            type="button"
            class="btn btn-sm"
            :class="days === n ? 'btn-primary' : 'btn-secondary'"
            :data-testid="`ui-range-${n}`"
            @click="setDays(n)"
          >
            {{ t('admin.userInsights.range', { n }) }}
          </button>
          <button type="button" class="btn btn-secondary btn-sm" :disabled="overviewLoading" @click="loadOverview">{{ t('admin.userInsights.refresh') }}</button>
        </div>
        <div v-else-if="tab === 'events'" class="flex items-center gap-2">
          <button type="button" class="btn btn-primary btn-sm" :disabled="monitorRunning" data-testid="ui-run-monitor" @click="runCheck">
            {{ monitorRunning ? t('admin.userInsights.events.running') : t('admin.userInsights.events.run') }}
          </button>
          <button type="button" class="btn btn-secondary btn-sm" :disabled="eventsLoading" @click="loadEvents">{{ t('admin.userInsights.refresh') }}</button>
        </div>
      </div>

      <!-- ===== Overview ===== -->
      <template v-if="tab === 'overview'">
        <div v-if="overviewError" class="card p-4 text-sm text-red-600">{{ t('admin.userInsights.loadFailed') }}：{{ overviewError }}</div>
        <template v-if="overview">
          <p class="text-sm text-gray-600 dark:text-dark-300">{{ t('admin.userInsights.since', { since: formatDateOnly(overview.since), days: overview.days }) }}</p>

          <div class="grid grid-cols-2 gap-3 md:grid-cols-4" data-testid="ui-segments">
            <div v-for="s in SEGMENTS" :key="s" class="card p-4">
              <div class="text-2xl font-bold" :class="SEGMENT_TEXT[s]">{{ overview.segments[s] }}</div>
              <div class="text-sm font-medium text-gray-900 dark:text-white">{{ t(`admin.userInsights.segments.${s}`) }}</div>
              <div class="text-xs text-gray-500 dark:text-dark-400">{{ t(`admin.userInsights.segments.hint.${s}`) }}</div>
            </div>
          </div>

          <div class="grid gap-4 lg:grid-cols-2">
            <!-- Sign-ups -->
            <div class="card p-4" data-testid="ui-signups">
              <h3 class="text-base font-semibold text-gray-900 dark:text-white">{{ t('admin.userInsights.signups.title') }}</h3>
              <p class="mb-2 text-xs text-gray-500 dark:text-dark-400">{{ t('admin.userInsights.signups.hint') }}</p>
              <div v-if="overview.signups?.length" class="mb-3 h-56"><Bar :data="signupChart" :options="stackedOptions" /></div>
              <div class="max-h-72 overflow-auto">
                <table class="w-full text-sm">
                  <thead>
                    <tr class="text-left text-xs text-gray-500 dark:text-dark-400">
                      <th class="py-1">{{ t('admin.userInsights.col.date') }}</th>
                      <th v-for="s in SEGMENTS" :key="s">{{ t(`admin.userInsights.segments.${s}`) }}</th>
                      <th>{{ t('admin.userInsights.total') }}</th>
                    </tr>
                  </thead>
                  <tbody>
                    <tr v-for="d in signupRows" :key="d.date" class="border-t border-gray-100 dark:border-dark-700">
                      <td class="py-1">{{ d.date }}</td>
                      <td v-for="s in SEGMENTS" :key="s">{{ d[s] }}</td>
                      <td class="font-medium">{{ d.inactive + d.trial + d.paying + d.churned }}</td>
                    </tr>
                    <tr v-if="!signupRows.length"><td colspan="6" class="py-2 text-gray-500">{{ t('admin.userInsights.empty') }}</td></tr>
                  </tbody>
                </table>
              </div>
            </div>

            <!-- Spend -->
            <div class="card p-4" data-testid="ui-spend">
              <h3 class="text-base font-semibold text-gray-900 dark:text-white">{{ t('admin.userInsights.spend.title') }}</h3>
              <p class="mb-2 text-xs text-gray-500 dark:text-dark-400">{{ t('admin.userInsights.spend.hint') }}</p>
              <div v-if="overview.spend?.length" class="mb-3 h-56"><Line :data="spendChart" :options="spendOptions" /></div>
              <div class="max-h-72 overflow-auto">
                <table class="w-full text-sm">
                  <thead>
                    <tr class="text-left text-xs text-gray-500 dark:text-dark-400">
                      <th class="py-1">{{ t('admin.userInsights.col.date') }}</th>
                      <th>{{ t('admin.userInsights.spend.free') }}</th>
                      <th>{{ t('admin.userInsights.spend.used') }}</th>
                      <th>{{ t('admin.userInsights.spend.paid') }}</th>
                    </tr>
                  </thead>
                  <tbody>
                    <tr v-if="spendRows.length" class="border-t border-gray-100 bg-gray-50 font-medium dark:border-dark-700 dark:bg-dark-800/60">
                      <td class="py-1">{{ t('admin.userInsights.total') }}</td>
                      <td>{{ usd(spendTotals.free) }}</td>
                      <td>{{ usd(spendTotals.used) }}</td>
                      <td>{{ cny(spendTotals.paid) }}</td>
                    </tr>
                    <tr v-for="d in spendRows" :key="d.date" class="border-t border-gray-100 dark:border-dark-700">
                      <td class="py-1">{{ d.date }}</td>
                      <td :class="d.free_usd > 0 ? 'text-amber-600 dark:text-amber-400' : ''">{{ usd(d.free_usd) }}</td>
                      <td>{{ usd(d.used_usd) }}</td>
                      <td>{{ cny(d.paid_cny) }}</td>
                    </tr>
                    <tr v-if="!spendRows.length"><td colspan="4" class="py-2 text-gray-500">{{ t('admin.userInsights.empty') }}</td></tr>
                  </tbody>
                </table>
              </div>
            </div>
          </div>

          <!-- Clients -->
          <div class="card p-4" data-testid="ui-clients">
            <h3 class="text-base font-semibold text-gray-900 dark:text-white">{{ t('admin.userInsights.clients.title') }}</h3>
            <p class="mb-2 text-xs text-gray-500 dark:text-dark-400">{{ t('admin.userInsights.clients.hint') }}</p>
            <div class="overflow-x-auto">
              <table class="w-full text-sm">
                <thead>
                  <tr class="text-left text-xs text-gray-500 dark:text-dark-400">
                    <th class="py-1">{{ t('admin.userInsights.col.class') }}</th>
                    <th>{{ t('admin.userInsights.col.requests') }}</th>
                    <th class="w-1/3"></th>
                    <th>{{ t('admin.userInsights.col.users') }}</th>
                    <th>{{ t('admin.userInsights.col.cost') }}</th>
                  </tr>
                </thead>
                <tbody>
                  <tr v-for="c in overview.clients || []" :key="c.class" class="border-t border-gray-100 dark:border-dark-700">
                    <td class="py-1" :class="c.class === 'relay' ? 'font-semibold text-red-600 dark:text-red-400' : ''">{{ clientLabel(c.class) }}</td>
                    <td>{{ c.requests }}</td>
                    <td>
                      <div class="h-3 rounded bg-gray-100 dark:bg-dark-700">
                        <div class="h-3 rounded" :class="c.class === 'relay' ? 'bg-red-500' : 'bg-primary-500'" :style="{ width: barWidth(c.requests, clientTop) }"></div>
                      </div>
                    </td>
                    <td>{{ c.users }}</td>
                    <td>{{ usd(c.cost_usd) }}</td>
                  </tr>
                  <tr v-if="!overview.clients?.length"><td colspan="5" class="py-2 text-gray-500">{{ t('admin.userInsights.empty') }}</td></tr>
                </tbody>
              </table>
            </div>
          </div>

          <!-- Clusters -->
          <div class="card p-4" data-testid="ui-clusters">
            <h3 class="text-base font-semibold text-gray-900 dark:text-white">{{ t('admin.userInsights.clusters.title') }}</h3>
            <p class="mb-2 text-xs text-gray-500 dark:text-dark-400">{{ t('admin.userInsights.clusters.hint') }}</p>
            <div class="overflow-x-auto">
              <table class="w-full text-sm">
                <thead>
                  <tr class="text-left text-xs text-gray-500 dark:text-dark-400">
                    <th class="py-1">{{ t('admin.userInsights.col.kind') }}</th>
                    <th>{{ t('admin.userInsights.col.key') }}</th>
                    <th>{{ t('admin.userInsights.col.members') }}</th>
                    <th>{{ t('admin.userInsights.col.freeUsd') }}</th>
                    <th>{{ t('admin.userInsights.col.paidCny') }}</th>
                  </tr>
                </thead>
                <tbody>
                  <tr v-for="cl in overview.clusters || []" :key="cl.kind + cl.key" class="border-t border-gray-100 align-top dark:border-dark-700">
                    <td class="whitespace-nowrap py-2">
                      <span class="rounded bg-gray-100 px-1.5 py-0.5 text-xs text-gray-700 dark:bg-dark-700 dark:text-dark-200">{{ t(`admin.userInsights.clusters.kind.${cl.kind}`) }}</span>
                    </td>
                    <td class="max-w-[12rem] truncate py-2 font-mono text-xs" :title="cl.key">{{ cl.key }}</td>
                    <td class="py-2">
                      <div class="mb-1 text-xs text-gray-500 dark:text-dark-400">{{ t('admin.userInsights.clusters.members', { n: cl.users.length }) }}</div>
                      <div class="flex flex-wrap gap-x-3 gap-y-1">
                        <span v-for="u in cl.users" :key="u.id" class="inline-flex items-center gap-1">
                          <router-link :to="profileLink(u.id)" class="text-primary-600 hover:underline dark:text-primary-400">{{ userName(u) }}</router-link>
                          <span v-if="u.status === 'disabled'" class="badge badge-danger">{{ t('admin.userInsights.badge.disabled') }}</span>
                          <span v-if="u.shadowed" class="badge badge-warning">{{ t('admin.userInsights.badge.shadowed') }}</span>
                          <span v-if="u.paid_cny > 0" class="badge badge-success">{{ t('admin.userInsights.badge.paid') }}</span>
                        </span>
                      </div>
                    </td>
                    <td class="py-2" :class="cl.free_usd > 0 ? 'text-amber-600 dark:text-amber-400' : ''">{{ usd(cl.free_usd) }}</td>
                    <td class="py-2">{{ cny(cl.paid_cny) }}</td>
                  </tr>
                  <tr v-if="!overview.clusters?.length"><td colspan="5" class="py-2 text-gray-500">{{ t('admin.userInsights.empty') }}</td></tr>
                </tbody>
              </table>
            </div>
          </div>

          <!-- Relay users -->
          <div class="card p-4" data-testid="ui-relay-users">
            <h3 class="text-base font-semibold text-gray-900 dark:text-white">{{ t('admin.userInsights.relayUsers.title') }}</h3>
            <p class="mb-2 text-xs text-gray-500 dark:text-dark-400">{{ t('admin.userInsights.relayUsers.hint') }}</p>
            <div class="overflow-x-auto">
              <table class="w-full text-sm">
                <thead>
                  <tr class="text-left text-xs text-gray-500 dark:text-dark-400">
                    <th class="py-1">{{ t('admin.userInsights.col.user') }}</th>
                    <th>{{ t('admin.userInsights.col.userAgent') }}</th>
                    <th>{{ t('admin.userInsights.col.requests') }}</th>
                    <th>{{ t('admin.userInsights.col.cost') }}</th>
                    <th>{{ t('admin.userInsights.col.paidCny') }}</th>
                  </tr>
                </thead>
                <tbody>
                  <tr v-for="r in overview.relay_users || []" :key="r.user.id" class="border-t border-gray-100 dark:border-dark-700">
                    <td class="py-1">
                      <router-link :to="profileLink(r.user.id)" class="text-primary-600 hover:underline dark:text-primary-400">{{ userName(r.user) }}</router-link>
                      <span v-if="r.user.status === 'disabled'" class="badge badge-danger ml-1">{{ t('admin.userInsights.badge.disabled') }}</span>
                      <span v-if="r.user.shadowed" class="badge badge-warning ml-1">{{ t('admin.userInsights.badge.shadowed') }}</span>
                    </td>
                    <td class="max-w-xs truncate font-mono text-xs" :title="r.user_agent">{{ r.user_agent }}</td>
                    <td>{{ r.requests }}</td>
                    <td>{{ usd(r.cost_usd) }}</td>
                    <td>{{ cny(r.user.paid_cny) }}</td>
                  </tr>
                  <tr v-if="!overview.relay_users?.length"><td colspan="5" class="py-2 text-gray-500">{{ t('admin.userInsights.empty') }}</td></tr>
                </tbody>
              </table>
            </div>
          </div>
        </template>
      </template>

      <!-- ===== Profile ===== -->
      <template v-else-if="tab === 'profile'">
        <form class="card flex flex-wrap items-center gap-2 p-4" data-testid="ui-profile-search" @submit.prevent="searchProfile">
          <input v-model="query" type="text" class="input max-w-sm flex-1" :placeholder="t('admin.userInsights.profile.searchPlaceholder')" data-testid="ui-profile-input" />
          <button type="submit" class="btn btn-primary btn-sm" :disabled="profileLoading || !query.trim()">{{ t('admin.userInsights.profile.search') }}</button>
          <button v-if="profile" type="button" class="btn btn-secondary btn-sm" :disabled="profileLoading" @click="reloadProfile">{{ t('admin.userInsights.refresh') }}</button>
        </form>

        <div v-if="profileError" class="card p-4 text-sm text-red-600" data-testid="ui-profile-error">{{ t('admin.userInsights.loadFailed') }}：{{ profileError }}</div>
        <p v-else-if="!profile && !profileLoading" class="card p-4 text-sm text-gray-500 dark:text-dark-400">{{ t('admin.userInsights.profile.prompt') }}</p>

        <template v-if="profile">
          <!-- Header -->
          <div class="card space-y-3 p-4" data-testid="ui-profile-header">
            <div class="flex flex-wrap items-start justify-between gap-3">
              <div class="space-y-1">
                <div class="flex flex-wrap items-center gap-2">
                  <h3 class="text-lg font-semibold text-gray-900 dark:text-white">{{ profile.user.email || profile.user.username }}</h3>
                  <span v-if="profile.user.status === 'disabled'" class="badge badge-danger">{{ t('admin.userInsights.badge.disabled') }}</span>
                  <span v-if="profile.user.shadowed" class="badge badge-warning">{{ t('admin.userInsights.badge.shadowed') }}</span>
                  <span v-if="profile.spend.has_real_paying" class="badge badge-success">{{ t('admin.userInsights.badge.paid') }}</span>
                </div>
                <div class="flex flex-wrap gap-x-4 gap-y-1 text-sm text-gray-600 dark:text-dark-300">
                  <span>{{ t('admin.userInsights.profile.id') }} {{ profile.user.id }}</span>
                  <span v-if="profile.user.username && profile.user.email">{{ profile.user.username }}</span>
                  <span>{{ t('admin.userInsights.profile.status') }}：{{ profile.user.status }}</span>
                  <span>{{ t('admin.userInsights.profile.role') }}：{{ profile.role }}</span>
                  <span>{{ t('admin.userInsights.profile.createdAt') }}：{{ formatDateTimeToMinute(profile.user.created_at) }}</span>
                  <span>{{ t('admin.userInsights.profile.signupSource') }}：{{ profile.signup_source || '–' }}</span>
                  <span>{{ t('admin.userInsights.profile.balance') }}：{{ usd(profile.balance) }}</span>
                </div>
              </div>
              <div class="flex flex-wrap gap-2" data-testid="ui-profile-actions">
                <button v-if="!profile.user.shadowed" type="button" class="btn btn-secondary btn-sm" data-testid="ui-action-shadow" @click="openAction('shadow')">{{ t('admin.userInsights.actions.shadow') }}</button>
                <button v-else type="button" class="btn btn-secondary btn-sm" data-testid="ui-action-unshadow" @click="openAction('unshadow')">{{ t('admin.userInsights.actions.unshadow') }}</button>
                <button v-if="profile.user.status !== 'disabled'" type="button" class="btn btn-danger btn-sm" data-testid="ui-action-ban" @click="openAction('ban')">{{ t('admin.userInsights.actions.ban') }}</button>
                <button v-else type="button" class="btn btn-secondary btn-sm" data-testid="ui-action-unban" @click="openAction('unban')">{{ t('admin.userInsights.actions.unban') }}</button>
              </div>
            </div>

            <div class="grid gap-4 md:grid-cols-2">
              <div class="rounded-lg border border-gray-100 p-3 dark:border-dark-700" data-testid="ui-risk">
                <div class="flex items-baseline gap-2">
                  <span class="text-3xl font-bold" :class="riskTextClass">{{ profile.risk_score }}</span>
                  <span class="text-sm text-gray-500 dark:text-dark-400">/ 100 · {{ t('admin.userInsights.profile.riskScore') }}</span>
                  <span class="rounded px-1.5 py-0.5 text-xs font-medium" :class="riskBadgeClass">{{ t(`admin.userInsights.profile.riskLevel.${riskLevel}`) }}</span>
                </div>
                <div class="mt-2 h-2 rounded bg-gray-100 dark:bg-dark-700">
                  <div class="h-2 rounded" :class="riskBarClass" :style="{ width: `${Math.min(100, Math.max(0, profile.risk_score))}%` }"></div>
                </div>
                <ul v-if="profile.risk_reasons?.length" class="mt-2 list-inside list-disc space-y-0.5 text-sm text-gray-700 dark:text-dark-200">
                  <li v-for="(r, i) in profile.risk_reasons" :key="i">{{ r }}</li>
                </ul>
                <p v-else class="mt-2 text-sm text-gray-500 dark:text-dark-400">{{ t('admin.userInsights.profile.noReasons') }}</p>
              </div>
              <div class="rounded-lg border border-gray-100 p-3 dark:border-dark-700" data-testid="ui-flags">
                <h4 class="mb-2 text-sm font-semibold text-gray-900 dark:text-white">{{ t('admin.userInsights.profile.flags') }}</h4>
                <div class="space-y-2 text-sm">
                  <div v-if="profile.user.status === 'disabled'"><span class="badge badge-danger">{{ t('admin.userInsights.profile.banned') }}</span></div>
                  <div v-if="profile.shadow" class="space-y-0.5">
                    <span class="badge badge-warning">{{ t('admin.userInsights.profile.shadowed') }}</span>
                    <p class="text-gray-700 dark:text-dark-200">{{ t('admin.userInsights.profile.shadowReason', { reason: profile.shadow.reason || '–' }) }}</p>
                    <p class="text-xs text-gray-500 dark:text-dark-400">
                      {{ t('admin.userInsights.profile.shadowMeta', { source: shadowSourceLabel(profile.shadow.source), time: formatDateTimeToMinute(profile.shadow.created_at) }) }}
                      <template v-if="profile.shadow.related_user_id">
                        ·
                        <router-link :to="profileLink(profile.shadow.related_user_id)" class="text-primary-600 hover:underline dark:text-primary-400">#{{ profile.shadow.related_user_id }}</router-link>
                      </template>
                    </p>
                  </div>
                  <p v-if="profile.user.status !== 'disabled' && !profile.shadow" class="text-gray-500 dark:text-dark-400">{{ t('admin.userInsights.profile.noFlags') }}</p>
                </div>
              </div>
            </div>
          </div>

          <div class="grid gap-4 lg:grid-cols-2">
            <!-- Sign-up -->
            <div class="card p-4" data-testid="ui-signup-info">
              <h3 class="mb-2 text-base font-semibold text-gray-900 dark:text-white">{{ t('admin.userInsights.profile.signup') }}</h3>
              <dl class="grid grid-cols-[max-content_1fr] gap-x-4 gap-y-1 text-sm">
                <template v-for="f in ATTRIBUTION_FIELDS" :key="f">
                  <dt class="text-gray-500 dark:text-dark-400">{{ t(`admin.userInsights.profile.attribution.${f}`) }}</dt>
                  <dd class="break-all text-gray-900 dark:text-white">{{ profile.attribution?.[f] || '–' }}</dd>
                </template>
                <dt class="text-gray-500 dark:text-dark-400">{{ t('admin.userInsights.profile.visitorId') }}</dt>
                <dd class="break-all font-mono text-xs text-gray-900 dark:text-white">{{ profile.visitor_id || '–' }}</dd>
                <dt class="text-gray-500 dark:text-dark-400">{{ t('admin.userInsights.profile.signupPrefix') }}</dt>
                <dd class="font-mono text-xs text-gray-900 dark:text-white">{{ profile.signup_ip_prefix || '–' }}</dd>
                <dt class="text-gray-500 dark:text-dark-400">{{ t('admin.userInsights.profile.trialGranted') }}</dt>
                <dd class="text-gray-900 dark:text-white">{{ profile.trial_granted ? t('admin.userInsights.profile.yes') : t('admin.userInsights.profile.no') }}</dd>
              </dl>
            </div>

            <!-- Spend -->
            <div class="card p-4" data-testid="ui-profile-spend">
              <h3 class="text-base font-semibold text-gray-900 dark:text-white">{{ t('admin.userInsights.profile.spend') }}</h3>
              <p class="mb-2 text-xs text-gray-500 dark:text-dark-400">{{ t('admin.userInsights.profile.spendWindow') }}</p>
              <div class="grid grid-cols-2 gap-3 sm:grid-cols-3">
                <div v-for="s in spendCells" :key="s.key">
                  <div class="text-lg font-semibold" :class="s.cls">{{ s.value }}</div>
                  <div class="text-xs text-gray-500 dark:text-dark-400">{{ t(`admin.userInsights.profile.${s.key}`) }}</div>
                </div>
              </div>
            </div>
          </div>

          <div class="grid gap-4 lg:grid-cols-2">
            <!-- Web prefixes -->
            <div class="card p-4">
              <h3 class="mb-2 text-base font-semibold text-gray-900 dark:text-white">{{ t('admin.userInsights.profile.webPrefixes') }}</h3>
              <table class="w-full text-sm">
                <thead>
                  <tr class="text-left text-xs text-gray-500 dark:text-dark-400">
                    <th class="py-1">{{ t('admin.userInsights.col.prefix') }}</th>
                    <th>{{ t('admin.userInsights.col.count') }}</th>
                    <th>{{ t('admin.userInsights.col.lastSeen') }}</th>
                  </tr>
                </thead>
                <tbody>
                  <tr v-for="r in profile.web_prefixes || []" :key="r.key" class="border-t border-gray-100 dark:border-dark-700">
                    <td class="py-1 font-mono text-xs">{{ r.key }}</td>
                    <td>{{ r.count }}</td>
                    <td class="text-xs">{{ formatDateTimeToMinute(r.last_seen) }}</td>
                  </tr>
                  <tr v-if="!profile.web_prefixes?.length"><td colspan="3" class="py-2 text-gray-500">{{ t('admin.userInsights.empty') }}</td></tr>
                </tbody>
              </table>
            </div>

            <!-- API IPs -->
            <div class="card p-4">
              <h3 class="mb-2 text-base font-semibold text-gray-900 dark:text-white">{{ t('admin.userInsights.profile.apiIps') }}</h3>
              <table class="w-full text-sm">
                <thead>
                  <tr class="text-left text-xs text-gray-500 dark:text-dark-400">
                    <th class="py-1">{{ t('admin.userInsights.col.ip') }}</th>
                    <th>{{ t('admin.userInsights.col.requests') }}</th>
                    <th>{{ t('admin.userInsights.col.cost') }}</th>
                    <th>{{ t('admin.userInsights.col.lastSeen') }}</th>
                  </tr>
                </thead>
                <tbody>
                  <tr v-for="r in profile.api_ips || []" :key="r.key" class="border-t border-gray-100 dark:border-dark-700">
                    <td class="py-1 font-mono text-xs">{{ r.key }}</td>
                    <td>{{ r.count }}</td>
                    <td>{{ usd(r.cost_usd || 0) }}</td>
                    <td class="text-xs">{{ formatDateTimeToMinute(r.last_seen) }}</td>
                  </tr>
                  <tr v-if="!profile.api_ips?.length"><td colspan="4" class="py-2 text-gray-500">{{ t('admin.userInsights.empty') }}</td></tr>
                </tbody>
              </table>
            </div>
          </div>

          <!-- API clients -->
          <div class="card p-4" data-testid="ui-profile-clients">
            <h3 class="mb-2 text-base font-semibold text-gray-900 dark:text-white">{{ t('admin.userInsights.profile.clients') }}</h3>
            <div class="overflow-x-auto">
              <table class="w-full text-sm">
                <thead>
                  <tr class="text-left text-xs text-gray-500 dark:text-dark-400">
                    <th class="py-1">{{ t('admin.userInsights.col.class') }}</th>
                    <th>{{ t('admin.userInsights.col.userAgent') }}</th>
                    <th>{{ t('admin.userInsights.col.requests') }}</th>
                    <th>{{ t('admin.userInsights.col.cost') }}</th>
                    <th>{{ t('admin.userInsights.col.lastSeen') }}</th>
                  </tr>
                </thead>
                <tbody>
                  <tr v-for="(c, i) in profile.clients || []" :key="i" class="border-t border-gray-100 dark:border-dark-700">
                    <td class="whitespace-nowrap py-1" :class="c.class === 'relay' ? 'font-semibold text-red-600 dark:text-red-400' : ''">{{ clientLabel(c.class) }}</td>
                    <td class="max-w-md truncate font-mono text-xs" :title="c.user_agent">{{ c.user_agent || '–' }}</td>
                    <td>{{ c.requests }}</td>
                    <td>{{ usd(c.cost_usd) }}</td>
                    <td class="whitespace-nowrap text-xs">{{ formatDateTimeToMinute(c.last_seen) }}</td>
                  </tr>
                  <tr v-if="!profile.clients?.length"><td colspan="5" class="py-2 text-gray-500">{{ t('admin.userInsights.empty') }}</td></tr>
                </tbody>
              </table>
            </div>
          </div>

          <div class="grid gap-4 lg:grid-cols-2">
            <div v-for="b in usageBlocks" :key="b.key" class="card p-4">
              <h3 class="mb-2 text-base font-semibold text-gray-900 dark:text-white">{{ t(`admin.userInsights.profile.${b.key}`) }}</h3>
              <table class="w-full text-sm">
                <thead>
                  <tr class="text-left text-xs text-gray-500 dark:text-dark-400">
                    <th class="py-1">{{ t(`admin.userInsights.col.${b.col}`) }}</th>
                    <th>{{ t('admin.userInsights.col.requests') }}</th>
                    <th>{{ t('admin.userInsights.col.cost') }}</th>
                    <th>{{ t('admin.userInsights.col.lastSeen') }}</th>
                  </tr>
                </thead>
                <tbody>
                  <tr v-for="r in b.rows" :key="r.key" class="border-t border-gray-100 dark:border-dark-700">
                    <td class="py-1">{{ r.key || '–' }}</td>
                    <td>{{ r.count }}</td>
                    <td>{{ usd(r.cost_usd || 0) }}</td>
                    <td class="text-xs">{{ formatDateTimeToMinute(r.last_seen) }}</td>
                  </tr>
                  <tr v-if="!b.rows.length"><td colspan="4" class="py-2 text-gray-500">{{ t('admin.userInsights.empty') }}</td></tr>
                </tbody>
              </table>
            </div>
          </div>

          <!-- Payments -->
          <div class="card p-4" data-testid="ui-payments">
            <h3 class="mb-2 text-base font-semibold text-gray-900 dark:text-white">{{ t('admin.userInsights.profile.payments') }}</h3>
            <div class="overflow-x-auto">
              <table class="w-full text-sm">
                <thead>
                  <tr class="text-left text-xs text-gray-500 dark:text-dark-400">
                    <th class="py-1">{{ t('admin.userInsights.col.createdAt') }}</th>
                    <th>{{ t('admin.userInsights.col.orderType') }}</th>
                    <th>{{ t('admin.userInsights.col.paymentType') }}</th>
                    <th>{{ t('admin.userInsights.col.amount') }}</th>
                    <th>{{ t('admin.userInsights.col.payAmount') }}</th>
                    <th>{{ t('admin.userInsights.col.status') }}</th>
                    <th>{{ t('admin.userInsights.col.paidAt') }}</th>
                  </tr>
                </thead>
                <tbody>
                  <tr v-for="p in profile.payments || []" :key="p.id" class="border-t border-gray-100 dark:border-dark-700">
                    <td class="whitespace-nowrap py-1 text-xs">{{ formatDateTimeToMinute(p.created_at) }}</td>
                    <td>{{ p.order_type }}</td>
                    <td>{{ p.payment_type }}</td>
                    <td>{{ p.amount.toFixed(2) }}</td>
                    <td>{{ cny(p.pay_amount) }}</td>
                    <td>{{ p.status }}</td>
                    <td class="whitespace-nowrap text-xs">{{ p.paid_at ? formatDateTimeToMinute(p.paid_at) : '–' }}</td>
                  </tr>
                  <tr v-if="!profile.payments?.length"><td colspan="7" class="py-2 text-gray-500">{{ t('admin.userInsights.empty') }}</td></tr>
                </tbody>
              </table>
            </div>
          </div>

          <!-- Subscriptions -->
          <div class="card p-4" data-testid="ui-subscriptions">
            <h3 class="mb-2 text-base font-semibold text-gray-900 dark:text-white">{{ t('admin.userInsights.profile.subscriptions') }}</h3>
            <div class="overflow-x-auto">
              <table class="w-full text-sm">
                <thead>
                  <tr class="text-left text-xs text-gray-500 dark:text-dark-400">
                    <th class="py-1">{{ t('admin.userInsights.col.group') }}</th>
                    <th>{{ t('admin.userInsights.col.status') }}</th>
                    <th>{{ t('admin.userInsights.col.startsAt') }}</th>
                    <th>{{ t('admin.userInsights.col.expiresAt') }}</th>
                    <th>{{ t('admin.userInsights.col.dailyUsage') }}</th>
                  </tr>
                </thead>
                <tbody>
                  <tr v-for="s in profile.subscriptions || []" :key="s.id" class="border-t border-gray-100 dark:border-dark-700">
                    <td class="py-1">
                      {{ s.group_name }}
                      <span v-if="s.free" class="badge badge-warning ml-1">{{ t('admin.userInsights.badge.free') }}</span>
                    </td>
                    <td>{{ s.status }}</td>
                    <td class="whitespace-nowrap text-xs">{{ formatDateTimeToMinute(s.starts_at) }}</td>
                    <td class="whitespace-nowrap text-xs">{{ formatDateTimeToMinute(s.expires_at) }}</td>
                    <td>{{ s.daily_usage_usd.toFixed(2) }} / {{ s.daily_limit_usd != null ? s.daily_limit_usd.toFixed(2) : '–' }}</td>
                  </tr>
                  <tr v-if="!profile.subscriptions?.length"><td colspan="5" class="py-2 text-gray-500">{{ t('admin.userInsights.empty') }}</td></tr>
                </tbody>
              </table>
            </div>
          </div>

          <div class="grid gap-4 lg:grid-cols-2">
            <!-- Invites -->
            <div class="card space-y-3 p-4" data-testid="ui-invites">
              <h3 class="text-base font-semibold text-gray-900 dark:text-white">{{ t('admin.userInsights.profile.invites') }}</h3>
              <div>
                <h4 class="mb-1 text-sm font-medium text-gray-700 dark:text-dark-200">{{ t('admin.userInsights.profile.inviteUp') }}</h4>
                <div v-if="profile.invite_up?.length" class="flex flex-wrap items-center gap-1 text-sm">
                  <span class="text-gray-900 dark:text-white">{{ userName(profile.user) }}</span>
                  <template v-for="u in profile.invite_up" :key="u.id">
                    <span class="text-gray-400">←</span>
                    <router-link :to="profileLink(u.id)" class="text-primary-600 hover:underline dark:text-primary-400">{{ userName(u) }}</router-link>
                    <span v-if="u.status === 'disabled'" class="badge badge-danger">{{ t('admin.userInsights.badge.disabled') }}</span>
                    <span v-if="u.shadowed" class="badge badge-warning">{{ t('admin.userInsights.badge.shadowed') }}</span>
                  </template>
                </div>
                <p v-else class="text-sm text-gray-500 dark:text-dark-400">{{ t('admin.userInsights.profile.noInviter') }}</p>
              </div>
              <div>
                <h4 class="mb-1 text-sm font-medium text-gray-700 dark:text-dark-200">{{ t('admin.userInsights.profile.inviteDown') }}（{{ profile.invite_down?.length || 0 }}）</h4>
                <table v-if="profile.invite_down?.length" class="w-full text-sm">
                  <thead>
                    <tr class="text-left text-xs text-gray-500 dark:text-dark-400">
                      <th class="py-1">{{ t('admin.userInsights.col.user') }}</th>
                      <th>{{ t('admin.userInsights.col.createdAt') }}</th>
                      <th>{{ t('admin.userInsights.col.spendUsd') }}</th>
                      <th>{{ t('admin.userInsights.col.paidCny') }}</th>
                    </tr>
                  </thead>
                  <tbody>
                    <tr v-for="u in profile.invite_down" :key="u.id" class="border-t border-gray-100 dark:border-dark-700">
                      <td class="py-1">
                        <router-link :to="profileLink(u.id)" class="text-primary-600 hover:underline dark:text-primary-400">{{ userName(u) }}</router-link>
                        <span v-if="u.status === 'disabled'" class="badge badge-danger ml-1">{{ t('admin.userInsights.badge.disabled') }}</span>
                        <span v-if="u.shadowed" class="badge badge-warning ml-1">{{ t('admin.userInsights.badge.shadowed') }}</span>
                      </td>
                      <td class="whitespace-nowrap text-xs">{{ formatDateTimeToMinute(u.created_at) }}</td>
                      <td>{{ usd(u.spend_usd) }}</td>
                      <td>{{ cny(u.paid_cny) }}</td>
                    </tr>
                  </tbody>
                </table>
                <p v-else class="text-sm text-gray-500 dark:text-dark-400">{{ t('admin.userInsights.profile.noInvitees') }}</p>
              </div>
            </div>

            <!-- Linked -->
            <div class="card p-4" data-testid="ui-linked">
              <h3 class="mb-2 text-base font-semibold text-gray-900 dark:text-white">{{ t('admin.userInsights.profile.linked') }}（{{ profile.linked?.length || 0 }}）</h3>
              <table class="w-full text-sm">
                <thead>
                  <tr class="text-left text-xs text-gray-500 dark:text-dark-400">
                    <th class="py-1">{{ t('admin.userInsights.col.user') }}</th>
                    <th>{{ t('admin.userInsights.col.via') }}</th>
                    <th>{{ t('admin.userInsights.col.spendUsd') }}</th>
                    <th>{{ t('admin.userInsights.col.paidCny') }}</th>
                  </tr>
                </thead>
                <tbody>
                  <tr v-for="l in profile.linked || []" :key="l.user.id" class="border-t border-gray-100 dark:border-dark-700">
                    <td class="py-1">
                      <router-link :to="profileLink(l.user.id)" class="text-primary-600 hover:underline dark:text-primary-400">{{ userName(l.user) }}</router-link>
                      <span v-if="l.user.status === 'disabled'" class="badge badge-danger ml-1">{{ t('admin.userInsights.badge.disabled') }}</span>
                      <span v-if="l.user.shadowed" class="badge badge-warning ml-1">{{ t('admin.userInsights.badge.shadowed') }}</span>
                    </td>
                    <td>
                      <div class="flex flex-wrap gap-1">
                        <span v-for="v in l.via" :key="v" class="rounded bg-gray-100 px-1.5 py-0.5 text-xs text-gray-700 dark:bg-dark-700 dark:text-dark-200">{{ viaLabel(v) }}</span>
                      </div>
                    </td>
                    <td>{{ usd(l.user.spend_usd) }}</td>
                    <td>{{ cny(l.user.paid_cny) }}</td>
                  </tr>
                  <tr v-if="!profile.linked?.length"><td colspan="4" class="py-2 text-gray-500">{{ t('admin.userInsights.empty') }}</td></tr>
                </tbody>
              </table>
            </div>
          </div>

          <!-- Events of this user -->
          <div class="card p-4" data-testid="ui-profile-events">
            <h3 class="mb-2 text-base font-semibold text-gray-900 dark:text-white">{{ t('admin.userInsights.profile.events') }}</h3>
            <table class="w-full text-sm">
              <thead>
                <tr class="text-left text-xs text-gray-500 dark:text-dark-400">
                  <th class="py-1">{{ t('admin.userInsights.col.time') }}</th>
                  <th>{{ t('admin.userInsights.col.rule') }}</th>
                  <th>{{ t('admin.userInsights.col.detail') }}</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="e in profile.events || []" :key="e.id" class="border-t border-gray-100 dark:border-dark-700">
                  <td class="whitespace-nowrap py-1 text-xs">{{ formatDateTimeToMinute(e.created_at) }}</td>
                  <td class="whitespace-nowrap">{{ ruleLabel(e.rule) }}</td>
                  <td class="text-gray-700 dark:text-dark-200">{{ e.detail }}</td>
                </tr>
                <tr v-if="!profile.events?.length"><td colspan="3" class="py-2 text-gray-500">{{ t('admin.userInsights.empty') }}</td></tr>
              </tbody>
            </table>
          </div>
        </template>
      </template>

      <!-- ===== Events ===== -->
      <template v-else-if="tab === 'events'">
        <div class="card p-4" data-testid="ui-events">
          <h3 class="text-base font-semibold text-gray-900 dark:text-white">{{ t('admin.userInsights.events.title') }}</h3>
          <p class="mb-2 text-xs text-gray-500 dark:text-dark-400">{{ t('admin.userInsights.events.hint') }}</p>
          <div v-if="eventsError" class="mb-2 text-sm text-red-600">{{ t('admin.userInsights.loadFailed') }}：{{ eventsError }}</div>
          <div class="overflow-x-auto">
            <table class="w-full text-sm">
              <thead>
                <tr class="text-left text-xs text-gray-500 dark:text-dark-400">
                  <th class="py-1">{{ t('admin.userInsights.col.time') }}</th>
                  <th>{{ t('admin.userInsights.col.user') }}</th>
                  <th>{{ t('admin.userInsights.col.rule') }}</th>
                  <th>{{ t('admin.userInsights.col.detail') }}</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="e in events" :key="e.id" class="border-t border-gray-100 dark:border-dark-700">
                  <td class="whitespace-nowrap py-1 text-xs">{{ formatDateTimeToMinute(e.created_at) }}</td>
                  <td class="whitespace-nowrap">
                    <router-link :to="profileLink(e.user_id)" class="text-primary-600 hover:underline dark:text-primary-400">{{ e.user_email || e.username || `#${e.user_id}` }}</router-link>
                  </td>
                  <td class="whitespace-nowrap">{{ ruleLabel(e.rule) }}</td>
                  <td class="text-gray-700 dark:text-dark-200">{{ e.detail }}</td>
                </tr>
                <tr v-if="!events.length && !eventsLoading"><td colspan="4" class="py-2 text-gray-500">{{ t('admin.userInsights.empty') }}</td></tr>
              </tbody>
            </table>
          </div>
          <Pagination
            v-if="eventsTotal > 0"
            class="mt-3"
            :page="eventsPage"
            :total="eventsTotal"
            :page-size="eventsPageSize"
            @update:page="onEventsPage"
            @update:page-size="onEventsPageSize"
          />
        </div>
      </template>
    </div>

    <ConfirmDialog
      :show="action !== null"
      :title="actionTitle"
      :message="actionMessage"
      :confirm-text="actionConfirmText"
      :danger="action === 'ban' || action === 'shadow'"
      @confirm="confirmAction"
      @cancel="closeAction"
    >
      <div v-if="action === 'shadow'" class="space-y-1">
        <label class="text-sm font-medium text-gray-700 dark:text-dark-200" for="ui-shadow-reason">{{ t('admin.userInsights.actions.reasonLabel') }}</label>
        <textarea
          id="ui-shadow-reason"
          v-model="shadowReason"
          rows="3"
          class="input w-full"
          :placeholder="t('admin.userInsights.actions.reasonPlaceholder')"
          data-testid="ui-shadow-reason"
        ></textarea>
      </div>
    </ConfirmDialog>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute, useRouter } from 'vue-router'
import { BarElement, CategoryScale, Chart as ChartJS, Legend, LinearScale, LineElement, PointElement, Tooltip } from 'chart.js'
import { Bar, Line } from 'vue-chartjs'
import AppLayout from '@/components/layout/AppLayout.vue'
import ConfirmDialog from '@/components/common/ConfirmDialog.vue'
import Pagination from '@/components/common/Pagination.vue'
import {
  getOverview,
  getProfile,
  listEvents,
  runMonitor,
  shadowUser,
  unshadowUser,
  type InsightsAttribution,
  type InsightsUserRef,
  type RiskEvent,
  type UserInsightsOverview,
  type UserInsightsProfile
} from '@/api/admin/userInsights'
import { adminAPI } from '@/api/admin'
import { useAppStore } from '@/stores/app'
import { extractApiErrorMessage } from '@/utils/apiError'
import { formatDateOnly, formatDateTimeToMinute } from '@/utils/format'

ChartJS.register(CategoryScale, LinearScale, BarElement, PointElement, LineElement, Tooltip, Legend)

type Tab = 'overview' | 'profile' | 'events'
type Segment = 'inactive' | 'trial' | 'paying' | 'churned'
type Action = 'shadow' | 'unshadow' | 'ban' | 'unban'

const TABS: Tab[] = ['overview', 'profile', 'events']
const RANGES = [7, 30, 90]
const SEGMENTS: Segment[] = ['inactive', 'trial', 'paying', 'churned']
const SEGMENT_TEXT: Record<Segment, string> = {
  inactive: 'text-gray-600 dark:text-dark-300',
  trial: 'text-amber-600 dark:text-amber-400',
  paying: 'text-emerald-600 dark:text-emerald-400',
  churned: 'text-red-600 dark:text-red-400'
}
const SEGMENT_COLOR: Record<Segment, string> = { inactive: '#9ca3af', trial: '#f59e0b', paying: '#10b981', churned: '#ef4444' }
const ATTRIBUTION_FIELDS: (keyof InsightsAttribution)[] = ['source', 'medium', 'campaign', 'aff_code', 'referrer_host', 'landing_path']

const { t, te } = useI18n()
const route = useRoute()
const router = useRouter()
const appStore = useAppStore()

function queryTab(): Tab {
  const q = String(route.query.tab || '')
  return (TABS as string[]).includes(q) ? (q as Tab) : 'overview'
}
function queryUser(): string {
  return String(route.query.user || '').trim()
}

const tab = ref<Tab>(queryTab())

function selectTab(tb: Tab): void {
  const query: Record<string, string> = { tab: tb }
  if (tb === 'profile' && queryUser()) query.user = queryUser()
  void router.replace({ query })
}

function profileLink(id: number): { query: Record<string, string> } {
  return { query: { tab: 'profile', user: String(id) } }
}

// ---------- formatting ----------

function usd(n: number | null | undefined): string {
  return `$${(n || 0).toFixed(2)}`
}
function cny(n: number | null | undefined): string {
  return `¥${(n || 0).toFixed(2)}`
}
function barWidth(n: number, top: number): string {
  return `${Math.max(2, (n / Math.max(1, top)) * 100)}%`
}
function userName(u: Pick<InsightsUserRef, 'id' | 'email' | 'username'>): string {
  return u.email || u.username || `#${u.id}`
}
function labelOr(key: string, fallback: string): string {
  return te(key) ? t(key) : fallback
}
function clientLabel(c: string): string {
  return labelOr(`admin.userInsights.clientClass.${c}`, c)
}
function ruleLabel(r: string): string {
  return labelOr(`admin.userInsights.rules.${r}`, r)
}
function viaLabel(v: string): string {
  return labelOr(`admin.userInsights.via.${v}`, v)
}
function shadowSourceLabel(s: string): string {
  return labelOr(`admin.userInsights.profile.shadowSource.${s}`, s)
}

// ---------- overview ----------

const days = ref(30)
const overview = ref<UserInsightsOverview | null>(null)
const overviewLoading = ref(false)
const overviewError = ref('')

async function loadOverview(): Promise<void> {
  overviewLoading.value = true
  overviewError.value = ''
  try {
    overview.value = await getOverview(days.value)
  } catch (e: unknown) {
    overviewError.value = extractApiErrorMessage(e, t('admin.userInsights.loadFailed'))
  } finally {
    overviewLoading.value = false
  }
}

function setDays(n: number): void {
  days.value = n
  void loadOverview()
}

// Tables show the newest day first; charts run left to right.
const signupRows = computed(() => [...(overview.value?.signups || [])].reverse())
const spendRows = computed(() => [...(overview.value?.spend || [])].reverse())
const spendTotals = computed(() =>
  (overview.value?.spend || []).reduce(
    (acc, d) => ({ free: acc.free + d.free_usd, used: acc.used + d.used_usd, paid: acc.paid + d.paid_cny }),
    { free: 0, used: 0, paid: 0 }
  )
)
const clientTop = computed(() => Math.max(1, ...(overview.value?.clients || []).map((c) => c.requests)))

const signupChart = computed(() => {
  const d = overview.value?.signups || []
  return {
    labels: d.map((x) => x.date.slice(5)),
    datasets: SEGMENTS.map((s) => ({
      label: t(`admin.userInsights.segments.${s}`),
      data: d.map((x) => x[s]),
      backgroundColor: SEGMENT_COLOR[s],
      stack: 'signups'
    }))
  }
})

const stackedOptions = {
  responsive: true,
  maintainAspectRatio: false,
  interaction: { intersect: false, mode: 'index' as const },
  plugins: { legend: { position: 'top' as const, labels: { usePointStyle: true, boxWidth: 8 } } },
  scales: { x: { stacked: true }, y: { stacked: true, beginAtZero: true, ticks: { precision: 0 } } }
}

const spendChart = computed(() => {
  const d = overview.value?.spend || []
  const line = (label: string, color: string, values: number[], yAxisID: string) => ({
    label,
    data: values,
    borderColor: color,
    backgroundColor: color,
    tension: 0.3,
    pointRadius: 2,
    yAxisID
  })
  return {
    labels: d.map((x) => x.date.slice(5)),
    datasets: [
      line(t('admin.userInsights.spend.free'), '#f59e0b', d.map((x) => x.free_usd), 'y'),
      line(t('admin.userInsights.spend.used'), '#6366f1', d.map((x) => x.used_usd), 'y'),
      line(t('admin.userInsights.spend.paid'), '#10b981', d.map((x) => x.paid_cny), 'y1')
    ]
  }
})

const spendOptions = {
  responsive: true,
  maintainAspectRatio: false,
  interaction: { intersect: false, mode: 'index' as const },
  plugins: { legend: { position: 'top' as const, labels: { usePointStyle: true, boxWidth: 8 } } },
  scales: {
    y: { beginAtZero: true, position: 'left' as const, title: { display: true, text: 'USD' } },
    y1: { beginAtZero: true, position: 'right' as const, grid: { drawOnChartArea: false }, title: { display: true, text: 'CNY' } }
  }
}

// ---------- profile ----------

const query = ref(queryUser())
const profile = ref<UserInsightsProfile | null>(null)
const profileLoading = ref(false)
const profileError = ref('')
let profileQuery = ''

async function loadProfile(q: string): Promise<void> {
  profileQuery = q
  profileLoading.value = true
  profileError.value = ''
  try {
    const p = await getProfile(q)
    if (profileQuery !== q) return
    profile.value = p
  } catch (e: unknown) {
    if (profileQuery !== q) return
    profile.value = null
    profileError.value = extractApiErrorMessage(e, t('admin.userInsights.loadFailed'))
  } finally {
    if (profileQuery === q) profileLoading.value = false
  }
}

function searchProfile(): void {
  const q = query.value.trim()
  if (!q) return
  if (q === queryUser() && tab.value === 'profile') {
    void loadProfile(q)
    return
  }
  void router.replace({ query: { tab: 'profile', user: q } })
}

function reloadProfile(): void {
  if (profile.value) void loadProfile(String(profile.value.user.id))
}

const riskLevel = computed<'low' | 'medium' | 'high'>(() => {
  const s = profile.value?.risk_score || 0
  if (s >= 60) return 'high'
  if (s >= 30) return 'medium'
  return 'low'
})
const riskTextClass = computed(
  () =>
    ({
      high: 'text-red-600 dark:text-red-400',
      medium: 'text-amber-600 dark:text-amber-400',
      low: 'text-emerald-600 dark:text-emerald-400'
    })[riskLevel.value]
)
const riskBarClass = computed(() => ({ high: 'bg-red-500', medium: 'bg-amber-500', low: 'bg-emerald-500' })[riskLevel.value])
const riskBadgeClass = computed(
  () =>
    ({
      high: 'bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-300',
      medium: 'bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-300',
      low: 'bg-emerald-100 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-300'
    })[riskLevel.value]
)

const spendCells = computed(() => {
  const s = profile.value?.spend
  if (!s) return []
  return [
    { key: 'spendTotal', value: usd(s.total_usd), cls: 'text-gray-900 dark:text-white' },
    { key: 'spendFreeSub', value: usd(s.free_sub_usd), cls: s.free_sub_usd > 0 ? 'text-amber-600 dark:text-amber-400' : 'text-gray-900 dark:text-white' },
    { key: 'spendPaidSub', value: usd(s.paid_sub_usd), cls: 'text-gray-900 dark:text-white' },
    { key: 'spendBalance', value: usd(s.balance_usd), cls: 'text-gray-900 dark:text-white' },
    { key: 'paidCny', value: cny(s.paid_cny), cls: 'text-emerald-600 dark:text-emerald-400' },
    {
      key: 'everPaid',
      value: s.has_real_paying ? t('admin.userInsights.profile.yes') : t('admin.userInsights.profile.no'),
      cls: s.has_real_paying ? 'text-emerald-600 dark:text-emerald-400' : 'text-gray-500 dark:text-dark-400'
    }
  ]
})

const usageBlocks = computed(() => [
  { key: 'groups', col: 'group', rows: profile.value?.groups || [] },
  { key: 'models', col: 'model', rows: profile.value?.models || [] }
])

// ---------- actions ----------

const action = ref<Action | null>(null)
const shadowReason = ref('')
const acting = ref(false)

function openAction(a: Action): void {
  shadowReason.value = ''
  action.value = a
}
function closeAction(): void {
  if (!acting.value) action.value = null
}

const actionTitle = computed(() => (action.value ? t(`admin.userInsights.actions.${action.value}Title`) : ''))
const actionMessage = computed(() =>
  action.value && profile.value ? t(`admin.userInsights.actions.${action.value}Message`, { email: userName(profile.value.user) }) : ''
)
const actionConfirmText = computed(() => (action.value ? t(`admin.userInsights.actions.${action.value}`) : ''))

async function confirmAction(): Promise<void> {
  const a = action.value
  const p = profile.value
  if (!a || !p || acting.value) return
  const id = p.user.id
  const reason = shadowReason.value.trim()
  if (a === 'shadow' && !reason) {
    appStore.showError(t('admin.userInsights.actions.reasonRequired'))
    return
  }
  acting.value = true
  try {
    if (a === 'shadow') await shadowUser(id, reason)
    else if (a === 'unshadow') await unshadowUser(id)
    else await adminAPI.users.toggleStatus(id, a === 'ban' ? 'disabled' : 'active')
    appStore.showSuccess(t('admin.userInsights.actions.done'))
    acting.value = false
    action.value = null
    await loadProfile(String(id))
  } catch (e: unknown) {
    appStore.showError(extractApiErrorMessage(e, t('admin.userInsights.actions.failed')))
  } finally {
    acting.value = false
  }
}

// ---------- events ----------

const events = ref<RiskEvent[]>([])
const eventsTotal = ref(0)
const eventsPage = ref(1)
const eventsPageSize = ref(20)
const eventsLoading = ref(false)
const eventsError = ref('')
const eventsLoaded = ref(false)
const monitorRunning = ref(false)

async function loadEvents(): Promise<void> {
  eventsLoading.value = true
  eventsError.value = ''
  try {
    const res = await listEvents({ page: eventsPage.value, page_size: eventsPageSize.value })
    events.value = res.items || []
    eventsTotal.value = res.total || 0
    eventsLoaded.value = true
  } catch (e: unknown) {
    eventsError.value = extractApiErrorMessage(e, t('admin.userInsights.loadFailed'))
  } finally {
    eventsLoading.value = false
  }
}

function onEventsPage(p: number): void {
  eventsPage.value = p
  void loadEvents()
}
function onEventsPageSize(n: number): void {
  eventsPageSize.value = n
  eventsPage.value = 1
  void loadEvents()
}

async function runCheck(): Promise<void> {
  monitorRunning.value = true
  try {
    const res = await runMonitor()
    const n = res.new_events?.length || 0
    appStore.showSuccess(n > 0 ? t('admin.userInsights.events.newEvents', { n }) : t('admin.userInsights.events.noNew'))
    eventsPage.value = 1
    await loadEvents()
  } catch (e: unknown) {
    appStore.showError(extractApiErrorMessage(e, t('admin.userInsights.loadFailed')))
  } finally {
    monitorRunning.value = false
  }
}

// ---------- URL sync ----------

function syncFromRoute(): void {
  tab.value = queryTab()
  if (tab.value === 'overview' && !overview.value && !overviewLoading.value) void loadOverview()
  if (tab.value === 'events' && !eventsLoaded.value && !eventsLoading.value) void loadEvents()
  if (tab.value === 'profile') {
    const u = queryUser()
    query.value = u
    if (u && u !== profileQuery) void loadProfile(u)
  }
}

watch(() => [route.query.tab, route.query.user], syncFromRoute)
onMounted(syncFromRoute)
</script>
