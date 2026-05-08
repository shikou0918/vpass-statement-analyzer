<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { deleteImport, listImports } from '../api/client'
import type { ImportFile } from '../api/types'

const imports = ref<ImportFile[]>([])
const loading = ref(false)
const deletingId = ref<number | null>(null)
const confirmTarget = ref<ImportFile | null>(null)
const error = ref('')
const message = ref('')

async function load() {
  loading.value = true
  error.value = ''
  try {
    const result = await listImports()
    imports.value = result.items
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
      <a class="button-like" href="http://localhost:8080/exports/categories">カテゴリJSON</a>
      <a class="button-like" href="http://localhost:8080/exports/category-rules">分類ルールJSON</a>
    </div>
    <div class="panel table-wrap">
      <table>
        <thead>
          <tr>
            <th>ファイル名</th>
            <th>形式</th>
            <th>行数</th>
            <th>インポート日時</th>
            <th>操作</th>
          </tr>
        </thead>
        <tbody>
          <tr v-if="imports.length === 0">
            <td colspan="5" class="empty">インポート履歴がありません。</td>
          </tr>
          <tr v-for="item in imports" :key="item.id">
            <td>{{ item.fileName }}</td>
            <td>{{ item.detectedFormat }}</td>
            <td>{{ item.rowCount }}</td>
            <td>{{ new Date(item.importedAt).toLocaleString() }}</td>
            <td>
              <button type="button" class="danger-button" :disabled="loading || deletingId !== null" @click="requestRemoveImport(item)">
                {{ deletingId === item.id ? '削除中' : '削除' }}
              </button>
            </td>
          </tr>
        </tbody>
      </table>
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
