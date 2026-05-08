<script setup lang="ts">
import { onMounted, ref } from 'vue'
import {
  applyCategoryRules,
  createCategory,
  createCategoryRule,
  deleteCategory,
  listCategories,
  listCategoryRules,
  listClassificationCandidates,
} from '../api/client'
import type { Category, CategoryRule, ClassificationCandidate } from '../api/types'

const categories = ref<Category[]>([])
const rules = ref<CategoryRule[]>([])
const candidates = ref<ClassificationCandidate[]>([])
const loading = ref(false)
const saving = ref(false)
const error = ref('')
const message = ref('')
const categoryName = ref('')
const categoryColor = ref('#2563eb')
const rulePattern = ref('')
const ruleMatchType = ref<CategoryRule['matchType']>('contains')
const ruleCategoryId = ref('')
const overwriteManualCategory = ref(false)
const candidateCategoryIds = ref<Record<string, string>>({})

async function load() {
  loading.value = true
  error.value = ''
  try {
    const [categoryResult, ruleResult, candidateResult] = await Promise.all([listCategories(), listCategoryRules(), listClassificationCandidates()])
    categories.value = categoryResult.items
    rules.value = ruleResult.items
    candidates.value = candidateResult.items
  } catch {
    error.value = 'カテゴリ情報を取得できませんでした'
  } finally {
    loading.value = false
  }
}

async function addCategory() {
  if (!categoryName.value.trim()) return
  saving.value = true
  error.value = ''
  try {
    await createCategory({ name: categoryName.value.trim(), color: categoryColor.value })
    categoryName.value = ''
    await load()
    message.value = 'カテゴリを作成しました'
  } catch {
    error.value = 'カテゴリを作成できませんでした'
  } finally {
    saving.value = false
  }
}

async function removeCategory(category: Category) {
  if (!window.confirm(`${category.name} を削除します。紐づく明細は未分類に戻ります。`)) return
  saving.value = true
  try {
    await deleteCategory(category.id)
    await load()
    message.value = 'カテゴリを削除しました'
  } catch {
    error.value = 'カテゴリを削除できませんでした'
  } finally {
    saving.value = false
  }
}

async function addRule() {
  if (!rulePattern.value.trim() || !ruleCategoryId.value) return
  saving.value = true
  error.value = ''
  try {
    await createCategoryRule({
      matchType: ruleMatchType.value,
      pattern: rulePattern.value.trim(),
      categoryId: Number(ruleCategoryId.value),
      priority: rules.value.length + 1,
    })
    rulePattern.value = ''
    await load()
    message.value = '分類ルールを作成しました'
  } catch {
    error.value = '分類ルールを作成できませんでした'
  } finally {
    saving.value = false
  }
}

async function createRuleFromCandidate(candidate: ClassificationCandidate) {
  const categoryId = candidateCategoryIds.value[candidate.merchantName]
  if (!categoryId) return
  saving.value = true
  error.value = ''
  try {
    await createCategoryRule({
      matchType: 'contains',
      pattern: candidate.merchantName,
      categoryId: Number(categoryId),
      priority: rules.value.length + 1,
    })
    delete candidateCategoryIds.value[candidate.merchantName]
    await load()
    message.value = '分類ルールを作成しました'
  } catch {
    error.value = '分類ルールを作成できませんでした'
  } finally {
    saving.value = false
  }
}

async function reapplyRules() {
  if (!window.confirm('分類ルールを既存明細へ再適用します。')) return
  saving.value = true
  error.value = ''
  try {
    const result = await applyCategoryRules(overwriteManualCategory.value)
    message.value = `再適用しました: 更新 ${result.updatedCount} 件 / 一致 ${result.matchedCount} 件`
  } catch {
    error.value = '分類ルールを再適用できませんでした'
  } finally {
    saving.value = false
  }
}

onMounted(load)
</script>

<template>
  <section class="screen-stack">
    <p v-if="error" class="error-line">{{ error }}</p>
    <p v-if="message" class="success-line">{{ message }}</p>

    <div class="two-column">
      <div class="panel">
        <h2>カテゴリ</h2>
        <div class="inline-form">
          <input v-model="categoryName" type="text" placeholder="カテゴリ名" />
          <input v-model="categoryColor" type="color" aria-label="カテゴリ色" />
          <button type="button" :disabled="saving" @click="addCategory">追加</button>
        </div>
        <div v-if="categories.length === 0" class="empty">カテゴリがありません。</div>
        <ul class="item-list">
          <li v-for="category in categories" :key="category.id">
            <span class="color-dot" :style="{ background: category.color }" />
            <span>{{ category.name }}</span>
            <button type="button" :disabled="saving" @click="removeCategory(category)">削除</button>
          </li>
        </ul>
      </div>

      <div class="panel">
        <h2>分類ルール</h2>
        <div class="inline-form stackable">
          <select v-model="ruleMatchType">
            <option value="contains">含む</option>
            <option value="startsWith">前方一致</option>
            <option value="equals">完全一致</option>
            <option value="regex">正規表現</option>
          </select>
          <input v-model="rulePattern" type="text" placeholder="利用先パターン" />
          <select v-model="ruleCategoryId">
            <option value="">カテゴリ</option>
            <option v-for="category in categories" :key="category.id" :value="category.id">{{ category.name }}</option>
          </select>
          <button type="button" :disabled="saving" @click="addRule">追加</button>
        </div>
        <ul class="item-list">
          <li v-for="rule in rules" :key="rule.id">
            <span class="badge">{{ rule.matchType }}</span>
            <span>{{ rule.pattern }}</span>
            <strong>#{{ rule.categoryId }}</strong>
          </li>
        </ul>
        <div class="divider" />
        <label class="check-row">
          <input v-model="overwriteManualCategory" type="checkbox" />
          手動カテゴリも上書きする
        </label>
        <button type="button" :disabled="saving || rules.length === 0" @click="reapplyRules">分類ルールを再適用</button>
      </div>
    </div>

    <div class="panel">
      <h2>未分類候補</h2>
      <div v-if="candidates.length === 0" class="empty">未分類の利用先はありません。</div>
      <div v-else class="table-wrap compact-table">
        <table>
          <thead>
            <tr>
              <th>利用先</th>
              <th>件数</th>
              <th>カテゴリ</th>
              <th>操作</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="candidate in candidates" :key="candidate.merchantName">
              <td>{{ candidate.merchantName }}</td>
              <td>{{ candidate.transactionCount }}</td>
              <td>
                <select v-model="candidateCategoryIds[candidate.merchantName]" :disabled="saving">
                  <option value="">カテゴリ</option>
                  <option v-for="category in categories" :key="category.id" :value="category.id">{{ category.name }}</option>
                </select>
              </td>
              <td>
                <button type="button" :disabled="saving || !candidateCategoryIds[candidate.merchantName]" @click="createRuleFromCandidate(candidate)">
                  ルール化
                </button>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>
  </section>
</template>
