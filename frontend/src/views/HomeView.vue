<template>
  <!-- Custom Home Content: Full Page Mode -->
  <div v-if="hasHomeContent" class="min-h-screen">
    <!-- iframe mode -->
    <iframe
      v-if="isHomeContentUrl"
      :src="homeContent.trim()"
      class="h-screen w-full border-0"
      allowfullscreen
    ></iframe>
    <!-- HTML mode - SECURITY: homeContent is admin-only setting, XSS risk is acceptable -->
    <div v-else v-html="homeContent"></div>
  </div>

  <!-- Compact Home Page -->
  <div
    v-else-if="compactHomeEnabled"
    data-testid="compact-home"
    class="flex min-h-screen flex-col bg-gray-50 text-gray-900 dark:bg-dark-950 dark:text-white"
  >
    <header class="border-b border-gray-200 px-4 py-4 sm:px-6 dark:border-dark-800">
      <nav class="mx-auto flex max-w-5xl flex-wrap items-center justify-between gap-3 sm:gap-4">
        <div class="flex min-w-0 flex-1 items-center gap-3">
          <img
            :src="siteLogo || '/logo.svg'"
            alt="Logo"
            class="h-9 w-9 shrink-0 rounded-lg object-contain"
          />
          <span class="min-w-0 truncate text-base font-semibold">{{ siteName }}</span>
        </div>
        <div class="flex max-w-full shrink-0 flex-wrap items-center justify-end gap-2">
          <LocaleSwitcher />
          <a
            v-if="docUrl"
            :href="docUrl"
            target="_blank"
            rel="noopener noreferrer"
            class="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg text-gray-500 hover:bg-gray-100 dark:text-dark-400 dark:hover:bg-dark-800"
            :title="t('home.viewDocs')"
          >
            <Icon name="book" size="md" />
          </a>
          <router-link
            v-if="showModelPlazaEntry"
            to="/model-plaza"
            class="flex h-10 shrink-0 items-center gap-1.5 rounded-lg px-2.5 text-sm font-medium text-gray-500 hover:bg-gray-100 hover:text-gray-700 dark:text-dark-400 dark:hover:bg-dark-800 dark:hover:text-white"
            :title="t('nav.modelPlaza')"
          >
            <Icon name="grid" size="md" />
            <span class="hidden sm:inline">{{ t('nav.modelPlaza') }}</span>
          </router-link>
          <button
            class="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg text-gray-500 hover:bg-gray-100 dark:text-dark-400 dark:hover:bg-dark-800"
            :title="isDark ? t('home.switchToLight') : t('home.switchToDark')"
            @click="toggleTheme"
          >
            <Icon v-if="isDark" name="sun" size="md" />
            <Icon v-else name="moon" size="md" />
          </button>
          <router-link
            :to="isAuthenticated ? dashboardPath : '/login'"
            class="inline-flex min-h-10 shrink-0 items-center justify-center rounded-lg bg-gray-900 px-4 py-2 text-sm font-medium text-white hover:bg-gray-800 dark:bg-white dark:text-gray-900 dark:hover:bg-gray-200"
          >
            {{ isAuthenticated ? t('home.dashboard') : t('home.login') }}
          </router-link>
        </div>
      </nav>
    </header>

    <main class="flex min-w-0 flex-1 items-center justify-center px-4 py-16 sm:px-6">
      <div class="min-w-0 max-w-2xl text-center">
        <img
          :src="siteLogo || '/logo.svg'"
          alt="Logo"
          class="mx-auto mb-6 h-20 w-20 rounded-2xl object-contain"
        />
        <h1 class="[overflow-wrap:anywhere] text-3xl font-bold md:text-4xl">{{ siteName }}</h1>
        <p class="mt-4 whitespace-pre-wrap [overflow-wrap:anywhere] text-base text-gray-600 dark:text-dark-300">{{ siteSubtitle }}</p>
        <router-link
          :to="isAuthenticated ? dashboardPath : '/login'"
          class="mt-8 inline-flex min-h-10 items-center justify-center rounded-lg bg-primary-600 px-5 py-2.5 text-sm font-medium text-white hover:bg-primary-700"
        >
          {{ isAuthenticated ? t('home.goToDashboard') : t('home.login') }}
        </router-link>
      </div>
    </main>

    <footer class="min-w-0 border-t border-gray-200 px-4 py-5 text-center text-sm text-gray-500 [overflow-wrap:anywhere] sm:px-6 dark:border-dark-800 dark:text-dark-400">
      &copy; {{ currentYear }} {{ siteName }}
    </footer>
  </div>

  <!-- Default Home Page -->
  <div
    v-else
    class="relative flex min-h-screen flex-col overflow-x-clip bg-gradient-to-br from-gray-50 via-primary-50/30 to-gray-100 dark:from-dark-950 dark:via-dark-900 dark:to-dark-950"
  >
    <!-- Background Decorations -->
    <div class="pointer-events-none absolute inset-0 overflow-hidden">
      <div
        class="absolute -right-40 -top-40 h-96 w-96 rounded-full bg-primary-400/20 blur-3xl"
      ></div>
      <div
        class="absolute -bottom-40 -left-40 h-96 w-96 rounded-full bg-primary-500/15 blur-3xl"
      ></div>
      <div
        class="absolute left-1/3 top-1/4 h-72 w-72 rounded-full bg-primary-300/10 blur-3xl"
      ></div>
      <div
        class="absolute bottom-1/4 right-1/4 h-64 w-64 rounded-full bg-primary-400/10 blur-3xl"
      ></div>
      <div
        class="absolute inset-0 bg-[linear-gradient(rgba(20,184,166,0.03)_1px,transparent_1px),linear-gradient(90deg,rgba(20,184,166,0.03)_1px,transparent_1px)] bg-[size:64px_64px]"
      ></div>
    </div>

    <!-- Header: five ways in on wide screens; everything else lives in the footer -->
    <header class="sticky top-0 z-30 border-b border-gray-200/50 bg-white/70 px-4 py-3 backdrop-blur-xl sm:px-6 dark:border-dark-800/60 dark:bg-dark-950/70">
      <nav class="mx-auto flex max-w-6xl items-center justify-between gap-2">
        <!-- Logo -->
        <router-link to="/home" class="flex min-w-0 items-center gap-2.5">
          <div class="h-9 w-9 flex-shrink-0 overflow-hidden rounded-xl shadow-md">
            <img :src="siteLogo || '/logo.svg'" alt="Logo" class="h-full w-full object-contain" />
          </div>
          <span class="hidden truncate text-base font-semibold text-gray-900 sm:inline dark:text-white">{{ siteName }}</span>
        </router-link>

        <div class="hidden flex-1 items-center gap-1 pl-6 text-sm md:flex" data-testid="home-nav-links">
          <a
            v-for="link in navLinks"
            :key="link.key"
            :href="link.href"
            v-bind="link.external ? { target: '_blank', rel: 'noopener noreferrer' } : {}"
            class="rounded-lg px-3 py-2 text-gray-600 transition-colors hover:bg-gray-100 hover:text-gray-900 dark:text-dark-300 dark:hover:bg-dark-800 dark:hover:text-white"
          >
            {{ t(`home.v3.nav.${link.key}`) }}
          </a>
        </div>

        <div class="flex shrink-0 items-center gap-1 sm:gap-2">
          <LocaleSwitcher />
          <button
            @click="toggleTheme"
            class="rounded-lg p-2 text-gray-500 transition-colors hover:bg-gray-100 hover:text-gray-700 dark:text-dark-400 dark:hover:bg-dark-800 dark:hover:text-white"
            :title="isDark ? t('home.switchToLight') : t('home.switchToDark')"
          >
            <Icon v-if="isDark" name="sun" size="md" />
            <Icon v-else name="moon" size="md" />
          </button>

          <!-- Login / Dashboard Button -->
          <router-link
            v-if="isAuthenticated"
            :to="dashboardPath"
            class="inline-flex shrink-0 items-center gap-1.5 rounded-full bg-gray-900 py-1 pl-1 pr-2.5 transition-colors hover:bg-gray-800 dark:bg-gray-800 dark:hover:bg-gray-700"
          >
            <span
              class="flex h-5 w-5 items-center justify-center rounded-full bg-gradient-to-br from-primary-400 to-primary-600 text-[10px] font-semibold text-white"
            >
              {{ userInitial }}
            </span>
            <span class="whitespace-nowrap text-xs font-medium text-white">{{ t('home.dashboard') }}</span>
          </router-link>
          <template v-else>
            <router-link
              to="/login"
              class="inline-flex shrink-0 items-center whitespace-nowrap rounded-full px-2.5 py-1 text-sm text-gray-700 transition-colors hover:bg-gray-100 dark:text-dark-200 dark:hover:bg-dark-800"
              @click="trackCta('nav-login')"
            >
              {{ t('home.login') }}
            </router-link>
            <router-link
              v-if="registrationEnabled"
              to="/register"
              class="inline-flex shrink-0 items-center gap-1.5 whitespace-nowrap rounded-xl bg-primary-600 px-3 py-1.5 text-sm font-semibold text-white shadow-md shadow-primary-600/30 transition hover:bg-primary-500"
              @click="trackCta('nav-register')"
            >
              {{ t('home.register') }}
              <span v-if="signupBonus > 0" class="rounded-md bg-amber-300 px-1.5 text-[11px] font-bold text-amber-950">{{ t('home.v3.nav.signupTag', { amount: signupBonusText }) }}</span>
            </router-link>
          </template>
        </div>
      </nav>
    </header>

    <!-- Main Content -->
    <main class="relative z-10 flex-1 px-4 pb-16 pt-8 sm:px-6 lg:pt-14">
      <div class="mx-auto max-w-6xl">
        <!-- Hero: the slogan, what signing up gets you, one button; on the right a sentence turning into a piece of work -->
        <section class="home-hero relative mb-24 grid items-center gap-10 lg:grid-cols-[1.02fr_1fr] lg:gap-14" data-testid="home-hero">
          <div>
            <p class="mb-5 inline-flex items-center gap-2 rounded-full border border-gray-200 bg-white/60 py-1 pl-1 pr-3 text-xs text-gray-500 sm:text-sm dark:border-white/10 dark:bg-white/5 dark:text-dark-300">
              <span class="rounded-full bg-teal-400 px-2 py-0.5 text-[11px] font-bold text-teal-950">NEW</span>
              <span class="text-gray-900 dark:text-white">{{ t('home.v3.hero.news') }}</span>· {{ t('home.v3.hero.newsTail') }}
            </p>
            <h1 class="break-keep text-[min(46px,11.5vw)] font-extrabold leading-[1.06] tracking-tight text-gray-900 sm:text-6xl lg:text-[80px] dark:text-white">
              <span class="block">{{ t('home.v3.hero.titleLead') }}</span>
              <span class="inline-block bg-gradient-to-r from-violet-400 via-primary-600 to-teal-500 bg-clip-text pb-1 text-transparent">{{ t('home.v3.hero.titleHighlight') }}</span>
            </h1>
            <p class="mt-5 max-w-xl text-base leading-relaxed text-gray-600 sm:text-lg dark:text-dark-300">
              {{ t('home.v3.hero.subtitleLead') }}<span class="text-gray-900 dark:text-white">{{ t('home.v3.hero.subtitleEm') }}</span>{{ t('home.v3.hero.subtitleTail') }}
            </p>

            <!-- What signing up gets you, in pictures and chats rather than dollars -->
            <div
              v-if="showOffer"
              class="mt-7 flex max-w-lg items-center gap-4 rounded-2xl border border-amber-300/60 bg-gradient-to-r from-amber-100/80 to-amber-50/30 p-3 pr-5 dark:border-amber-300/30 dark:from-amber-300/15 dark:to-amber-300/[0.03]"
              data-testid="home-offer"
            >
              <span class="grid h-14 w-14 shrink-0 place-items-center rounded-xl bg-amber-300 text-2xl font-extrabold tracking-tight text-amber-950">${{ signupBonusText }}</span>
              <span>
                <span class="block font-semibold text-gray-900 dark:text-white">{{ t('home.v3.hero.offerTitle', { amount: signupBonusText }) }}</span>
                <span class="mt-0.5 block text-sm text-amber-900/80 dark:text-amber-100/70">{{ offerLine }}</span>
              </span>
            </div>

            <div class="mt-7 flex flex-wrap gap-3">
              <div ref="heroCta" class="w-full sm:w-auto">
                <router-link
                  v-if="isAuthenticated"
                  :to="dashboardPath"
                  class="btn btn-primary w-full px-7 py-3.5 text-base shadow-lg shadow-primary-600/30 sm:w-auto sm:text-lg"
                >
                  {{ t('home.v3.hero.ctaDashboard') }} →
                </router-link>
                <router-link
                  v-else-if="registrationEnabled"
                  to="/register"
                  class="btn btn-primary w-full px-7 py-3.5 text-base shadow-lg shadow-primary-600/30 sm:w-auto sm:text-lg"
                  data-testid="home-hero-signup"
                  @click="trackCta('hero-signup')"
                >
                  {{ signupBonus > 0 ? t('home.v3.hero.cta', { amount: signupBonusText }) : t('home.v3.hero.ctaNoBonus') }} →
                </router-link>
              </div>
              <a href="#jobs" class="btn btn-secondary hidden px-6 py-3.5 text-base sm:inline-flex">{{ t('home.v3.hero.ctaSecondary') }}</a>
            </div>
            <div class="mt-6 hidden flex-wrap gap-x-6 gap-y-2 text-sm text-gray-500 sm:flex dark:text-dark-400">
              <span v-for="item in trustItems" :key="item" class="inline-flex items-center gap-2">
                <span class="h-1.5 w-1.5 rotate-45 rounded-[1px] bg-teal-500" aria-hidden="true"></span>{{ item }}
              </span>
            </div>
          </div>

          <HomeHeroDemo @generate="onGenerate" />
        </section>

        <HomeJobs />

        <HomePricing :is-authenticated="isAuthenticated" :signup-bonus="signupBonus" :contact-info="contactInfo" />

        <HomeCommunityWall />

        <HomeFaq :site-name="siteName" />

        <!-- Final CTA: the slogan once more -->
        <section class="relative overflow-hidden rounded-3xl bg-gradient-to-br from-primary-600 via-violet-600 to-teal-600 px-6 py-14 text-center text-white shadow-xl sm:px-12">
          <div class="pointer-events-none absolute -right-16 -top-16 h-56 w-56 rounded-full bg-white/10 blur-2xl"></div>
          <h2 class="break-keep text-3xl font-extrabold tracking-tight sm:text-5xl">{{ t('home.v3.hero.titleLead') }}{{ t('home.v3.hero.titleHighlight') }}</h2>
          <p class="mt-4 text-white/80">{{ signupBonus > 0 ? t('home.v3.finalCta.bonus', { amount: signupBonusText }) : t('home.v3.finalCta.noBonus') }}</p>
          <router-link
            v-if="isAuthenticated || registrationEnabled"
            :to="isAuthenticated ? dashboardPath : '/register'"
            class="mt-8 inline-flex items-center rounded-xl bg-white px-7 py-3 text-base font-semibold text-primary-700 shadow-lg transition hover:bg-gray-100"
            @click="trackCta('final-cta')"
          >
            {{ isAuthenticated ? t('home.v3.hero.ctaDashboard') : signupBonus > 0 ? t('home.v3.hero.cta', { amount: signupBonusText }) : t('home.v3.hero.ctaNoBonus') }} →
          </router-link>
        </section>
      </div>
    </main>

    <!-- Footer -->
    <footer class="relative z-10 mt-8 border-t border-gray-200/50 px-6 pb-24 pt-12 sm:pb-8 dark:border-dark-800/50">
      <div class="mx-auto grid max-w-6xl gap-10 sm:grid-cols-2 lg:grid-cols-4">
        <div class="lg:col-span-2">
          <div class="flex items-center gap-2.5">
            <img :src="siteLogo || '/logo.svg'" alt="Logo" class="h-8 w-8 rounded-lg object-contain" />
            <span class="text-lg font-semibold text-gray-900 dark:text-white">{{ siteName }}</span>
          </div>
          <p class="mt-3 max-w-sm text-sm text-gray-500 dark:text-dark-400">{{ t('home.v2.footer.desc') }}</p>
          <p v-if="contactInfo" class="mt-3 text-sm text-gray-500 dark:text-dark-400">{{ t('home.v2.support.contact.label', { info: contactInfo }) }}</p>
        </div>
        <div>
          <h3 class="text-sm font-semibold text-gray-900 dark:text-white">{{ t('home.v2.footer.product') }}</h3>
          <ul class="mt-3 space-y-2 text-sm text-gray-500 dark:text-dark-400">
            <li><a :href="canvasLink('footer')" target="_blank" rel="noopener noreferrer" class="hover:text-gray-900 dark:hover:text-white">{{ t('home.v2.footer.canvas') }}</a></li>
            <li><a href="/editor/?utm_source=home&utm_medium=footer" class="hover:text-gray-900 dark:hover:text-white">{{ t('home.v3.footer.editor') }}</a></li>
            <li><router-link to="/style" class="hover:text-gray-900 dark:hover:text-white">{{ t('home.v3.footer.style') }}</router-link></li>
            <li><router-link to="/pricing" class="hover:text-gray-900 dark:hover:text-white">{{ t('pricing.navLabel') }}</router-link></li>
            <li><a href="/learn/connect/" class="hover:text-gray-900 dark:hover:text-white">{{ t('pricing.guideNav') }}</a></li>
            <li><router-link to="/contests" class="hover:text-gray-900 dark:hover:text-white">{{ t('home.v2.footer.contests') }}</router-link></li>
            <li v-if="hasCourses"><router-link to="/courses" class="hover:text-gray-900 dark:hover:text-white">{{ t('courses.navLabel') }}</router-link></li>
            <li v-if="showModelPlazaEntry"><router-link to="/model-plaza" class="hover:text-gray-900 dark:hover:text-white">{{ t('home.v2.footer.plaza') }}</router-link></li>
            <li><router-link to="/batch-image" class="hover:text-gray-900 dark:hover:text-white">{{ t('home.v2.footer.batch') }}</router-link></li>
          </ul>
        </div>
        <div>
          <h3 class="text-sm font-semibold text-gray-900 dark:text-white">{{ t('home.v2.footer.support') }}</h3>
          <ul class="mt-3 space-y-2 text-sm text-gray-500 dark:text-dark-400">
            <li v-if="docUrl"><a :href="docUrl" target="_blank" rel="noopener noreferrer" class="hover:text-gray-900 dark:hover:text-white">{{ t('home.v2.footer.docs') }}</a></li>
            <li><a :href="learnUrl('footer')" class="hover:text-gray-900 dark:hover:text-white">{{ t('home.v2.footer.learn') }}</a></li>
            <li><a href="/learn/scenes/?utm_source=home&utm_medium=footer" class="hover:text-gray-900 dark:hover:text-white">{{ t('home.v2.footer.scenes') }}</a></li>
            <li><router-link to="/keys" class="hover:text-gray-900 dark:hover:text-white">{{ t('home.v2.footer.keys') }}</router-link></li>
            <li v-for="site in partnerSites" :key="site.key">
              <a :href="partnerSiteUrl(site, '/', 'footer')" target="_blank" rel="noopener noreferrer" class="hover:text-gray-900 dark:hover:text-white">
                {{ t('home.v2.footer.partner', { site: site.name }) }}
              </a>
            </li>
          </ul>
        </div>
      </div>
      <div class="mx-auto mt-10 flex max-w-6xl flex-col items-center justify-between gap-2 border-t border-gray-200/50 pt-6 text-xs text-gray-400 sm:flex-row dark:border-dark-800/50">
        <p>&copy; {{ currentYear }} {{ siteName }}. {{ t('home.footer.allRightsReserved') }}</p>
        <a :href="githubUrl" target="_blank" rel="noopener noreferrer" class="hover:text-gray-600 dark:hover:text-dark-200">Powered by Sub2API</a>
      </div>
    </footer>

    <!-- Phones: once the hero button scrolls away, sign-up stays one tap away -->
    <div
      v-if="showMobileBar"
      :class="['fixed inset-x-0 bottom-0 z-40 flex items-center gap-3 border-t border-gray-200 bg-white/95 px-4 pb-[calc(0.625rem+env(safe-area-inset-bottom))] pt-2.5 backdrop-blur transition-transform duration-300 sm:hidden dark:border-white/10 dark:bg-dark-950/95', mobileBarVisible ? 'translate-y-0' : 'translate-y-full']"
      data-testid="home-mobile-bar"
    >
      <div class="min-w-0 flex-1 leading-tight">
        <b class="block text-[15px] text-gray-900 dark:text-white">{{ signupBonus > 0 ? t('home.v3.mobileBar.title', { amount: signupBonusText }) : t('home.v3.hero.ctaNoBonus') }}</b>
        <span class="text-xs text-gray-500 dark:text-dark-400">{{ estimate.images > 0 ? t('home.v3.mobileBar.images', { images: estimate.images }) : t('home.v3.mobileBar.generic') }}</span>
      </div>
      <router-link to="/register" class="btn btn-primary shrink-0 px-5 py-2.5" @click="trackCta('mobile-bar')">{{ t('home.v3.mobileBar.cta') }}</router-link>
    </div>

    <HomeSignupSheet
      :open="sheetOpen"
      :prompt="sheetPrompt"
      :bonus="signupBonus"
      :images="estimate.images"
      :registration-enabled="registrationEnabled"
      @close="sheetOpen = false"
    />

    <!-- 智能客服（后台「智能客服」开启后显示） -->
    <HomeAssistant :site-name="siteName" :lifted="showMobileBar && mobileBarVisible" />
    <!-- 右侧福利小蜂：悬停展开本站优惠（来自增长配置，关掉的优惠不显示） -->
    <HomePromoMascot />
  </div>
</template>

<script setup lang="ts">
import { listCourses } from '@/api/courses'
import { ref, computed, onBeforeUnmount, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAuthStore, useAppStore } from '@/stores'
import LocaleSwitcher from '@/components/common/LocaleSwitcher.vue'
import Icon from '@/components/icons/Icon.vue'
import { sanitizeUrl } from '@/utils/url'
import { track } from '@/utils/analytics'
import { FeatureFlags, isFeatureFlagEnabled } from '@/utils/featureFlags'
import { canvasUrl, otherPartnerSites, partnerSiteUrl } from '@/constants/crossSites'
import { formatTotalRequests, getPublicPricing, signupCreditEstimate, type PublicPricing } from '@/api/pricing'
import { savePendingPrompt } from '@/utils/pendingPrompt'
import HomeHeroDemo from '@/components/home/HomeHeroDemo.vue'
import HomeJobs from '@/components/home/HomeJobs.vue'
import HomeSignupSheet from '@/components/home/HomeSignupSheet.vue'
import HomePricing from '@/components/home/HomePricing.vue'
import HomeCommunityWall from '@/components/home/HomeCommunityWall.vue'
import HomeAssistant from '@/components/home/HomeAssistant.vue'
import HomePromoMascot from '@/components/home/HomePromoMascot.vue'
import HomeFaq from '@/components/home/HomeFaq.vue'

const { t, locale } = useI18n()

const authStore = useAuthStore()
const appStore = useAppStore()

// Site settings - directly from appStore (already initialized from injected config)
const siteName = computed(() => appStore.cachedPublicSettings?.site_name || appStore.siteName || 'Sub2API')
const siteLogo = computed(() => sanitizeUrl(appStore.cachedPublicSettings?.site_logo || appStore.siteLogo || '', { allowRelative: true, allowDataUrl: true }))
const siteSubtitle = computed(() => appStore.cachedPublicSettings?.site_subtitle || 'AI API Gateway Platform')
const docUrl = computed(() => sanitizeUrl(appStore.cachedPublicSettings?.doc_url || appStore.docUrl || ''))
const homeContent = computed(() => appStore.cachedPublicSettings?.home_content || '')
const hasHomeContent = computed(() => homeContent.value.trim().length > 0)
const compactHomeEnabled = computed(() => appStore.cachedPublicSettings?.compact_home_enabled === true)
const modelPlazaEnabled = computed(() => isFeatureFlagEnabled(FeatureFlags.modelPlaza))

// Check if homeContent is a URL (for iframe display)
const isHomeContentUrl = computed(() => {
  const content = homeContent.value.trim()
  return content.startsWith('http://') || content.startsWith('https://')
})

// Theme
const isDark = ref(document.documentElement.classList.contains('dark'))

// GitHub URL
const githubUrl = 'https://github.com/Wei-Shaw/sub2api'
// 互推站点：AI 应用开发学习站（带 UTM 便于统计引流）
// AI 学习 lives at /learn/ (VitePress pages served by the backend, not SPA routes).
const learnUrl = (medium: string) => `/learn/?utm_source=home&utm_medium=${medium}`
// 互推站点：无限画布（子域名部署的生图工作台），预填本站网关地址
const canvasLink = (medium: string, path: '/' | `/${string}` = '/') =>
  canvasUrl({ medium, baseUrl: window.location.origin, path })
const partnerSites = otherPartnerSites()
const contactInfo = computed(() => String(appStore.cachedPublicSettings?.contact_info || '').trim())
const registrationEnabled = computed(() => appStore.cachedPublicSettings?.registration_enabled !== false)
const signupBonus = computed(() => Number(appStore.cachedPublicSettings?.signup_bonus || 0))
const signupBonusText = computed(() => String(Math.round(signupBonus.value * 100) / 100))
const trackCta = (where: string) => track('cta_click', { where })

const navLinks = [
  { key: 'draw', href: canvasLink('nav-draw', '/image'), external: true },
  { key: 'write', href: '/editor/?utm_source=home&utm_medium=nav', external: false },
  { key: 'code', href: '/learn/codex/hivegpt?utm_source=home&utm_medium=nav', external: false },
  { key: 'works', href: canvasLink('nav-explore', '/explore'), external: true },
  { key: 'pricing', href: '/pricing', external: false }
]

// Live numbers: what the sign-up credit buys and how many requests the site has served.
const pricing = ref<PublicPricing | null>(null)
const estimate = computed(() => signupCreditEstimate(signupBonus.value, pricing.value))
const offerLine = computed(() => {
  const { images, chats } = estimate.value
  if (images > 0 && chats > 0) return t('home.v3.hero.offerBoth', { images, chats })
  if (images > 0) return t('home.v3.hero.offerImages', { images })
  if (chats > 0) return t('home.v3.hero.offerChats', { chats })
  return t('home.v3.hero.offerGeneric')
})
const trustItems = computed(() => {
  const total = formatTotalRequests(pricing.value?.total_requests, String(locale.value))
  return [t('home.v3.hero.trustPay'), t('home.v3.hero.trustMetered'), ...(total ? [t('home.v3.hero.trustTotal', { n: total })] : [])]
})

// 生成 in the hero: members go straight to the canvas; visitors keep their sentence and sign up first.
const sheetOpen = ref(false)
const sheetPrompt = ref('')
function onGenerate(prompt: string) {
  trackCta('hero-generate')
  if (isAuthenticated.value) {
    window.open(canvasUrl({ medium: 'home-hero', baseUrl: window.location.origin, path: '/image', prompt }), '_blank', 'noopener,noreferrer')
    return
  }
  savePendingPrompt(prompt)
  sheetPrompt.value = prompt
  sheetOpen.value = true
}

// Auth state
const isAuthenticated = computed(() => authStore.isAuthenticated)
const modelPlazaRequiresAuth = computed(
  () => appStore.cachedPublicSettings?.model_plaza_require_auth === true,
)
const showModelPlazaEntry = computed(
  () => modelPlazaEnabled.value && (isAuthenticated.value || !modelPlazaRequiresAuth.value),
)
const isAdmin = computed(() => authStore.isAdmin)
const dashboardPath = computed(() => isAdmin.value ? '/admin/dashboard' : '/dashboard')
const userInitial = computed(() => {
  const user = authStore.user
  if (!user || !user.email) return ''
  return user.email.charAt(0).toUpperCase()
})

const showOffer = computed(() => !isAuthenticated.value && registrationEnabled.value && signupBonus.value > 0)
const showMobileBar = computed(() => !isAuthenticated.value && registrationEnabled.value)

// Phones get a bottom sign-up bar once the hero button has scrolled up out of view.
const heroCta = ref<HTMLElement | null>(null)
const mobileBarVisible = ref(false)
let heroObserver: IntersectionObserver | null = null

// Current year for footer
const currentYear = computed(() => new Date().getFullYear())

// Toggle theme
function toggleTheme() {
  isDark.value = !isDark.value
  document.documentElement.classList.toggle('dark', isDark.value)
  localStorage.setItem('theme', isDark.value ? 'dark' : 'light')
}

// Initialize theme
function initTheme() {
  const savedTheme = localStorage.getItem('theme')
  if (
    savedTheme === 'dark' ||
    (!savedTheme && window.matchMedia('(prefers-color-scheme: dark)').matches)
  ) {
    isDark.value = true
    document.documentElement.classList.add('dark')
  }
}

// The 课程 link shows once a course is on sale.
const hasCourses = ref(false)

onMounted(() => {
  initTheme()
  listCourses()
    .then((list) => (hasCourses.value = list.length > 0))
    .catch(() => {})

  // Check auth state
  authStore.checkAuth()

  // Ensure public settings are loaded (will use cache if already loaded from injected config)
  if (!appStore.publicSettingsLoaded) {
    appStore.fetchPublicSettings()
  }

  getPublicPricing()
    .then((p) => (pricing.value = p))
    .catch(() => {})

  if (heroCta.value && typeof IntersectionObserver !== 'undefined') {
    heroObserver = new IntersectionObserver(([entry]) => {
      mobileBarVisible.value = !entry.isIntersecting && entry.boundingClientRect.top < 0
    })
    heroObserver.observe(heroCta.value)
  }
})

onBeforeUnmount(() => heroObserver?.disconnect())
</script>
