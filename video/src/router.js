import { createRouter, createWebHistory } from 'vue-router'

export const router = createRouter({
  history: createWebHistory(),
  scrollBehavior: (to, from, saved) => saved || (to.hash ? { el: to.hash } : { top: 0 }),
  routes: [
    { path: '/', name: 'home', component: () => import('./pages/Home.vue') },
    { path: '/create', redirect: (to) => ({ path: '/', query: to.query }) },
    { path: '/p/:id', name: 'project', component: () => import('./pages/Project.vue'), meta: { full: true } },
    { path: '/gallery/:category?', name: 'gallery', component: () => import('./pages/Gallery.vue') },
    { path: '/w/:id', name: 'work', component: () => import('./pages/Work.vue') },
    { path: '/tools', name: 'tools', component: () => import('./pages/Tools.vue') },
    { path: '/tools/:slug', name: 'tool', component: () => import('./pages/Tool.vue') },
    { path: '/pricing', name: 'pricing', component: () => import('./pages/Pricing.vue') },
    { path: '/review', name: 'review', component: () => import('./pages/Review.vue') },
    { path: '/:rest(.*)*', name: 'missing', component: () => import('./pages/Missing.vue') }
  ]
})
