<script setup lang="ts">
import Chart from 'chart.js/auto'
import { nextTick, onBeforeUnmount, onMounted, ref } from 'vue'
import { getCategorySummary, getMerchantSummary, getMonthlySummary, getMonthlyTrends } from '../api/client'
import type { CategorySummaryItem, ChartPoint, MonthlySummary, RankingItem } from '../api/types'

const month = ref(new Date().toISOString().slice(0, 7))
const loading = ref(false)
const error = ref('')
const summary = ref<MonthlySummary | null>(null)
const merchants = ref<RankingItem[]>([])
const categories = ref<CategorySummaryItem[]>([])
const monthlyTrends = ref<ChartPoint[]>([])
const monthlyTrendCanvas = ref<HTMLCanvasElement | null>(null)
let monthlyTrendChart: Chart | null = null

function formatYen(amount: number) {
  return `${amount.toLocaleString()}円`
}

function renderMonthlyTrendChart() {
  if (!monthlyTrendCanvas.value || monthlyTrends.value.length === 0) return

  monthlyTrendChart?.destroy()
  monthlyTrendChart = new Chart(monthlyTrendCanvas.value, {
    type: 'line',
    data: {
      labels: monthlyTrends.value.map((item) => item.label),
      datasets: [
        {
          label: '支出額',
          data: monthlyTrends.value.map((item) => item.amount),
          borderColor: '#174ea6',
          backgroundColor: 'rgb(23 78 166 / 0.12)',
          pointBackgroundColor: monthlyTrends.value.map((item) => (item.amount < 0 ? '#0f766e' : '#174ea6')),
          pointBorderColor: '#fff',
          pointBorderWidth: 2,
          pointRadius: 5,
          pointHoverRadius: 7,
          borderWidth: 2,
          fill: true,
          tension: 0.32,
        },
      ],
    },
    options: {
      responsive: true,
      maintainAspectRatio: false,
      animation: { duration: 220 },
      plugins: {
        legend: { display: false },
        tooltip: {
          displayColors: false,
          callbacks: {
            label: (context) => formatYen(Number(context.parsed.y ?? 0)),
          },
        },
      },
      scales: {
        x: {
          grid: { display: false },
          offset: false,
          ticks: { color: '#475569' },
        },
        y: {
          beginAtZero: true,
          grid: { color: '#e2e8f0' },
          ticks: {
            color: '#64748b',
            callback: (value) => formatYen(Number(value)),
          },
        },
      },
    },
  })
}

async function load() {
  loading.value = true
  error.value = ''
  try {
    const [monthly, merchantResult, categoryResult, monthlyTrendResult] = await Promise.all([
      getMonthlySummary(month.value),
      getMerchantSummary(month.value),
      getCategorySummary(month.value),
      getMonthlyTrends(),
    ])
    summary.value = monthly
    merchants.value = merchantResult.items ?? []
    categories.value = categoryResult.items ?? []
    monthlyTrends.value = monthlyTrendResult.items ?? []
    await nextTick()
    renderMonthlyTrendChart()
  } catch {
    error.value = '集計を取得できませんでした'
  } finally {
    loading.value = false
  }
}

onMounted(load)
onBeforeUnmount(() => monthlyTrendChart?.destroy())
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

    <div class="panel">
      <h2>月別支出推移</h2>
      <div v-if="monthlyTrends.length === 0" class="empty">月別推移を表示できる明細がありません。</div>
      <div v-else class="chart-panel" aria-label="月別支出推移">
        <canvas ref="monthlyTrendCanvas" />
      </div>
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
