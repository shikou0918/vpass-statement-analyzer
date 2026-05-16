<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { listCategories, listCreditCards, listTransactions, updateTransaction } from '../api/client'
import type { Category, CreditCard, Transaction } from '../api/types'

const rows = ref<Transaction[]>([])
const categories = ref<Category[]>([])
const creditCards = ref<CreditCard[]>([])
const loading = ref(false)
const bulkSaving = ref(false)
const error = ref('')
const message = ref('')
const page = ref(1)
const billingMonth = ref('')
const creditCardId = ref('')
const keyword = ref('')
const selectedIds = ref<Set<number>>(new Set())
const bulkCategoryId = ref('')

const selectedRows = computed(() => rows.value.filter((row) => selectedIds.value.has(row.id)))
const allRowsSelected = computed(() => rows.value.length > 0 && selectedIds.value.size === rows.value.length)
const someRowsSelected = computed(() => selectedIds.value.size > 0 && !allRowsSelected.value)
const creditCardNameById = computed(() => new Map(creditCards.value.map((card) => [card.id, card.displayName])))

async function load() {
  loading.value = true
  error.value = ''
  message.value = ''
  try {
    const params = new URLSearchParams({ page: String(page.value), pageSize: '50', sort: 'usageDate', order: 'desc' })
    if (billingMonth.value) params.set('billingMonth', billingMonth.value)
    if (creditCardId.value) params.set('creditCardId', creditCardId.value)
    if (keyword.value) params.set('keyword', keyword.value)
    const [txResult, categoryResult, cardResult] = await Promise.all([listTransactions(params), listCategories(), listCreditCards()])
    rows.value = txResult.items
    categories.value = categoryResult.items
    creditCards.value = cardResult.items
    selectedIds.value = new Set()
  } catch {
    error.value = '明細を取得できませんでした'
  } finally {
    loading.value = false
  }
}

function toggleAllRows(checked: boolean) {
  selectedIds.value = checked ? new Set(rows.value.map((row) => row.id)) : new Set()
}

function toggleRow(row: Transaction, checked: boolean) {
  const next = new Set(selectedIds.value)
  if (checked) {
    next.add(row.id)
  } else {
    next.delete(row.id)
  }
  selectedIds.value = next
}

async function changeCategory(row: Transaction, value: string) {
  const previous = row.categoryId
  row.categoryId = value ? Number(value) : null
  error.value = ''
  message.value = ''
  try {
    await updateTransaction(row.id, { categoryId: value || '' })
  } catch {
    row.categoryId = previous
    error.value = 'カテゴリ更新に失敗しました'
  }
}

async function applyBulkCategory() {
  const targets = selectedRows.value
  if (targets.length === 0) return

  bulkSaving.value = true
  error.value = ''
  message.value = ''
  const nextCategoryId = bulkCategoryId.value ? Number(bulkCategoryId.value) : null
  const previous = new Map(targets.map((row) => [row.id, row.categoryId]))
  targets.forEach((row) => {
    row.categoryId = nextCategoryId
  })

  try {
    await Promise.all(targets.map((row) => updateTransaction(row.id, { categoryId: bulkCategoryId.value || '' })))
    message.value = `${targets.length}件のカテゴリを更新しました`
    selectedIds.value = new Set()
  } catch {
    targets.forEach((row) => {
      row.categoryId = previous.get(row.id) ?? null
    })
    error.value = '一括カテゴリ更新に失敗しました'
  } finally {
    bulkSaving.value = false
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
        カード
        <select v-model="creditCardId" @change="page = 1; load()">
          <option value="">すべて</option>
          <option v-for="card in creditCards" :key="card.id" :value="String(card.id)">{{ card.displayName }}</option>
        </select>
      </label>
      <label>
        キーワード
        <input v-model="keyword" type="search" @keydown.enter="page = 1; load()" />
      </label>
      <button type="button" :disabled="loading" @click="page = 1; load()">検索</button>
    </div>
    <p v-if="error" class="error-line">{{ error }}</p>
    <p v-if="message" class="success-line">{{ message }}</p>
    <div class="panel toolbar">
      <span class="muted">選択中 {{ selectedIds.size }} 件</span>
      <label>
        一括カテゴリ
        <select v-model="bulkCategoryId" :disabled="bulkSaving || selectedIds.size === 0">
          <option value="">未分類</option>
          <option v-for="category in categories" :key="category.id" :value="category.id">{{ category.name }}</option>
        </select>
      </label>
      <button type="button" :disabled="bulkSaving || selectedIds.size === 0" @click="applyBulkCategory">
        {{ bulkSaving ? '更新中' : '選択行に適用' }}
      </button>
    </div>
    <div class="panel table-wrap">
      <table>
        <thead>
          <tr>
            <th class="select-cell">
              <input
                type="checkbox"
                :checked="allRowsSelected"
                :indeterminate.prop="someRowsSelected"
                :disabled="loading || rows.length === 0"
                aria-label="表示中の明細をすべて選択"
                @change="toggleAllRows(($event.target as HTMLInputElement).checked)"
              />
            </th>
            <th>利用日</th>
            <th>カード</th>
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
            <td colspan="9" class="empty">明細がありません。</td>
          </tr>
          <tr v-for="row in rows" :key="row.id">
            <td class="select-cell">
              <input
                type="checkbox"
                :checked="selectedIds.has(row.id)"
                :disabled="bulkSaving"
                :aria-label="`${row.merchantName} を選択`"
                @change="toggleRow(row, ($event.target as HTMLInputElement).checked)"
              />
            </td>
            <td>{{ String(row.usageDate).slice(0, 10) }}</td>
            <td>{{ row.creditCardId ? creditCardNameById.get(row.creditCardId) ?? '-' : '-' }}</td>
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
