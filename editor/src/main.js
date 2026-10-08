import { createApp } from 'vue';
import 'highlight.js/styles/atom-one-dark.css';
import './editor.css';
import './features.css';
import App from './App.vue';
import { initAnalytics, trackPageView } from './lib/analytics.js';

initAnalytics('editor');
trackPageView();
createApp(App).mount('#app');
