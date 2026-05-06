import { ref } from 'vue'
import { ApiClientError } from '../api/client'

export function useAsyncState<T>() {
  const data = ref<T | null>(null)
  const loading = ref(false)
  const error = ref('')

  async function run(task: () => Promise<T>) {
    loading.value = true
    error.value = ''
    try {
      data.value = await task()
      return data.value
    } catch (e) {
      error.value = e instanceof ApiClientError ? e.apiError.message : '通信に失敗しました'
      return null
    } finally {
      loading.value = false
    }
  }

  return { data, loading, error, run }
}

