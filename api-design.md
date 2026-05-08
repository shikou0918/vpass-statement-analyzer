# Vpass明細分析アプリ API設計書

## 1. API設計方針

### 1.1 目的

`overview-design.md` をもとに、Vue 3 + TypeScript フロントエンドと Go API バックエンドの間で利用する REST API 契約を定義する。

初期版は本人のみが利用するローカルWebアプリのため、認証・複数ユーザー・クラウド同期は API スコープ外とする。OpenAPI 3.1 仕様は [openapi.yaml](./openapi.yaml) に定義する。

### 1.2 前提

- Base URL はローカル起動の `http://localhost:8080` とする
- 初期版では path versioning の `/api/v1` は付けない
- 認証は定義しない
- リクエスト/レスポンスは原則 JSON とする
- CSVアップロードのみ `multipart/form-data` を使用する
- Vpass CSV の列数は固定しない
- CSVにヘッダーがある場合はヘッダー名を優先し、ない場合は既知フォーマット候補からマッピング候補を作る
- CSVプレビューは保存処理ではない

### 1.3 バージョニング

初期版はローカル利用で外部クライアント互換性を維持する必要が低いため、URL path に `/api/v1` は付けない。

将来的に外部公開や長期互換性が必要になった場合は、以下のいずれかを再検討する。

- `/api/v1` のような path versioning
- `Accept` header による media type versioning
- OpenAPI `info.version` とアプリ配布バージョンの対応管理

## 2. 共通規約

### 2.1 ID

各リソース ID は API 上では `string` として扱う。実装では SQLite の整数IDまたはUUIDのどちらでもよいが、フロントエンドに DB 実装詳細を漏らさないため文字列に正規化する。

### 2.2 日付・月

| 用途 | 形式 | 例 |
|---|---|---|
| 日付 | `YYYY-MM-DD` | `2026-05-04` |
| 請求月・集計月 | `YYYY-MM` | `2026-05` |
| 日時 | RFC3339 | `2026-05-04T10:30:00+09:00` |

### 2.3 ページング

一覧 API は必要に応じて以下のクエリを受け取る。

| query | 型 | 既定値 | 説明 |
|---|---:|---:|---|
| `page` | integer | `1` | 1始まりのページ番号 |
| `pageSize` | integer | `50` | 1ページ件数 |
| `sort` | string | APIごとに定義 | ソート対象 |
| `order` | string | `desc` | `asc` または `desc` |

レスポンスは `items` と `pagination` を持つ envelope にする。

### 2.4 エラー形式

全APIのエラーは共通形式にする。

```json
{
  "code": "VALIDATION_ERROR",
  "message": "入力内容を確認してください",
  "details": {
    "field": "billingMonth"
  }
}
```

代表的な HTTP status:

| status | 用途 |
|---:|---|
| 400 | クエリやリクエストボディの形式不正 |
| 404 | 対象リソースが存在しない |
| 409 | 同一ファイル、重複明細などの競合 |
| 422 | CSV行、マッピング、日付/金額変換など業務検証エラー |
| 500 | 想定外のサーバーエラー |

## 3. エンドポイント一覧

### 3.1 CSVインポート

| Method | Path | operationId | 概要 |
|---|---|---|---|
| POST | `/import-previews` | `createImportPreview` | CSVファイルを受け取り、保存前プレビューとマッピング候補を返す |
| POST | `/imports` | `createImport` | 確定マッピングをもとに CSV を保存する |
| GET | `/imports` | `listImports` | インポート履歴を取得する |
| GET | `/imports/{importFileId}` | `getImport` | インポート結果詳細を取得する |
| DELETE | `/imports/{importFileId}` | `deleteImport` | 指定ファイル由来の明細・マッピング・エラー・履歴を削除する |

`/import-previews` は DB 保存をしない。初期設計では `previewId` を返すが、これは短時間の一時参照として扱う。永続化するか、ファイルハッシュから再計算するか、フロントエンドが保存時に再度ファイルを送るかは実装時の未決事項とする。

### 3.2 明細

| Method | Path | operationId | 概要 |
|---|---|---|---|
| GET | `/transactions` | `listTransactions` | 明細を検索・フィルタ・ページングして取得する |
| GET | `/transactions/{transactionId}` | `getTransaction` | 明細1件を取得する |
| PATCH | `/transactions/{transactionId}` | `updateTransaction` | カテゴリ、メモ、除外フラグなど編集可能項目を更新する |

### 3.3 集計・分析

| Method | Path | operationId | 概要 |
|---|---|---|---|
| GET | `/summaries/monthly` | `getMonthlySummary` | 対象月の合計、前月比、日別推移を取得する |
| GET | `/summaries/merchants` | `getMerchantSummary` | 対象期間の利用先別ランキングを取得する |
| GET | `/summaries/categories` | `getCategorySummary` | 対象期間のカテゴリ別内訳を取得する |
| GET | `/analytics/monthly-trends` | `getMonthlyTrends` | 月別支出推移を取得する |
| GET | `/analytics/merchant-trends` | `getMerchantTrends` | 利用先別の月次推移を取得する |
| GET | `/analytics/category-trends` | `getCategoryTrends` | カテゴリ別の月次推移を取得する |
| GET | `/analytics/recurring-candidates` | `listRecurringCandidates` | 固定費候補を取得する |
| GET | `/analytics/small-frequent-transactions` | `listSmallFrequentTransactions` | 少額高頻度支出候補を取得する |

集計条件では `basisDate` と `basisAmount` を分ける。

| query | 値 | 説明 |
|---|---|---|
| `basisDate` | `billingMonth` / `usageDate` | 集計対象期間の基準 |
| `basisAmount` | `billedAmount` / `usageAmount` | 集計金額の基準 |

### 3.4 カテゴリ・分類ルール

| Method | Path | operationId | 概要 |
|---|---|---|---|
| GET | `/categories` | `listCategories` | カテゴリ一覧を取得する |
| POST | `/categories` | `createCategory` | カテゴリを作成する |
| PATCH | `/categories/{categoryId}` | `updateCategory` | カテゴリ名・色を更新する |
| DELETE | `/categories/{categoryId}` | `deleteCategory` | カテゴリを削除し、紐づく明細を未分類へ戻す |
| GET | `/category-rules` | `listCategoryRules` | 分類ルール一覧を取得する |
| POST | `/category-rules` | `createCategoryRule` | 分類ルールを作成する |
| PATCH | `/category-rules/{categoryRuleId}` | `updateCategoryRule` | 分類ルールを更新する |
| DELETE | `/category-rules/{categoryRuleId}` | `deleteCategoryRule` | 分類ルールを削除する |
| GET | `/classification-candidates` | `listClassificationCandidates` | 未分類の利用先を件数順に集約して取得する |
| POST | `/category-rule-applications` | `createCategoryRuleApplication` | 既存明細へ分類ルールを再適用する |

`/category-rule-applications` は処理リソースとして扱う。初期版では同期処理で結果件数を即時返す。将来、対象件数が増えた場合は `applicationId` を永続化して進捗取得 API を追加する。

### 3.5 エクスポート・設定

| Method | Path | operationId | 概要 |
|---|---|---|---|
| GET | `/exports/transactions` | `exportTransactions` | 明細を CSV または JSON でエクスポートする |
| GET | `/settings` | `getSettings` | アプリ設定を取得する |
| PATCH | `/settings` | `updateSettings` | アプリ設定を更新する |

## 4. 主要リクエスト/レスポンス

### 4.1 `POST /import-previews`

Request:

- content type: `multipart/form-data`
- `file`: CSVファイル
- `sourceType`: `vpass`

Response:

- `previewId`
- `fileName`
- `fileHash`
- `detectedFormat`
- `encoding`
- `hasHeader`
- `mappingCandidates`
- `previewRows`
- `errors`
- `duplicateFile`

保存は行わない。実明細データをログに出力しない。

### 4.2 `POST /imports`

Request:

- `previewId`
- `fileHash`
- `confirmedMapping`
- `options.applyCategoryRules`

Response:

- 作成された `ImportFile`
- 保存件数
- 明細重複スキップ件数
- エラー件数

保存前にサーバー側で再検証する。`fileHash` が既に存在する場合は `409` を返す。

### 4.3 `DELETE /imports/{importFileId}`

Response:

- 成功時は `204 No Content`
- 対象のインポート履歴が存在しない場合は `404`

削除対象:

- 対象 `ImportFile`
- 対象ファイル由来の `Transaction`
- 対象ファイル由来の `ImportMapping`
- 対象ファイル由来の `ImportError`

削除後は同一 `fileHash` のCSVを再度インポートできる。削除処理はトランザクション内で実行し、一部だけ削除された状態を残さない。

### 4.4 `GET /transactions`

主な query:

- `billingMonth`
- `usageDateFrom`
- `usageDateTo`
- `merchantName`
- `categoryId`
- `keyword`
- `amountMin`
- `amountMax`
- `includeExcluded`
- `page`
- `pageSize`
- `sort`
- `order`

Response は `TransactionListResponse`。

### 4.5 `PATCH /transactions/{transactionId}`

更新可能項目:

- `categoryId`
- `memo`
- `excludedFromAnalytics`

金額、利用日、利用先名など CSV 由来の正規化項目は初期版では更新対象外とする。

### 4.6 `POST /category-rule-applications`

Request:

- `scope`: `all` / `filtered`
- `filters`
- `overwriteManualCategory`

Response:

- `matchedCount`
- `updatedCount`
- `unchangedCount`
- `uncategorizedCount`

手動カテゴリを上書きするかは必ず request で明示する。

## 5. バリデーション

### 5.1 CSVプレビュー

- CSVファイルが空でないこと
- 文字コード変換できること
- CSVとしてパースできること
- 必須項目にマッピングできること
- 日付、請求月、金額が変換できること
- `usageAmount` または `billedAmount` の少なくとも一方があること

### 5.2 インポート保存

- `previewId` または再検証可能な入力が存在すること
- `confirmedMapping` が必須項目を満たすこと
- 同一 `fileHash` が未保存であること
- `dedupeKey` が既存明細と重複する場合はスキップすること

### 5.3 インポート削除

- `importFileId` が存在すること
- 対象ファイル由来の明細、マッピング、エラー、履歴を同一トランザクションで削除すること
- 削除後は対象 `fileHash` を重複判定対象から外すこと

### 5.4 カテゴリ

- `name` は空不可
- `color` は `#RRGGBB` 形式
- 削除時は紐づく明細を未分類へ戻す

### 5.5 分類ルール

- `matchType` は `contains` / `startsWith` / `equals` / `regex`
- `pattern` は空不可
- `regex` の場合は正規表現としてコンパイルできること
- `categoryId` は存在するカテゴリを指すこと

## 6. 未決事項

| 項目 | 選択肢 | APIへの影響 |
|---|---|---|
| import preview の保持方式 | 一時メモリ / 一時ファイル / DB保存 / 再アップロード | `previewId` の有効期限、保存APIの request 形式が変わる |
| 主集計金額 | `billedAmount` / `usageAmount` | dashboard と analysis の既定 query が変わる |
| 主集計日付 | `billingMonth` / `usageDate` | summary API の既定条件が変わる |
| 手動分類の優先度 | ルール再適用で上書きする / しない | `createCategoryRuleApplication` の既定値が変わる |
| Export形式 | CSV中心 / JSON中心 / 両方 | `format` query と content type が変わる |
