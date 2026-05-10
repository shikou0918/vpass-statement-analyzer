import { createRouter, createWebHistory } from 'vue-router'
import ImportView from './views/ImportView.vue'
import DashboardView from './views/DashboardView.vue'
import TransactionsView from './views/TransactionsView.vue'
import CategoriesView from './views/CategoriesView.vue'
import DataView from './views/DataView.vue'

export const routes = [
  { path: '/', redirect: '/imports/new' },
  { path: '/imports/new', name: 'import', component: ImportView, meta: { title: 'インポート' } },
  { path: '/dashboard', name: 'dashboard', component: DashboardView, meta: { title: 'ダッシュボード' } },
  { path: '/transactions', name: 'transactions', component: TransactionsView, meta: { title: '明細一覧' } },
  { path: '/categories', name: 'categories', component: CategoriesView, meta: { title: 'カテゴリ・ルール' } },
  { path: '/data', name: 'data', component: DataView, meta: { title: 'データ管理' } },
  { path: '/:pathMatch(.*)*', redirect: '/imports/new' },
]

export const router = createRouter({
  history: createWebHistory(),
  routes,
})
