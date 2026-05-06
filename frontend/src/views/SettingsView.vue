<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { getSettings, updateSettings } from '../api/client'
import type { Settings } from '../api/types'

const settings = ref<Settings>({ defaultBasisDate: 'billingMonth', defaultBasisAmount: 'billedAmount' })
const loading = ref(false)
const error = ref('')
const message = ref('')

async function load() {
  loading.value = true
  try {
    settings.value = await getSettings()
  } catch {
    error.value = '設定を取得できませんでした'
  } finally {
    loading.value = false
  }
}

async function save() {
  loading.value = true
  error.value = ''
  try {
    settings.value = await updateSettings(settings.value)
    message.value = '設定を保存しました'
  } catch {
    error.value = '設定を保存できませんでした'
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
    <div class="panel form-panel">
      <label>
        集計日付基準
        <select v-model="settings.defaultBasisDate">
          <option value="billingMonth">請求月</option>
          <option value="usageDate">利用日</option>
        </select>
      </label>
      <label>
        集計金額基準
        <select v-model="settings.defaultBasisAmount">
          <option value="billedAmount">請求金額</option>
          <option value="usageAmount">利用金額</option>
        </select>
      </label>
      <button type="button" :disabled="loading" @click="save">保存</button>
    </div>
  </section>
</template>

