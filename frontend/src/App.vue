<script setup lang="ts">
import { computed, ref } from 'vue'
import ImportView from './views/ImportView.vue'
import DashboardView from './views/DashboardView.vue'
import TransactionsView from './views/TransactionsView.vue'
import CategoriesView from './views/CategoriesView.vue'
import DataView from './views/DataView.vue'
import SettingsView from './views/SettingsView.vue'

type ViewKey = 'import' | 'dashboard' | 'transactions' | 'categories' | 'data' | 'settings'

const activeView = ref<ViewKey>('import')

const navItems: Array<{ key: ViewKey; label: string }> = [
  { key: 'import', label: 'インポート' },
  { key: 'dashboard', label: 'ダッシュボード' },
  { key: 'transactions', label: '明細一覧' },
  { key: 'categories', label: 'カテゴリ・ルール' },
  { key: 'data', label: 'データ管理' },
  { key: 'settings', label: '設定' },
]

const title = computed(() => navItems.find((item) => item.key === activeView.value)?.label ?? 'インポート')
</script>

<template>
  <div class="app-shell">
    <aside class="sidebar" aria-label="主要画面">
      <div class="brand">
        <span class="brand-mark">V</span>
        <div>
          <strong>Vpass明細分析</strong>
          <small>Local SQLite</small>
        </div>
      </div>
      <nav class="nav-list">
        <button
          v-for="item in navItems"
          :key="item.key"
          class="nav-item"
          :class="{ active: activeView === item.key }"
          type="button"
          @click="activeView = item.key"
        >
          {{ item.label }}
        </button>
      </nav>
    </aside>

    <main class="main">
      <header class="topbar">
        <h1>{{ title }}</h1>
        <span class="status-pill">ローカル実行</span>
      </header>

      <ImportView v-if="activeView === 'import'" @imported="activeView = 'transactions'" />
      <DashboardView v-else-if="activeView === 'dashboard'" @go-import="activeView = 'import'" />
      <TransactionsView v-else-if="activeView === 'transactions'" @go-import="activeView = 'import'" />
      <CategoriesView v-else-if="activeView === 'categories'" />
      <DataView v-else-if="activeView === 'data'" />
      <SettingsView v-else />
    </main>
  </div>
</template>

