<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { deleteImport, listCreditCards, listImports, updateImportCreditCard } from '../api/client'
import type { CreditCard, ImportFile } from '../api/types'

const imports = ref<ImportFile[]>([])
const creditCards = ref<CreditCard[]>([])
const loading = ref(false)
const deletingId = ref<number | null>(null)
const savingId = ref<number | null>(null)
const confirmTarget = ref<ImportFile | null>(null)
const editingTarget = ref<ImportFile | null>(null)
const creditCardName = ref('')
const error = ref('')
const message = ref('')

async function load() {
  loading.value = true
  error.value = ''
  try {
    const [result, cardResult] = await Promise.all([listImports(), listCreditCards()])
    imports.value = result.items
    creditCards.value = cardResult.items
  } catch {
    error.value = 'インポート履歴を取得できませんでした'
  } finally {
    loading.value = false
  }
}

function requestRemoveImport(item: ImportFile) {
  error.value = ''
  message.value = ''
  confirmTarget.value = item
}

function requestEditCreditCard(item: ImportFile) {
  error.value = ''
  message.value = ''
  editingTarget.value = item
  creditCardName.value = creditCards.value.find((card) => card.id === item.creditCardId)?.displayName ?? ''
}

function cancelEditCreditCard() {
  if (savingId.value !== null) return
  editingTarget.value = null
  creditCardName.value = ''
}

async function saveCreditCard() {
  if (!editingTarget.value) return
  savingId.value = editingTarget.value.id
  error.value = ''
  message.value = ''
  try {
    await updateImportCreditCard(editingTarget.value.id, creditCardName.value)
    await load()
    message.value = 'クレジットカードを更新しました'
    cancelEditCreditCard()
  } catch {
    error.value = 'クレジットカードを更新できませんでした'
  } finally {
    savingId.value = null
  }
}

function cardName(item: ImportFile) {
  return creditCards.value.find((card) => card.id === item.creditCardId)?.displayName ?? '未設定'
}

function cancelRemoveImport() {
  if (deletingId.value !== null) return
  confirmTarget.value = null
}

async function confirmRemoveImport() {
  if (!confirmTarget.value) return
  const item = confirmTarget.value
  deletingId.value = item.id
  error.value = ''
  message.value = ''
  try {
    await deleteImport(item.id)
    await load()
    message.value = 'インポートを削除しました'
    confirmTarget.value = null
  } catch {
    error.value = 'インポートを削除できませんでした'
  } finally {
    deletingId.value = null
  }
}

onMounted(load)
</script>

<template>
  <section class="screen-stack">
    <p v-if="error" class="error-line">{{ error }}</p>
    <p v-if="message" class="success-line">{{ message }}</p>
    <div class="panel toolbar">
      <button type="button" :disabled="loading" @click="load">再読み込み</button>
      <a class="button-like" href="http://localhost:8080/exports/transactions?format=csv">明細CSV</a>
    </div>
    <div class="panel table-wrap">
      <table>
        <thead>
          <tr>
            <th>ファイル名</th>
            <th>クレジットカード</th>
            <th>形式</th>
            <th>行数</th>
            <th>インポート日時</th>
            <th>操作</th>
          </tr>
        </thead>
        <tbody>
          <tr v-if="imports.length === 0">
            <td colspan="6" class="empty">インポート履歴がありません。</td>
          </tr>
          <tr v-for="item in imports" :key="item.id">
            <td>{{ item.fileName }}</td>
            <td>{{ cardName(item) }}</td>
            <td>{{ item.detectedFormat }}</td>
            <td>{{ item.rowCount }}</td>
            <td>{{ new Date(item.importedAt).toLocaleString() }}</td>
            <td>
              <button type="button" class="secondary-button" :disabled="loading || deletingId !== null || savingId !== null" @click="requestEditCreditCard(item)">
                カード設定
              </button>
              <button type="button" class="danger-button" :disabled="loading || deletingId !== null" @click="requestRemoveImport(item)">
                {{ deletingId === item.id ? '削除中' : '削除' }}
              </button>
            </td>
          </tr>
        </tbody>
      </table>
    </div>
    <div v-if="editingTarget" class="dialog-backdrop" role="presentation" @click.self="cancelEditCreditCard">
      <section class="dialog" role="dialog" aria-modal="true" aria-labelledby="edit-import-card-title">
        <h2 id="edit-import-card-title">クレジットカードを設定</h2>
        <dl class="detail-list">
          <div>
            <dt>ファイル名</dt>
            <dd>{{ editingTarget.fileName }}</dd>
          </div>
        </dl>
        <label class="field-block">
          クレジットカード
          <input v-model="creditCardName" type="text" list="credit-card-options" placeholder="例: Olive ゴールド / 個人用" />
        </label>
        <datalist id="credit-card-options">
          <option v-for="card in creditCards" :key="card.id" :value="card.displayName" />
        </datalist>
        <div class="toolbar right">
          <button type="button" class="secondary-button" :disabled="savingId !== null" @click="cancelEditCreditCard">キャンセル</button>
          <button type="button" :disabled="savingId !== null" @click="saveCreditCard">
            {{ savingId === editingTarget.id ? '保存中' : '保存' }}
          </button>
        </div>
      </section>
    </div>
    <div v-if="confirmTarget" class="dialog-backdrop" role="presentation" @click.self="cancelRemoveImport">
      <section class="dialog" role="dialog" aria-modal="true" aria-labelledby="delete-import-title">
        <h2 id="delete-import-title">インポートを削除</h2>
        <dl class="detail-list">
          <div>
            <dt>ファイル名</dt>
            <dd>{{ confirmTarget.fileName }}</dd>
          </div>
          <div>
            <dt>インポート日時</dt>
            <dd>{{ new Date(confirmTarget.importedAt).toLocaleString() }}</dd>
          </div>
          <div>
            <dt>影響</dt>
            <dd>対象ファイル由来の明細とインポート情報を削除します。</dd>
          </div>
        </dl>
        <div class="toolbar right">
          <button type="button" class="secondary-button" :disabled="deletingId !== null" @click="cancelRemoveImport">キャンセル</button>
          <button type="button" class="danger-button" :disabled="deletingId !== null" @click="confirmRemoveImport">
            {{ deletingId === confirmTarget.id ? '削除中' : 'インポートを削除' }}
          </button>
        </div>
      </section>
    </div>
  </section>
</template>
