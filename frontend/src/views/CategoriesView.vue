<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import {
  applyCategoryRules,
  createCategory,
  createCategoryRule,
  deleteCategory,
  deleteCategoryRule,
  ApiClientError,
  listCategories,
  listCategoryRules,
  listClassificationCandidates,
  previewCategoryRuleApplication,
  updateCategoryRule,
} from '../api/client'
import type { Category, CategoryRule, CategoryRuleApplicationPreview, ClassificationCandidate } from '../api/types'

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
const editingRuleId = ref<number | null>(null)
const editRuleMatchType = ref<CategoryRule['matchType']>('contains')
const editRulePattern = ref('')
const editRuleCategoryId = ref('')
const candidateCategoryIds = ref<Record<string, string>>({})
const ruleApplicationPreview = ref<CategoryRuleApplicationPreview | null>(null)
const previewResolver = ref<((ok: boolean) => void) | null>(null)

const categoryNameById = computed(() => new Map(categories.value.map((category) => [category.id, category.name])))
const newRuleIsDuplicate = computed(
  () => Boolean(rulePattern.value.trim() && ruleCategoryId.value) && ruleExists(ruleMatchType.value, rulePattern.value, ruleCategoryId.value),
)
const editRuleIsDuplicate = computed(
  () =>
    editingRuleId.value !== null &&
    Boolean(editRulePattern.value.trim() && editRuleCategoryId.value) &&
    ruleExists(editRuleMatchType.value, editRulePattern.value, editRuleCategoryId.value, editingRuleId.value),
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

function ruleExists(matchType: CategoryRule['matchType'], pattern: string, categoryId: string | number, excludeId?: number) {
  const key = ruleKey(matchType, pattern, categoryId)
  return rules.value.some((rule) => rule.id !== excludeId && ruleKey(rule.matchType, rule.pattern, rule.categoryId) === key)
}

function candidateRuleExists(candidate: ClassificationCandidate) {
  const categoryId = candidateCategoryIds.value[candidate.merchantName]
  return Boolean(categoryId) && ruleExists('contains', candidate.merchantName, categoryId)
}

function errorMessage(err: unknown, fallback: string) {
  return err instanceof ApiClientError ? err.apiError.message : fallback
}

type CategoryRulePayload = Omit<CategoryRule, 'id'>

async function confirmRuleApplication(rule: CategoryRulePayload) {
  const preview = await previewCategoryRuleApplication(rule, true)
  if (preview.changedCount === 0) return true
  return showRuleApplicationPreview(preview)
}

function showRuleApplicationPreview(preview: CategoryRuleApplicationPreview) {
  ruleApplicationPreview.value = preview
  return new Promise<boolean>((resolve) => {
    previewResolver.value = resolve
  })
}

function closeRuleApplicationPreview(ok: boolean) {
  previewResolver.value?.(ok)
  previewResolver.value = null
  ruleApplicationPreview.value = null
}

function categoryLabel(categoryId?: number | null) {
  if (!categoryId) return '未分類'
  return categoryNameById.value.get(categoryId) ?? `#${categoryId}`
}

async function applyRuleToExistingTransactions(rule: CategoryRulePayload) {
  return applyCategoryRules(true, rule)
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
    const rule = {
      matchType: ruleMatchType.value,
      pattern: rulePattern.value.trim(),
      categoryId: Number(ruleCategoryId.value),
      priority: rules.value.length + 1,
    }
    const ok = await confirmRuleApplication(rule)
    if (!ok) return
    await createCategoryRule(rule)
    const result = await applyRuleToExistingTransactions(rule)
    rulePattern.value = ''
    await load()
    message.value = `分類ルールを作成しました。既存明細を ${result.updatedCount} 件更新しました`
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
    const rule = {
      matchType: 'contains',
      pattern: candidate.merchantName,
      categoryId: Number(categoryId),
      priority: rules.value.length + 1,
    } satisfies CategoryRulePayload
    const ok = await confirmRuleApplication(rule)
    if (!ok) return
    await createCategoryRule(rule)
    const result = await applyRuleToExistingTransactions(rule)
    delete candidateCategoryIds.value[candidate.merchantName]
    await load()
    message.value = `分類ルールを作成しました。既存明細を ${result.updatedCount} 件更新しました`
  } catch (err) {
    error.value = errorMessage(err, '分類ルールを作成できませんでした')
  } finally {
    saving.value = false
  }
}

function startEditRule(rule: CategoryRule) {
  editingRuleId.value = rule.id
  editRuleMatchType.value = rule.matchType
  editRulePattern.value = rule.pattern
  editRuleCategoryId.value = String(rule.categoryId)
  error.value = ''
  message.value = ''
}

function cancelEditRule() {
  editingRuleId.value = null
  editRulePattern.value = ''
  editRuleCategoryId.value = ''
  editRuleMatchType.value = 'contains'
}

async function saveRule(rule: CategoryRule) {
  message.value = ''
  if (!editRulePattern.value.trim()) {
    error.value = '利用先パターンを入力してください'
    return
  }
  if (!editRuleCategoryId.value) {
    error.value = '分類先カテゴリを選択してください'
    return
  }
  if (editRuleIsDuplicate.value) {
    error.value = '同じ分類ルールが既に存在します'
    return
  }
  saving.value = true
  error.value = ''
  try {
    const nextRule = {
      matchType: editRuleMatchType.value,
      pattern: editRulePattern.value.trim(),
      categoryId: Number(editRuleCategoryId.value),
      priority: rule.priority,
    }
    const ok = await confirmRuleApplication(nextRule)
    if (!ok) return
    await updateCategoryRule(rule.id, nextRule)
    const result = await applyRuleToExistingTransactions(nextRule)
    cancelEditRule()
    await load()
    message.value = `分類ルールを更新しました。既存明細を ${result.updatedCount} 件更新しました`
  } catch (err) {
    error.value = errorMessage(err, '分類ルールを更新できませんでした')
  } finally {
    saving.value = false
  }
}

async function removeRule(rule: CategoryRule) {
  if (!window.confirm(`${rule.pattern} の分類ルールを削除します。既存明細のカテゴリは変更されません。`)) return
  saving.value = true
  error.value = ''
  message.value = ''
  try {
    await deleteCategoryRule(rule.id)
    if (editingRuleId.value === rule.id) cancelEditRule()
    await load()
    message.value = '分類ルールを削除しました'
  } catch (err) {
    error.value = errorMessage(err, '分類ルールを削除できませんでした')
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
                <th>操作</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="rule in rules" :key="rule.id">
                <template v-if="editingRuleId === rule.id">
                  <td>
                    <select v-model="editRuleMatchType" :disabled="saving">
                      <option value="contains">含む</option>
                      <option value="startsWith">前方一致</option>
                      <option value="equals">完全一致</option>
                      <option value="regex">正規表現</option>
                    </select>
                  </td>
                  <td><input v-model="editRulePattern" type="text" :disabled="saving" /></td>
                  <td>
                    <select v-model="editRuleCategoryId" :disabled="saving">
                      <option value="">カテゴリ</option>
                      <option v-for="category in categories" :key="category.id" :value="category.id">{{ category.name }}</option>
                    </select>
                  </td>
                  <td>
                    <div class="table-actions">
                      <button type="button" :disabled="saving || editRuleIsDuplicate" @click="saveRule(rule)">保存</button>
                      <button type="button" class="secondary-button" :disabled="saving" @click="cancelEditRule">キャンセル</button>
                    </div>
                    <p v-if="editRuleIsDuplicate" class="muted-text">同じ分類ルールが既に存在します。</p>
                  </td>
                </template>
                <template v-else>
                  <td><span class="badge">{{ matchTypeLabel(rule.matchType) }}</span></td>
                  <td>{{ rule.pattern }}</td>
                  <td>{{ categoryNameById.get(rule.categoryId) ?? `#${rule.categoryId}` }}</td>
                  <td>
                    <div class="table-actions">
                      <button type="button" class="secondary-button" :disabled="saving" @click="startEditRule(rule)">編集</button>
                      <button type="button" class="danger-button" :disabled="saving" @click="removeRule(rule)">削除</button>
                    </div>
                  </td>
                </template>
              </tr>
            </tbody>
          </table>
        </div>
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

    <div v-if="ruleApplicationPreview" class="modal-backdrop" role="presentation">
      <div class="modal-panel" role="dialog" aria-modal="true" aria-labelledby="rule-application-title">
        <header class="modal-header">
          <h2 id="rule-application-title">既存明細を更新しますか？</h2>
        </header>
        <p class="muted-text">
          この分類ルールに一致する既存明細 {{ ruleApplicationPreview.changedCount }} 件のカテゴリが更新されます。
        </p>
        <div class="table-wrap compact-table modal-table">
          <table>
            <thead>
              <tr>
                <th>利用日</th>
                <th>利用先</th>
                <th>現在のカテゴリ</th>
                <th>更新後</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="item in ruleApplicationPreview.items" :key="item.transactionId">
                <td>{{ item.usageDate }}</td>
                <td>{{ item.merchantName }}</td>
                <td>{{ categoryLabel(item.currentCategoryId) }}</td>
                <td>{{ categoryLabel(item.newCategoryId) }}</td>
              </tr>
            </tbody>
          </table>
        </div>
        <footer class="modal-actions">
          <button type="button" class="secondary-button" @click="closeRuleApplicationPreview(false)">キャンセル</button>
          <button type="button" @click="closeRuleApplicationPreview(true)">はい、更新する</button>
        </footer>
      </div>
    </div>
  </section>
</template>
