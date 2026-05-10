<script setup lang="ts">
import { computed } from 'vue'
import { RouterLink, RouterView, useRoute } from 'vue-router'

const route = useRoute()

const navItems = [
  { to: '/imports/new', label: 'インポート' },
  { to: '/dashboard', label: 'ダッシュボード' },
  { to: '/transactions', label: '明細一覧' },
  { to: '/categories', label: 'カテゴリ・ルール' },
  { to: '/data', label: 'データ管理' },
  { to: '/settings', label: '設定' },
]

const title = computed(() => String(route.meta.title ?? 'インポート'))
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
        <RouterLink
          v-for="item in navItems"
          :key="item.to"
          :to="item.to"
          class="nav-item"
          active-class="active"
        >
          {{ item.label }}
        </RouterLink>
      </nav>
    </aside>

    <main class="main">
      <header class="topbar">
        <h1>{{ title }}</h1>
      </header>

      <RouterView />
    </main>
  </div>
</template>
