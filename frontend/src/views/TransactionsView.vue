<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { listCategories, listTransactions, updateTransaction } from '../api/client'
import type { Category, Transaction } from '../api/types'

defineEmits<{ goImport: [] }>()

const rows = ref<Transaction[]>([])
const categories = ref<Category[]>([])
const loading = ref(false)
const error = ref('')
const page = ref(1)
const billingMonth = ref('')
const keyword = ref('')

async function load() {
  loading.value = true
  error.value = ''
  try {
    const params = new URLSearchParams({ page: String(page.value), pageSize: '50', sort: 'usageDate', order: 'desc' })
    if (billingMonth.value) params.set('billingMonth', billingMonth.value)
    if (keyword.value) params.set('keyword', keyword.value)
    const [txResult, categoryResult] = await Promise.all([listTransactions(params), listCategories()])
    rows.value = txResult.items
    categories.value = categoryResult.items
  } catch {
    error.value = '明細を取得できませんでした'
  } finally {
    loading.value = false
  }
}

async function changeCategory(row: Transaction, value: string) {
  const previous = row.categoryId
  row.categoryId = value ? Number(value) : null
  try {
    await updateTransaction(row.id, { categoryId: value || '' })
  } catch {
    row.categoryId = previous
    error.value = 'カテゴリ更新に失敗しました'
  }
}

onMounted(load)
</script>

<template>
  <section class="screen-stack">
    <div class="panel toolbar">
      <label>
        請求月
        <input v-model="billingMonth" type="month" @change="page = 1; load()" />
      </label>
      <label>
        キーワード
        <input v-model="keyword" type="search" @keydown.enter="page = 1; load()" />
      </label>
      <button type="button" :disabled="loading" @click="page = 1; load()">検索</button>
    </div>
    <p v-if="error" class="error-line">{{ error }}</p>
    <div class="panel table-wrap">
      <table>
        <thead>
          <tr>
            <th>利用日</th>
            <th>利用先</th>
            <th>請求月</th>
            <th>利用金額</th>
            <th>請求金額</th>
            <th>カテゴリ</th>
            <th>メモ</th>
          </tr>
        </thead>
        <tbody>
          <tr v-if="rows.length === 0">
            <td colspan="7" class="empty">明細がありません。</td>
          </tr>
          <tr v-for="row in rows" :key="row.id">
            <td>{{ String(row.usageDate).slice(0, 10) }}</td>
            <td>{{ row.merchantName }}</td>
            <td>{{ row.billingMonth }}</td>
            <td>{{ row.usageAmount?.toLocaleString() ?? '-' }}</td>
            <td>{{ row.billedAmount?.toLocaleString() ?? '-' }}</td>
            <td>
              <select :value="row.categoryId ?? ''" @change="changeCategory(row, ($event.target as HTMLSelectElement).value)">
                <option value="">未分類</option>
                <option v-for="category in categories" :key="category.id" :value="category.id">{{ category.name }}</option>
              </select>
            </td>
            <td>{{ row.memo }}</td>
          </tr>
        </tbody>
      </table>
    </div>
  </section>
</template>

