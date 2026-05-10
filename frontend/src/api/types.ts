export type Pagination = {
  page: number
  pageSize: number
  totalItems: number
  totalPages: number
}

export type ApiError = {
  code: string
  message: string
  details?: Record<string, unknown> | null
}

export type ImportMappingCandidate = {
  sourceColumnName?: string
  sourceColumnIndex: number
  targetField: string
  sampleValues: string[]
  required: boolean
}

export type ImportPreviewRow = {
  rowNumber: number
  normalized: Record<string, unknown>
  rawColumns: string[]
}

export type ImportRowError = {
  rowNumber: number
  errorType: string
  message: string
  rawColumns?: string[]
}

export type ImportPreview = {
  previewId: string
  fileName: string
  fileHash: string
  detectedFormat: string
  encoding: string
  hasHeader: boolean
  mappingCandidates: ImportMappingCandidate[]
  previewRows: ImportPreviewRow[]
  errors: ImportRowError[]
  duplicateFile: boolean
}

export type ImportFile = {
  id: number
  fileName: string
  fileHash: string
  detectedFormat: string
  hasHeader: boolean
  rowCount: number
  importedAt: string
}

export type Transaction = {
  id: number
  sourceFileId: number
  usageDate: string
  merchantName: string
  cardUser: string
  paymentMethod: string
  billingMonth: string
  usageAmount?: number | null
  billedAmount?: number | null
  categoryId?: number | null
  memo: string
  excludedFromAnalytics: boolean
}

export type Category = {
  id: number
  name: string
  color: string
}

export type CategoryRule = {
  id: number
  matchType: 'contains' | 'startsWith' | 'equals' | 'regex'
  pattern: string
  categoryId: number
  priority: number
}

export type ClassificationCandidate = {
  merchantName: string
  transactionCount: number
}

export type CategoryRuleApplicationPreviewItem = {
  transactionId: number
  usageDate: string
  merchantName: string
  currentCategoryId?: number | null
  newCategoryId: number
}

export type CategoryRuleApplicationPreview = {
  matchedCount: number
  changedCount: number
  items: CategoryRuleApplicationPreviewItem[]
}

export type ChartPoint = {
  label: string
  amount: number
}

export type MonthlySummary = {
  month: string
  totalAmount: number
  previousAmount: number
  diffAmount: number
  transactionCount: number
  dailyTrend: ChartPoint[]
}

export type RankingItem = {
  merchantName: string
  totalAmount: number
  transactionCount: number
}

export type CategorySummaryItem = {
  categoryId?: number | null
  categoryName: string
  color: string
  totalAmount: number
  transactionCount: number
  ratio: number
}

export type ListResponse<T> = {
  items: T[]
  pagination?: Pagination
}
