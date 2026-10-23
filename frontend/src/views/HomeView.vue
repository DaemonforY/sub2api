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
          <!-- On phones the links above are hidden; AI 对话 stays reachable here. -->
          <a
            :href="chatLink"
            class="inline-flex items-center gap-1 whitespace-nowrap rounded-full border border-primary-200 bg-primary-50 px-2.5 py-1 text-xs font-medium text-primary-700 md:hidden dark:border-primary-800 dark:bg-primary-900/30 dark:text-primary-300"
            data-testid="home-nav-chat-mobile"
            @click="trackCta('nav-chat')"
          >
            <Icon name="chat" size="xs" />{{ t('home.v3.nav.chat') }}
          </a>
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
              class="inline-flex shrink-0 items-center gap-1.5 whitespace-nowrap rounded-xl bg-gradient-to-r from-blue-600 to-sky-500 px-3 py-1.5 text-sm font-semibold text-white shadow-md shadow-blue-600/30 transition hover:brightness-110"
              @click="trackCta('nav-register')"
            >
              {{ t('home.register') }}
              <span v-if="signupBonus > 0" class="rounded-md bg-amber-300 px-1.5 text-[11px] font-bold text-amber-950">{{ t('home.v3.nav.signupTag', { amount: signupBonusText }) }}</span>
            </router-link>
          </template>
        </div>
      </nav>
    </header>

    <!-- Main Content: the motion screens are always dark; the rest follows the theme -->
    <main class="relative z-10 flex-1">
      <!-- ① The slogan, made of tokens: they fly together into the words as the page opens. Under it what
           you get, one button (sign up, or for members the top-up page) and the live token counter. -->
      <section ref="heroSection" class="home-hero relative overflow-hidden bg-[#050a14] text-white" data-testid="home-hero">
        <div class="stage-glow pointer-events-none absolute inset-0" aria-hidden="true"></div>
        <div class="hero-grid pointer-events-none absolute inset-0" aria-hidden="true"></div>
        <HomeMotion kind="assemble" eager :deadline="2500" :context="assembleContext" @started="heroTitleMode = 'motion'" @fallback="heroTitleMode = 'text'" />
        <div class="relative z-[2] flex min-h-[calc(100svh-61px)] w-full flex-col items-center justify-center px-4 pb-24 pt-10 text-center sm:px-6">
          <p class="mb-8 inline-flex items-center gap-2 rounded-full border border-white/15 bg-[#050a14]/60 py-1 pl-1 pr-3 text-xs text-slate-400 sm:mb-10 sm:text-sm">
            <span class="rounded-full bg-cyan-300 px-2 py-0.5 text-[11px] font-bold text-cyan-950">NEW</span>
            <span class="text-white">{{ t('home.v3.hero.news') }}</span>· {{ t('home.v3.hero.newsTail') }}
          </p>
          <!-- the particles draw this heading in its place; it shows itself when there is no motion -->
          <h1 :class="['hero-title', heroTitleMode]" data-testid="home-hero-title">
            <span ref="heroLine1">{{ t('home.v3.hero.titleLead') }}</span>{{ heroTitleGap }}<span ref="heroLine2" class="t2">{{ t('home.v3.hero.titleHighlight') }}</span>
          </h1>
          <p class="mt-10 max-w-xl text-[15px] text-slate-300 sm:mt-14 sm:text-base" data-testid="home-hero-line">
            <template v-for="(part, i) in ctaLine" :key="i"><span v-if="i" class="px-1.5 text-slate-500">·</span><span class="inline-block">{{ part }}</span></template>
          </p>
          <div class="mt-7 flex w-full flex-col items-center gap-3 sm:w-auto sm:flex-row">
            <div ref="heroCta" class="w-full sm:w-auto">
              <router-link v-if="primaryCta" :to="primaryCta.to" class="hero-btn" data-testid="home-hero-cta" @click="trackCta(`hero-${primaryCta.where}`)">
                {{ primaryCta.label }} →
              </router-link>
            </div>
            <a href="#pricing" class="ghost-btn" data-testid="home-hero-pricing" @click="trackCta('hero-pricing')">{{ t('home.v3.hero.ctaPricing') }}</a>
          </div>
          <HomeTokenCounter class="mt-10" />
        </div>
        <p class="pointer-events-none absolute bottom-5 left-1/2 z-[2] hidden -translate-x-1/2 font-mono text-[10px] tracking-[0.3em] text-slate-500 sm:block" aria-hidden="true">SCROLL</p>
      </section>

      <!-- ② 一句话进去，作品出来: the heading decodes as tokens, what signing up gets you, and on the right a
           sentence turning into a piece of work (a real prompt box). Tokens stream past the words into the card. -->
      <section ref="trySection" class="relative overflow-hidden bg-[#050a14] text-white" data-testid="home-try">
        <div class="hero-glow pointer-events-none absolute inset-0" aria-hidden="true"></div>
        <div class="hero-grid pointer-events-none absolute inset-0" aria-hidden="true"></div>
        <HomeMotion kind="river" :context="riverContext" />
        <div class="relative z-[2] mx-auto grid max-w-6xl items-center gap-10 px-4 py-20 sm:px-6 lg:grid-cols-[1.05fr_1fr] lg:gap-14 lg:py-28">
          <div ref="tryCopy" class="relative">
            <HomeHeroSlogan as="h2" :lead="t('home.v3.try.titleLead')" :highlight="t('home.v3.try.titleHighlight')" :huds="heroHuds" />
            <p class="mt-5 max-w-xl text-base leading-relaxed text-slate-400 sm:text-lg">
              {{ t('home.v3.hero.subtitleLead') }}<span class="text-white">{{ t('home.v3.hero.subtitleEm') }}</span>{{ t('home.v3.hero.subtitleTail') }}
            </p>

            <!-- What signing up gets you, in pictures and chats rather than dollars -->
            <div
              v-if="showOffer"
              class="mt-7 flex max-w-lg items-center gap-4 rounded-2xl border border-amber-300/30 bg-gradient-to-r from-amber-300/15 to-amber-300/[0.03] p-3 pr-5"
              data-testid="home-offer"
            >
              <span class="grid h-14 w-14 shrink-0 place-items-center rounded-xl bg-amber-300 text-2xl font-extrabold tracking-tight text-amber-950">${{ signupBonusText }}</span>
              <span>
                <span class="block font-semibold text-white">{{ t('home.v3.hero.offerTitle', { amount: signupBonusText }) }}</span>
                <span class="mt-0.5 block text-sm text-amber-100/70">{{ offerLine }}</span>
              </span>
            </div>

            <div class="mt-7 flex flex-wrap gap-3">
              <router-link v-if="primaryCta" :to="primaryCta.to" class="hero-btn" data-testid="home-try-cta" @click="trackCta(`try-${primaryCta.where}`)">
                {{ primaryCta.label }} →
              </router-link>
              <a href="#teardown" class="ghost-btn hidden sm:inline-flex">{{ t('home.v3.hero.ctaSecondary') }}</a>
            </div>
          </div>

          <div ref="heroCard"><HomeHeroDemo @generate="onGenerate" /></div>
        </div>
      </section>

      <!-- ② 拆开看看: one sentence → a picture, as an exploded view -->
      <HomeTeardown :site-name="siteName" :image-cost-usd="pricing?.starter_image_cost_usd" />

      <div class="mx-auto max-w-6xl px-4 pt-20 sm:px-6">
        <HomeJobs />
      </div>

      <!-- ④ 数据在流动: the honeycomb floor and the real totals -->
      <HomeLiveStats :total-tokens="usage?.total_tokens" :total-requests="usage?.total_requests" />

      <div id="pricing" class="mx-auto max-w-6xl scroll-mt-16 px-4 pb-4 pt-20 sm:px-6">
        <HomePricing :is-authenticated="isAuthenticated" :signup-bonus="signupBonus" :contact-info="contactInfo" />
        <HomeCommunityWall />
        <HomeFaq :site-name="siteName" />
      </div>

      <!-- ⑥ Final: the slogan once more, and the same buttons as the first screen -->
      <section class="relative overflow-hidden bg-[#050a14] py-24 text-white sm:py-32" data-testid="home-finale">
        <div class="stage-glow pointer-events-none absolute inset-0" aria-hidden="true"></div>
        <div class="hero-grid pointer-events-none absolute inset-0" aria-hidden="true"></div>
        <div class="relative z-[2] mx-auto max-w-4xl px-4 text-center">
          <h2 class="break-keep text-4xl font-extrabold tracking-tight sm:text-6xl">
            {{ t('home.v3.hero.titleLead') }}{{ heroTitleGap }}<span class="flow-text">{{ t('home.v3.hero.titleHighlight') }}</span>
          </h2>
          <p class="mt-6 text-slate-300">
            <template v-for="(part, i) in ctaLine" :key="i"><span v-if="i" class="px-1.5 text-slate-500">·</span><span class="inline-block">{{ part }}</span></template>
          </p>
          <div class="mt-7 flex flex-col items-center justify-center gap-3 sm:flex-row">
            <router-link v-if="primaryCta" :to="primaryCta.to" class="hero-btn" data-testid="home-final-cta" @click="trackCta(primaryCta.where === 'signup' ? 'final-cta' : `final-${primaryCta.where}`)">
              {{ primaryCta.label }} →
            </router-link>
            <a href="#pricing" class="ghost-btn" @click="trackCta('final-pricing')">{{ t('home.v3.hero.ctaPricing') }}</a>
          </div>
        </div>
      </section>
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
            <li><router-link to="/games" class="hover:text-gray-900 dark:hover:text-white">{{ t('games.title') }}</router-link></li>
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
import { ref, computed, onBeforeUnmount, onMounted, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAuthStore, useAppStore } from '@/stores'
import LocaleSwitcher from '@/components/common/LocaleSwitcher.vue'
import Icon from '@/components/icons/Icon.vue'
import { sanitizeUrl } from '@/utils/url'
import { track } from '@/utils/analytics'
import { FeatureFlags, isFeatureFlagEnabled } from '@/utils/featureFlags'
import { canvasUrl, otherPartnerSites, partnerSiteUrl } from '@/constants/crossSites'
import { getPublicPricing, getUsageTotals, signupCreditEstimate, type PublicPricing, type UsageTotals } from '@/api/pricing'
import { savePendingPrompt } from '@/utils/pendingPrompt'
import { growthAPI } from '@/api/growth'
import { paymentAPI } from '@/api/payment'
import { detectMotion } from '@/components/home/motion/tier'
import HomeHeroDemo from '@/components/home/HomeHeroDemo.vue'
import HomeHeroSlogan from '@/components/home/HomeHeroSlogan.vue'
import HomeTokenCounter from '@/components/home/HomeTokenCounter.vue'
import HomeMotion from '@/components/home/HomeMotion.vue'
import HomeTeardown from '@/components/home/HomeTeardown.vue'
import HomeLiveStats from '@/components/home/HomeLiveStats.vue'
import HomeJobs from '@/components/home/HomeJobs.vue'
import HomeSignupSheet from '@/components/home/HomeSignupSheet.vue'
import HomePricing from '@/components/home/HomePricing.vue'
import HomeCommunityWall from '@/components/home/HomeCommunityWall.vue'
import HomeAssistant from '@/components/home/HomeAssistant.vue'
import HomePromoMascot from '@/components/home/HomePromoMascot.vue'
import HomeFaq from '@/components/home/HomeFaq.vue'

const { t } = useI18n()

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

const chatLink = '/chat?utm_source=home&utm_medium=nav'
const navLinks = [
  { key: 'chat', href: chatLink, external: false },
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
const usage = ref<UsageTotals | null>(null)

// Motion screens. The first screen's particles take the place of the heading's two lines; the
// heading stays hidden while they load ('wait') and shows itself when there is no motion or it took
// too long ('text'). The river on the second screen needs to know where the words and the card are.
const heroSection = ref<HTMLElement | null>(null)
const heroLine1 = ref<HTMLElement | null>(null)
const heroLine2 = ref<HTMLElement | null>(null)
const heroTitleMode = ref<'wait' | 'motion' | 'text'>(detectMotion() ? 'wait' : 'text')
// 「你当老板，」 ends in a full-width comma; "You're the boss." needs a space before the next line
const heroTitleGap = computed(() => (/[，。、！？]$/.test(t('home.v3.hero.titleLead')) ? '' : ' '))
const assembleContext = () => ({
  anchors: () => [heroLine1.value, heroLine2.value].filter((el): el is HTMLElement => !!el),
  screen: heroSection.value!,
})
const trySection = ref<HTMLElement | null>(null)
const tryCopy = ref<HTMLElement | null>(null)
const heroCard = ref<HTMLElement | null>(null)
const riverContext = () => ({ host: trySection.value!, text: () => (tryCopy.value ? [tryCopy.value] : []), card: () => heroCard.value })
const heroHuds = computed(() => [
  { idx: 'INPUT', name: t('home.v3.motion.hud.input'), spec: t('home.v3.motion.hud.inputSpec') },
  { idx: 'MODEL', name: t('home.v3.motion.hud.model'), spec: t('home.v3.motion.hud.modelSpec') },
])

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

// The big button: visitors sign up; members go to the top-up page rather than the console, with
// the first top-up bonus on the button while they still qualify.
const paymentEnabled = computed(() => appStore.cachedPublicSettings?.payment_enabled === true)
const firstTopup = ref<{ percent: number; cap: number; min: number } | null>(null)
const money = (n: number) => String(Math.round(n * 100) / 100)
const primaryCta = computed<{ to: string; label: string; where: string } | null>(() => {
  if (isAuthenticated.value) {
    if (!paymentEnabled.value) return { to: dashboardPath.value, label: t('home.v3.hero.ctaDashboard'), where: 'dashboard' }
    return firstTopup.value
      ? { to: '/purchase', label: t('home.v3.hero.ctaTopupBonus', { percent: firstTopup.value.percent }), where: 'topup' }
      : { to: '/purchase', label: t('home.v3.hero.ctaTopup'), where: 'topup' }
  }
  if (!registrationEnabled.value) return null
  return { to: '/register', label: signupBonus.value > 0 ? t('home.v3.hero.cta', { amount: signupBonusText.value }) : t('home.v3.hero.ctaNoBonus'), where: 'signup' }
})
// The line above it says what the button gets you, in parts that wrap as wholes.
const ctaLine = computed<string[]>(() => {
  if (isAuthenticated.value) {
    const f = firstTopup.value
    if (!paymentEnabled.value || !f) return [t('home.v3.hero.linePayInstant'), t('home.v3.hero.linePayMetered'), t('home.v3.hero.linePlans')]
    return [
      t('home.v3.hero.lineFirstTopup', { percent: f.percent }),
      f.min > 0 ? t('home.v3.hero.lineFirstTopupMin', { min: money(f.min) }) : '',
      f.cap > 0 ? t('home.v3.hero.lineFirstTopupCap', { cap: money(f.cap) }) : '',
      t('home.v3.hero.linePayInstant'),
    ].filter(Boolean)
  }
  const pay = [t('home.v3.hero.linePay'), t('home.v3.hero.linePayMetered')]
  if (signupBonus.value <= 0) return pay
  const images = estimate.value.images
  const bonus = images > 0 ? t('home.v3.hero.lineSignupImages', { amount: signupBonusText.value, images }) : t('home.v3.hero.lineSignup', { amount: signupBonusText.value })
  return [bonus, ...pay]
})
async function loadFirstTopup() {
  if (!isAuthenticated.value || !paymentEnabled.value || firstTopup.value) return
  try {
    const cfg = await growthAPI.getPublicConfig()
    const percent = Number(cfg.first_topup_bonus_percent || 0)
    if (!percent) return
    const { data } = await paymentAPI.getCheckoutInfo()
    if (data.first_topup_eligible) firstTopup.value = { percent, cap: Number(cfg.first_topup_bonus_cap || 0), min: Number(cfg.first_topup_min_amount || 0) }
  } catch {
    // the plain top-up button still works
  }
}
watch([isAuthenticated, paymentEnabled], () => void loadFirstTopup())
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
  getUsageTotals()
    .then((u) => (usage.value = u))
    .catch(() => {})
  void loadFirstTopup()

  if (heroCta.value && typeof IntersectionObserver !== 'undefined') {
    heroObserver = new IntersectionObserver(([entry]) => {
      mobileBarVisible.value = !entry.isIntersecting && entry.boundingClientRect.top < 0
    })
    heroObserver.observe(heroCta.value)
  }
})

onBeforeUnmount(() => heroObserver?.disconnect())
</script>

<style scoped>
.hero-glow {
  background:
    radial-gradient(55% 60% at 72% 45%, rgba(37, 99, 235, 0.28), transparent 70%),
    radial-gradient(40% 50% at 10% 15%, rgba(94, 231, 255, 0.1), transparent 70%);
}
.hero-grid {
  opacity: 0.55;
  background-image:
    linear-gradient(rgba(94, 231, 255, 0.06) 1px, transparent 1px),
    linear-gradient(90deg, rgba(94, 231, 255, 0.06) 1px, transparent 1px);
  background-size: 56px 56px;
  -webkit-mask-image: radial-gradient(70% 70% at 60% 40%, #000, transparent);
  mask-image: radial-gradient(70% 70% at 60% 40%, #000, transparent);
}
.stage-glow {
  background: radial-gradient(50% 45% at 50% 42%, rgba(37, 99, 235, 0.22), transparent 70%);
}
/* the first screen's heading: the particles draw it in its place, so it is transparent while they
   load and run; each line is one unbroken box, so a narrow screen puts the lines one above the other */
.hero-title {
  font-size: clamp(44px, 7.2vw, 118px);
  font-weight: 800;
  line-height: 1.12;
  letter-spacing: 0;
}
.hero-title > span {
  display: inline-block;
  white-space: nowrap;
}
.hero-title.wait > span,
.hero-title.motion > span {
  color: transparent;
}
.hero-title.text {
  animation: title-in 0.6s ease-out;
}
.hero-title.text .t2,
.flow-text {
  background: linear-gradient(100deg, #9fdcff 0%, #5ee7ff 30%, #3b82f6 70%, #5ee7ff 100%);
  background-size: 220% 100%;
  -webkit-background-clip: text;
  background-clip: text;
  color: transparent;
  animation: flow 5s linear infinite;
}
@keyframes title-in {
  from { opacity: 0; transform: translateY(8px); }
}
@keyframes flow {
  to { background-position: -220% 0; }
}
@media (max-width: 639px) {
  .hero-title {
    font-size: min(14.5vw, 84px);
  }
}
@media (prefers-reduced-motion: reduce) {
  .hero-title.text, .hero-title.text .t2, .flow-text { animation: none; }
}
/* the one big button on the dark screens */
.hero-btn {
  display: inline-flex;
  width: 100%;
  align-items: center;
  justify-content: center;
  border-radius: 16px;
  padding: 14px 28px;
  font-size: 17px;
  font-weight: 600;
  color: #fff;
  background: linear-gradient(100deg, #2563eb, #3b82f6 55%, #22b8e6);
  box-shadow: 0 10px 30px -10px rgba(59, 130, 246, 0.8), inset 0 1px 0 rgba(255, 255, 255, 0.25);
  transition: filter 0.15s, transform 0.15s;
}
.hero-btn:hover {
  filter: brightness(1.1);
}
.hero-btn:active {
  transform: translateY(1px);
}
/* the quieter one next to it */
.ghost-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border-radius: 16px;
  border: 1px solid rgba(255, 255, 255, 0.15);
  padding: 13px 24px;
  font-size: 16px;
  color: #cbd5e1;
  transition: border-color 0.15s, color 0.15s;
}
.ghost-btn:hover {
  border-color: rgba(103, 232, 249, 0.5);
  color: #fff;
}
@media (min-width: 640px) {
  .hero-btn {
    width: auto;
    font-size: 18px;
  }
}
</style>
