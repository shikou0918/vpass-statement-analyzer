<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import {
  applyCategoryRules,
  createCategory,
  createCategoryRule,
  deleteCategory,
  ApiClientError,
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

const categoryNameById = computed(() => new Map(categories.value.map((category) => [category.id, category.name])))
const existingRuleKeys = computed(() => new Set(rules.value.map((rule) => ruleKey(rule.matchType, rule.pattern, rule.categoryId))))
const newRuleIsDuplicate = computed(
  () => Boolean(rulePattern.value.trim() && ruleCategoryId.value) && existingRuleKeys.value.has(ruleKey(ruleMatchType.value, rulePattern.value, ruleCategoryId.value)),
)
const canAddRule = computed(() => Boolean(rulePattern.value.trim() && ruleCategoryId.value) && !newRuleIsDuplicate.value)

function matchTypeLabel(matchType: CategoryRule['matchType']) {
  const labels: Record<CategoryRule['matchType'], string> = {
    contains: '含む',
    startsWith: '前方一致',
    equals: '完全一致',
    regex: '正規表現',
  }
  return labels[matchType]
}

function ruleKey(matchType: CategoryRule['matchType'], pattern: string, categoryId: string | number) {
  return `${matchType}:${pattern.trim()}:${categoryId}`
}

function candidateRuleExists(candidate: ClassificationCandidate) {
  const categoryId = candidateCategoryIds.value[candidate.merchantName]
  return Boolean(categoryId) && existingRuleKeys.value.has(ruleKey('contains', candidate.merchantName, categoryId))
}

function errorMessage(err: unknown, fallback: string) {
  return err instanceof ApiClientError ? err.apiError.message : fallback
}

async function applyRulesToUnclassified() {
  return applyCategoryRules(false)
}

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
  message.value = ''
  if (!rulePattern.value.trim()) {
    error.value = '利用先パターンを入力してください'
    return
  }
  if (!ruleCategoryId.value) {
    error.value = '分類先カテゴリを選択してください'
    return
  }
  if (newRuleIsDuplicate.value) {
    error.value = '同じ分類ルールが既に存在します'
    return
  }
  saving.value = true
  error.value = ''
  try {
    await createCategoryRule({
      matchType: ruleMatchType.value,
      pattern: rulePattern.value.trim(),
      categoryId: Number(ruleCategoryId.value),
      priority: rules.value.length + 1,
    })
    const result = await applyRulesToUnclassified()
    rulePattern.value = ''
    await load()
    message.value = `分類ルールを作成しました。未分類明細を ${result.updatedCount} 件更新しました`
  } catch (err) {
    error.value = errorMessage(err, '分類ルールを作成できませんでした')
  } finally {
    saving.value = false
  }
}

async function createRuleFromCandidate(candidate: ClassificationCandidate) {
  const categoryId = candidateCategoryIds.value[candidate.merchantName]
  message.value = ''
  if (!categoryId) {
    error.value = '分類先カテゴリを選択してください'
    return
  }
  if (candidateRuleExists(candidate)) {
    error.value = '同じ分類ルールが既に存在します'
    return
  }
  saving.value = true
  error.value = ''
  try {
    await createCategoryRule({
      matchType: 'contains',
      pattern: candidate.merchantName,
      categoryId: Number(categoryId),
      priority: rules.value.length + 1,
    })
    const result = await applyRulesToUnclassified()
    delete candidateCategoryIds.value[candidate.merchantName]
    await load()
    message.value = `分類ルールを作成しました。未分類明細を ${result.updatedCount} 件更新しました`
  } catch (err) {
    error.value = errorMessage(err, '分類ルールを作成できませんでした')
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
        <h2>自動分類ルール</h2>
        <div class="inline-form stackable">
          <label>
            条件
            <select v-model="ruleMatchType">
              <option value="contains">含む</option>
              <option value="startsWith">前方一致</option>
              <option value="equals">完全一致</option>
              <option value="regex">正規表現</option>
            </select>
          </label>
          <label>
            利用先パターン
            <input v-model="rulePattern" type="text" placeholder="例: セブン" />
          </label>
          <label>
            分類先カテゴリ
            <select v-model="ruleCategoryId">
              <option value="">カテゴリ</option>
              <option v-for="category in categories" :key="category.id" :value="category.id">{{ category.name }}</option>
            </select>
          </label>
          <button type="button" :disabled="saving || !canAddRule" @click="addRule">ルール追加</button>
        </div>
        <p v-if="!rulePattern.trim() || !ruleCategoryId" class="muted-text">利用先パターンと分類先カテゴリを入力すると追加できます。</p>
        <p v-if="newRuleIsDuplicate" class="warning-line">同じ分類ルールが既に存在します。</p>
        <div v-if="rules.length === 0" class="empty">分類ルールがありません。</div>
        <div v-else class="table-wrap compact-table">
          <table>
            <thead>
              <tr>
                <th>条件</th>
                <th>利用先パターン</th>
                <th>分類先カテゴリ</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="rule in rules" :key="rule.id">
                <td><span class="badge">{{ matchTypeLabel(rule.matchType) }}</span></td>
                <td>{{ rule.pattern }}</td>
                <td>{{ categoryNameById.get(rule.categoryId) ?? `#${rule.categoryId}` }}</td>
              </tr>
            </tbody>
          </table>
        </div>
        <div class="divider" />
        <label class="check-row">
          <input v-model="overwriteManualCategory" type="checkbox" />
          手動カテゴリも上書きする
        </label>
        <button type="button" :disabled="saving || rules.length === 0" @click="reapplyRules">既存明細へ適用</button>
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
                <button
                  type="button"
                  :disabled="saving || !candidateCategoryIds[candidate.merchantName] || candidateRuleExists(candidate)"
                  @click="createRuleFromCandidate(candidate)"
                >
                  ルール化
                </button>
                <p v-if="candidateRuleExists(candidate)" class="muted-text">作成済み</p>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>
  </section>
</template>
