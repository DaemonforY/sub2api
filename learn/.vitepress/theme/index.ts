import DefaultTheme from 'vitepress/theme'
import type { Theme } from 'vitepress'
import Layout from './Layout.vue'
import RunBox from './components/RunBox.vue'
import TryInCanvas from './components/TryInCanvas.vue'
import Mermaid from './components/Mermaid.vue'
import LearnHome from './components/LearnHome.vue'
import TrackPage from './components/TrackPage.vue'
import Checkpoint from './components/Checkpoint.vue'
import CertPanel from './components/CertPanel.vue'
import CertView from './components/CertView.vue'
import MockInterview from './components/MockInterview.vue'
import AgentLoop from './components/AgentLoop.vue'
import Showcase from './components/Showcase.vue'
import CampusPerks from './components/CampusPerks.vue'
import BigdataHub from './components/BigdataHub.vue'
import { initAnalytics, trackPageView } from './analytics'
import './style.css'
import './heroui.css'

// Remember ?aff= (poster and invite links) the way the main site does: same origin, same
// localStorage key and 30-day lifetime, so signing up from a lesson still binds the inviter.
function rememberAffiliate() {
  try {
    const q = new URLSearchParams(window.location.search)
    const code = (q.get('aff') || q.get('aff_code') || '').trim()
    if (code && /^[A-Za-z0-9_-]{1,64}$/.test(code)) {
      const expiresAt = Date.now() + 30 * 24 * 60 * 60 * 1000
      window.localStorage.setItem('affiliate_referral_code', JSON.stringify({ code, expiresAt }))
    }
  } catch {
    // storage unavailable
  }
}

export default {
  extends: DefaultTheme,
  Layout,
  enhanceApp({ app, router }) {
    if (typeof window !== 'undefined') {
      rememberAffiliate()
      // 埋点: the first page, then every client-side navigation (".html" dropped).
      initAnalytics('learn')
      const view = (href: string) => trackPageView(new URL(href, window.location.origin).pathname.replace(/\.html$/, ''))
      view(window.location.href)
      router.onAfterRouteChange = view
    }
    app.component('RunBox', RunBox)
    app.component('TryInCanvas', TryInCanvas)
    app.component('Mermaid', Mermaid)
    app.component('LearnHome', LearnHome)
    app.component('TrackPage', TrackPage)
    app.component('Checkpoint', Checkpoint)
    app.component('CertPanel', CertPanel)
    app.component('CertView', CertView)
    app.component('MockInterview', MockInterview)
    app.component('AgentLoop', AgentLoop)
    app.component('Showcase', Showcase)
    app.component('CampusPerks', CampusPerks)
    app.component('BigdataHub', BigdataHub)
  },
} satisfies Theme
