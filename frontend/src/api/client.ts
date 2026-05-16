import type {
  ApiError,
  Category,
  CategoryRule,
  CategoryRuleApplicationPreview,
  CategorySummaryItem,
  ChartPoint,
  ClassificationCandidate,
  CreditCard,
  ImportFile,
  ImportPreview,
  ListResponse,
  MonthlySummary,
  RankingItem,
  Transaction,
} from './types'

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL ?? 'http://localhost:8080'

export class ApiClientError extends Error {
  readonly apiError: ApiError

  constructor(apiError: ApiError) {
    super(apiError.message)
    this.apiError = apiError
  }
}

async function request<T>(path: string, options: RequestInit = {}): Promise<T> {
  const response = await fetch(`${API_BASE_URL}${path}`, {
    ...options,
    headers: options.body instanceof FormData ? options.headers : { 'Content-Type': 'application/json', ...options.headers },
  })

  if (!response.ok) {
    const error = (await safeJson(response)) as ApiError | null
    throw new ApiClientError(error ?? { code: 'HTTP_ERROR', message: `HTTP ${response.status}` })
  }

  if (response.status === 204) {
    return undefined as T
  }

  return (await response.json()) as T
}

async function safeJson(response: Response): Promise<unknown | null> {
  try {
    return await response.json()
  } catch {
    return null
  }
}

export async function createImportPreview(file: File): Promise<ImportPreview> {
  const body = new FormData()
  body.append('file', file)
  body.append('sourceType', 'vpass')
  return request<ImportPreview>('/import-previews', { method: 'POST', body })
}

export async function createImport(preview: ImportPreview, confirmedMapping: Record<string, string>, creditCardName: string) {
  return request('/imports', {
    method: 'POST',
    body: JSON.stringify({
      previewId: preview.previewId,
      fileHash: preview.fileHash,
      creditCardName,
      confirmedMapping,
      options: { applyCategoryRules: true },
    }),
  })
}

export function listImports(): Promise<ListResponse<ImportFile>> {
  return request<ListResponse<ImportFile>>('/imports?page=1&pageSize=20')
}

export function deleteImport(id: number): Promise<void> {
  return request<void>(`/imports/${id}`, { method: 'DELETE' })
}

export function updateImportCreditCard(id: number, creditCardName: string): Promise<ImportFile> {
  return request<ImportFile>(`/imports/${id}`, { method: 'PATCH', body: JSON.stringify({ creditCardName }) })
}

export function listTransactions(params: URLSearchParams): Promise<ListResponse<Transaction>> {
  return request<ListResponse<Transaction>>(`/transactions?${params.toString()}`)
}

export function listCreditCards(): Promise<{ items: CreditCard[] }> {
  return request<{ items: CreditCard[] }>('/credit-cards')
}

export function updateTransaction(id: number, body: { categoryId?: string | null; memo?: string; excludedFromAnalytics?: boolean }): Promise<Transaction> {
  return request<Transaction>(`/transactions/${id}`, { method: 'PATCH', body: JSON.stringify(body) })
}

function appendCreditCard(path: string, creditCardId?: string) {
  if (!creditCardId) return path
  return `${path}&creditCardId=${encodeURIComponent(creditCardId)}`
}

export function getMonthlySummary(month: string, creditCardId?: string): Promise<MonthlySummary> {
  return request<MonthlySummary>(
    appendCreditCard(`/summaries/monthly?month=${encodeURIComponent(month)}&basisDate=billingMonth&basisAmount=billedAmount`, creditCardId),
  )
}

export function getMerchantSummary(month: string, creditCardId?: string): Promise<{ items: RankingItem[] }> {
  return request<{ items: RankingItem[] }>(appendCreditCard(`/summaries/merchants?month=${encodeURIComponent(month)}&basisAmount=billedAmount`, creditCardId))
}

export function getCategorySummary(month: string, creditCardId?: string): Promise<{ items: CategorySummaryItem[] }> {
  return request<{ items: CategorySummaryItem[] }>(appendCreditCard(`/summaries/categories?month=${encodeURIComponent(month)}&basisAmount=billedAmount`, creditCardId))
}

export function getMonthlyTrends(creditCardId?: string): Promise<{ items: ChartPoint[] }> {
  return request<{ items: ChartPoint[] }>(appendCreditCard('/analytics/monthly-trends?basisAmount=billedAmount', creditCardId))
}

export function listCategories(): Promise<{ items: Category[] }> {
  return request<{ items: Category[] }>('/categories')
}

export function createCategory(body: { name: string; color: string }): Promise<Category> {
  return request<Category>('/categories', { method: 'POST', body: JSON.stringify(body) })
}

export function deleteCategory(id: number): Promise<void> {
  return request<void>(`/categories/${id}`, { method: 'DELETE' })
}

export function listCategoryRules(): Promise<{ items: CategoryRule[] }> {
  return request<{ items: CategoryRule[] }>('/category-rules')
}

export function createCategoryRule(body: Omit<CategoryRule, 'id'>): Promise<CategoryRule> {
  return request<CategoryRule>('/category-rules', { method: 'POST', body: JSON.stringify(body) })
}

export function updateCategoryRule(id: number, body: Omit<CategoryRule, 'id'>): Promise<CategoryRule> {
  return request<CategoryRule>(`/category-rules/${id}`, { method: 'PATCH', body: JSON.stringify(body) })
}

export function deleteCategoryRule(id: number): Promise<void> {
  return request<void>(`/category-rules/${id}`, { method: 'DELETE' })
}

export function previewCategoryRuleApplication(body: Omit<CategoryRule, 'id'>, overwriteManualCategory: boolean) {
  return request<CategoryRuleApplicationPreview>('/category-rule-application-previews', {
    method: 'POST',
    body: JSON.stringify({ ...body, overwriteManualCategory }),
  })
}

export function listClassificationCandidates(): Promise<{ items: ClassificationCandidate[] }> {
  return request<{ items: ClassificationCandidate[] }>('/classification-candidates?limit=50')
}

export function applyCategoryRules(overwriteManualCategory: boolean, rule?: Omit<CategoryRule, 'id'>) {
  return request<{ matchedCount: number; updatedCount: number; unchangedCount: number; uncategorizedCount: number }>('/category-rule-applications', {
    method: 'POST',
    body: JSON.stringify({ ...(rule ?? {}), scope: 'all', overwriteManualCategory }),
  })
}
