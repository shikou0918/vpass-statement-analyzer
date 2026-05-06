<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { getCategorySummary, getMerchantSummary, getMonthlySummary } from '../api/client'
import type { CategorySummaryItem, MonthlySummary, RankingItem } from '../api/types'

defineEmits<{ goImport: [] }>()

const month = ref(new Date().toISOString().slice(0, 7))
const loading = ref(false)
const error = ref('')
const summary = ref<MonthlySummary | null>(null)
const merchants = ref<RankingItem[]>([])
const categories = ref<CategorySummaryItem[]>([])

async function load() {
  loading.value = true
  error.value = ''
  try {
    const [monthly, merchantResult, categoryResult] = await Promise.all([
      getMonthlySummary(month.value),
      getMerchantSummary(month.value),
      getCategorySummary(month.value),
    ])
    summary.value = monthly
    merchants.value = merchantResult.items
    categories.value = categoryResult.items
  } catch {
    error.value = '集計を取得できませんでした'
  } finally {
    loading.value = false
  }
}

onMounted(load)
</script>

<template>
  <section class="screen-stack">
    <div class="panel toolbar">
      <label>
        対象月
        <input v-model="month" type="month" />
      </label>
      <button type="button" :disabled="loading" @click="load">再読み込み</button>
    </div>
    <p v-if="error" class="error-line">{{ error }}</p>

    <div class="kpi-grid">
      <div class="kpi"><span>支出合計</span><strong>{{ summary?.totalAmount?.toLocaleString() ?? '-' }}円</strong></div>
      <div class="kpi"><span>前月比</span><strong>{{ summary?.diffAmount?.toLocaleString() ?? '-' }}円</strong></div>
      <div class="kpi"><span>明細件数</span><strong>{{ summary?.transactionCount ?? '-' }}</strong></div>
    </div>

    <div class="two-column">
      <div class="panel">
        <h2>カテゴリ内訳</h2>
        <div v-if="categories.length === 0" class="empty">対象月のカテゴリ集計がありません。</div>
        <div v-for="item in categories" :key="item.categoryName" class="bar-row">
          <span>{{ item.categoryName }}</span>
          <div class="bar"><i :style="{ width: `${Math.max(item.ratio * 100, 4)}%`, background: item.color }" /></div>
          <strong>{{ item.totalAmount.toLocaleString() }}円</strong>
        </div>
      </div>
      <div class="panel">
        <h2>利用先ランキング</h2>
        <div v-if="merchants.length === 0" class="empty">対象月の利用先集計がありません。</div>
        <ol class="ranking">
          <li v-for="item in merchants" :key="item.merchantName">
            <span>{{ item.merchantName }}</span>
            <strong>{{ item.totalAmount.toLocaleString() }}円</strong>
          </li>
        </ol>
      </div>
    </div>
  </section>
</template>

