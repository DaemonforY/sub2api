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
    app.component('Checkpoint', Checkpoint)
    app.component('CertPanel', CertPanel)
    app.component('CertView', CertView)
    app.component('MockInterview', MockInterview)
    app.component('AgentLoop', AgentLoop)
  },
} satisfies Theme
