import { createRouter, createWebHistory } from 'vue-router'

export const router = createRouter({
  history: createWebHistory(),
  scrollBehavior: (to, from, saved) => saved || (to.hash ? { el: to.hash } : { top: 0 }),
  routes: [
    { path: '/', name: 'home', component: () => import('./pages/Home.vue') },
    { path: '/create', redirect: (to) => ({ path: '/', query: to.query }) },
    { path: '/p/:id', name: 'project', component: () => import('./pages/Project.vue'), meta: { full: true } },
    { path: '/upload', name: 'upload', component: () => import('./pages/Upload.vue') },
    { path: '/gallery/:category?', name: 'gallery', component: () => import('./pages/Gallery.vue') },
    { path: '/w/:id', name: 'work', component: () => import('./pages/Work.vue') },
    { path: '/tools', name: 'tools', component: () => import('./pages/Tools.vue') },
    { path: '/tools/:slug', name: 'tool', component: () => import('./pages/Tool.vue') },
    { path: '/pricing', name: 'pricing', component: () => import('./pages/Pricing.vue') },
    { path: '/voices', name: 'voices', component: () => import('./pages/Voices.vue') },
    { path: '/review', name: 'review', component: () => import('./pages/Review.vue') },
    { path: '/:rest(.*)*', name: 'missing', component: () => import('./pages/Missing.vue') }
  ]
})

// After a deploy, a tab opened on the old build asks for chunks that no longer exist, so the
// navigation (e.g. to the new project after 生成) fails silently. Load the target page afresh;
// the timestamp stops a reload loop if the chunk is really broken.
router.onError((error, to) => {
  const msg = String(error?.message || '')
  const chunk = /Failed to fetch dynamically imported module|Importing a module script failed|error loading dynamically imported module|Unable to preload CSS/i.test(msg)
  if (!chunk) return
  const key = 'hivegpt-video:chunk-reload'
  const last = Number(sessionStorage.getItem(key) || 0)
  if (Date.now() - last < 10000) return
  sessionStorage.setItem(key, String(Date.now()))
  window.location.assign(to?.fullPath || window.location.href)
})
