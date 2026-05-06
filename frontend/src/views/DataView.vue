<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { deleteImport, listImports } from '../api/client'
import type { ImportFile } from '../api/types'

const imports = ref<ImportFile[]>([])
const loading = ref(false)
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

async function removeImport(item: ImportFile) {
  if (!window.confirm(`${item.fileName} を削除します。対象ファイル由来の明細も削除されます。`)) return
  loading.value = true
  try {
    await deleteImport(item.id)
    await load()
    message.value = 'インポートを削除しました'
  } catch {
    error.value = 'インポートを削除できませんでした'
  } finally {
    loading.value = false
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
            <td><button type="button" :disabled="loading" @click="removeImport(item)">削除</button></td>
          </tr>
        </tbody>
      </table>
    </div>
  </section>
</template>

