import DefaultTheme from 'vitepress/theme'
import type { Theme } from 'vitepress'
import Layout from './Layout.vue'
import RunBox from './components/RunBox.vue'
import TryInCanvas from './components/TryInCanvas.vue'
import Mermaid from './components/Mermaid.vue'
import LearnHome from './components/LearnHome.vue'
import TrackPage from './components/TrackPage.vue'
import './style.css'

export default {
  extends: DefaultTheme,
  Layout,
  enhanceApp({ app }) {
    app.component('RunBox', RunBox)
    app.component('TryInCanvas', TryInCanvas)
    app.component('Mermaid', Mermaid)
    app.component('LearnHome', LearnHome)
    app.component('TrackPage', TrackPage)
  },
} satisfies Theme
