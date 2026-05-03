import { createRouter, createWebHashHistory } from 'vue-router'
import HomeView from '../views/Home.vue'

const router = createRouter({
  history: createWebHashHistory('/'),
  routes: [
    {
      path: '/',
      name: 'home',
      component: HomeView,
      meta: {
        showNav: true,
        title: '首页',
        home: true,
        icon: "&#xE80F;"
      },
    },
    {
      path: '/settings',
      name: 'settings',
      component: () => import('../views/Settings.vue'),
      meta: {
        showNav: true,
        title: '设置',
        home: true,
        icon: "&#xE713;"
      },
    }
  ],
})

export default router
