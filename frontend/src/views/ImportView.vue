<script setup lang="ts">
import { computed, ref } from 'vue'
import { useRouter } from 'vue-router'
import { createImport, createImportPreview } from '../api/client'
import type { ImportMappingCandidate, ImportPreview } from '../api/types'
import { useAsyncState } from '../composables/useAsyncState'

const router = useRouter()

const fileInput = ref<HTMLInputElement | null>(null)
const file = ref<File | null>(null)
const preview = ref<ImportPreview | null>(null)
const confirmedMapping = ref<Record<string, string>>({})
const saveMessage = ref('')
const previewState = useAsyncState<ImportPreview>()
const importState = useAsyncState<unknown>()

const previewColumnCount = computed(() => {
  return Math.max(0, ...(preview.value?.previewRows.map((row) => row.rawColumns.length) ?? [0]))
})

const missingRequired = computed(() => {
  const targets = new Set(Object.values(confirmedMapping.value).filter(Boolean))
  const missing = ['usageDate', 'merchantName', 'billingMonth'].filter((target) => !targets.has(target))
  if (!targets.has('usageAmount') && !targets.has('billedAmount')) {
    missing.push('usageAmount または billedAmount')
  }
  return missing
})

async function onSelect(event: Event) {
  const input = event.target as HTMLInputElement
  const selectedFile = input.files?.[0] ?? null
  file.value = selectedFile
  preview.value = null
  confirmedMapping.value = {}
  saveMessage.value = ''
  if (selectedFile) {
    await previewFile(selectedFile)
  }
}

function openFileDialog() {
  fileInput.value?.click()
}

async function previewFile(selectedFile: File) {
  const result = await previewState.run(() => createImportPreview(selectedFile))
  if (!result || file.value !== selectedFile) return
  preview.value = result
  confirmedMapping.value = Object.fromEntries(
    result.mappingCandidates.map((candidate: ImportMappingCandidate) => [String(candidate.sourceColumnIndex), candidate.targetField]),
  )
}

async function saveImport() {
  if (!preview.value || missingRequired.value.length > 0 || preview.value.duplicateFile) return
  const result = await importState.run(() => createImport(preview.value as ImportPreview, confirmedMapping.value))
  if (result) {
    saveMessage.value = 'インポートを保存しました'
    await router.push('/transactions')
  }
}
</script>

<template>
  <section class="screen-stack">
    <div class="panel">
      <div class="step-row">
        <span class="step active">1. ファイル選択</span>
        <span class="step" :class="{ active: preview }">2. プレビュー確認</span>
        <span class="step" :class="{ active: saveMessage }">3. 保存結果</span>
      </div>
      <div class="toolbar">
        <button class="file-button" type="button" @click="openFileDialog">
          CSVファイル
        </button>
        <input ref="fileInput" class="visually-hidden-file" type="file" accept=".csv,text/csv" @change="onSelect" />
        <span class="muted">{{ file?.name ?? '未選択' }}</span>
        <span v-if="previewState.loading.value" class="muted">解析中</span>
      </div>
      <p v-if="previewState.error.value" class="error-line">{{ previewState.error.value }}</p>
    </div>

    <div v-if="preview" class="panel">
      <div class="summary-grid">
        <div><span>encoding</span><strong>{{ preview.encoding }}</strong></div>
        <div><span>format</span><strong>{{ preview.detectedFormat }}</strong></div>
        <div><span>header</span><strong>{{ preview.hasHeader ? 'あり' : 'なし' }}</strong></div>
        <div><span>duplicate</span><strong>{{ preview.duplicateFile ? 'あり' : 'なし' }}</strong></div>
      </div>

      <p v-if="preview.duplicateFile" class="warning-line">同一ファイルが保存済みです。保存はできません。</p>
      <p v-if="missingRequired.length > 0" class="warning-line">不足: {{ missingRequired.join(', ') }}</p>

      <div class="csv-preview">
        <div class="csv-preview-header">
          <h2>CSV行プレビュー</h2>
          <span class="muted">先頭{{ preview.previewRows.length }}行</span>
        </div>
        <div class="csv-grid-wrap">
          <table class="csv-grid">
            <thead>
              <tr>
                <th>行</th>
                <th v-for="columnIndex in previewColumnCount" :key="columnIndex">列 {{ columnIndex }}</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="row in preview.previewRows" :key="row.rowNumber">
                <th>{{ row.rowNumber }}</th>
                <td v-for="columnIndex in previewColumnCount" :key="columnIndex">
                  {{ row.rawColumns[columnIndex - 1] ?? '' }}
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>

      <div class="toolbar right">
        <button type="button" :disabled="missingRequired.length > 0 || preview.duplicateFile || importState.loading.value" @click="saveImport">
          {{ importState.loading.value ? '保存中' : 'インポート実行' }}
        </button>
      </div>
      <p v-if="importState.error.value" class="error-line">{{ importState.error.value }}</p>
    </div>
  </section>
</template>
