# Vpass明細分析アプリ 概要設計書

## 1. 設計方針

### 1.1 目的

Vpassからエクスポートしたクレジットカード利用明細CSVを、ローカル環境で安全に取り込み、月別・利用先別・カテゴリ別に支出傾向を分析できるWebアプリとして設計する。

主な目的は以下とする。

- Vpass CSVをCP932/Shift_JIS系CSVとして正しく取り込み、列構成を固定せずにマッピングできる
- 明細データを外部サービスへ送信せず、ローカルDBで管理する
- 月別支出、利用先別ランキング、カテゴリ別内訳を素早く確認できる
- 利用先名に基づく分類ルールを蓄積し、分析の手間を減らす
- 将来的に他カードCSVや銀行CSVを追加できる拡張性を残す

### 1.2 前提

確定要件:

- 初期版は本人のみが利用するローカルWebアプリとする
- PC版ChromeまたはEdgeの最新版を主対象とする
- 認証、複数ユーザー、クラウド同期は初期スコープ外とする
- Vpass CSVはCP932/Shift_JIS系を想定するが、列数は固定しない
- CSVにヘッダーがある場合はヘッダー名を優先し、ヘッダーがない場合は既知フォーマット候補からマッピングする
- データ保存はSQLiteを基本とする
- フロントエンドはVue 3 + TypeScript + Viteを基本とする
- バックエンドはGo API、ORMはGORM、DBはSQLiteを基本とする

設計上の仮定:

- 仮定: 集計の主軸は初期状態では「請求金額」とし、画面上で「利用金額」と切り替え可能にする余地を残す
- 仮定: 月別集計は初期状態では「支払月」ベースとし、「利用日」ベースは分析条件として拡張可能にする
- 仮定: CSVファイルはブラウザからアップロードし、サーバー側で文字コード変換・パース・保存を行う
- 仮定: ローカル実行を前提とするため、外部APIやクラウドストレージは使用しない

### 1.3 スコープ

MVPスコープ:

- Vpass CSVアップロード
- CP932/Shift_JIS系CSVの読み込み
- ヘッダー名または既知フォーマット候補に基づく項目マッピング
- インポート前プレビュー
- 必須項目不足、マッピング不可、金額変換エラーの表示
- 重複インポート防止
- SQLiteへの明細保存
- 明細一覧、検索、フィルタ、並び替え
- 月別支出合計
- 利用先別ランキング
- カテゴリ作成・編集・削除
- 明細へのカテゴリ手動設定
- 利用先名に基づく分類ルール作成
- 未分類の利用先候補の集約とルール化
- 既存明細への分類ルール再適用

### 1.4 初期スコープ外

- 銀行口座や証券口座との自動連携
- Vpassへの自動ログインやスクレイピング
- スマートフォンアプリ化
- 複数ユーザー対応
- クラウド同期
- 外部AI APIによる自動分類
- レシート読み取り
- 予算管理

## 2. 全体アーキテクチャ

### 2.1 システム構成

```mermaid
flowchart LR
  User["利用者"] --> Browser["PCブラウザ"]
  Browser --> App["Vue 3 + TypeScript + Vite"]
  App --> API["Go API"]
  API --> Importer["Vpass CSV Importer"]
  Importer --> Parser["CP932/Shift_JIS + CSV Parser"]
  API --> ORM["GORM"]
  ORM --> DB[("SQLite")]
  API --> Analytics["Aggregation / Analysis Service"]
  App --> Charts["Vue Chart Components"]
```

### 2.2 技術スタック

| 領域 | 採用方針 | 理由 |
|---|---|---|
| フロントエンド | Vue 3 + TypeScript + Vite | 実務スタックに寄せつつ、ローカルWebアプリとして画面追加しやすい |
| スタイリング | Tailwind CSS | ダッシュボードやテーブルUIを素早く構築できる |
| バックエンド | Go | CSV処理、文字コード変換、SQLite保存、集計APIを堅実に実装しやすい |
| CSVパース | Go標準 `encoding/csv` + `golang.org/x/text/encoding/japanese` | CP932/Shift_JIS対応、ヘッダー有無判定、列マッピングをバックエンドに集約するため |
| データ保存 | SQLite | 本人利用のローカルアプリに向いており、導入が軽い |
| ORM | GORM | GoでSQLiteを扱いやすく、モデル定義とCRUDを簡潔に実装できる |
| グラフ | Chart.js系VueラッパーまたはECharts | Vueで月次・カテゴリ別グラフを実装しやすい |

### 2.3 採用理由

- 明細データは金融情報に近いため、外部送信しないローカル保存を優先する
- Vpass CSV固有の文字コード・フォーマット検出・列マッピングはインポーターに閉じ込め、将来のCSV追加に備える
- Go + GORM + SQLiteにより、ローカル実行でも履歴・分類ルール・集計を扱いやすくする
- Vue 3 + TypeScript + Viteにより、実務スタックに近い形で画面開発を進められる
- 分類ルールを明細データと分離し、再適用やルール改善をしやすくする

### 2.4 クライアント / サーバー責務分担

| 領域 | クライアント責務 | サーバー責務 |
|---|---|---|
| CSVアップロード | ファイル選択、マッピング候補確認、プレビュー表示、実行操作 | 文字コード変換、CSVパース、フォーマット検出、項目マッピング、検証、DB保存 |
| 明細一覧 | 検索条件入力、テーブル表示、カテゴリ変更操作 | 検索、フィルタ、並び替え、更新処理 |
| ダッシュボード | 月選択、グラフ表示 | 月別・利用先別・カテゴリ別集計 |
| カテゴリ管理 | カテゴリ/ルール編集UI、未分類候補確認 | ルール保存、未分類利用先の集約、再適用、整合性チェック |
| データ管理 | エクスポート/削除操作 | ファイル単位削除、バックアップ用エクスポート |

## 3. 画面設計

### 3.1 画面一覧

| 画面 | 目的 | 主な要素 |
|---|---|---|
| インポート画面 | Vpass CSVを読み込み、保存前に検証する | アップロード、プレビュー、エラー行一覧、インポート履歴 |
| ダッシュボード画面 | 月ごとの支出状況を把握する | 月選択、支出合計、前月比、カテゴリ別グラフ、利用先ランキング、日別推移 |
| 明細一覧画面 | 明細を検索・確認・分類する | 明細テーブル、年月/利用先/カテゴリ/金額フィルタ、並び替え、カテゴリ変更 |
| カテゴリ・ルール管理画面 | 分類体系と自動分類ルールを管理する | カテゴリ一覧、色設定、ルール一覧、未分類候補、ルール作成、再適用 |
| 分析画面 | 長期傾向や支出パターンを見る | 月別推移、カテゴリ別推移、利用先別推移、固定費候補、少額高頻度支出 |
| データ管理画面 | インポート済みデータを管理する | インポート履歴、ファイル単位削除、データエクスポート |

### 3.2 画面遷移

```mermaid
flowchart TD
  Import["インポート"] --> Dashboard["ダッシュボード"]
  Dashboard --> Transactions["明細一覧"]
  Dashboard --> Analysis["分析"]
  Transactions --> CategoryRules["カテゴリ・ルール管理"]
  CategoryRules --> Transactions
  Import --> DataManagement["データ管理"]
  DataManagement --> Transactions
```

### 3.3 各画面の責務

| 画面 | 責務 |
|---|---|
| インポート | CSVの読み込み、ヘッダー/列構成の検出、項目マッピング結果の確認、エラー検出、重複判定結果の提示 |
| ダッシュボード | 支出総額、前月比、ランキング、内訳、推移を一目で確認できる状態にする |
| 明細一覧 | 明細検索、絞り込み、並び替え、カテゴリ手動補正を行う |
| カテゴリ・ルール管理 | カテゴリと利用先名ルールを独立管理し、未分類候補からルール化して既存の未分類明細へ自動適用できるようにする |
| 分析 | 月次推移、利用先別推移、固定費候補、少額高頻度候補を確認する |
| データ管理 | インポート履歴、ファイル単位削除、エクスポートを扱う |

## 4. データ設計

### 4.1 エンティティ一覧

| エンティティ | 説明 |
|---|---|
| Transaction | Vpass明細1行を正規化して保存する |
| ImportFile | インポートしたCSVファイル単位の履歴を管理する |
| ImportMapping | インポート時に検出・確定した列マッピングを管理する |
| Category | 支出カテゴリを管理する |
| CategoryRule | 利用先名に基づく分類ルールを管理する |
| ImportError | インポート時の不正行・変換エラーを記録または一時保持する |
| AppSetting | 集計基準や表示設定など、アプリ設定を保持する |

### 4.2 主要テーブル

| テーブル | 主な項目 |
|---|---|
| transactions | id, sourceFileId, usageDate, merchantName, cardUser, paymentMethod, billingMonth, usageAmount, billedAmount, categoryId, rawColumns, dedupeKey, createdAt, updatedAt |
| import_files | id, fileName, fileHash, detectedFormat, hasHeader, rowCount, importedAt |
| import_mappings | id, sourceFileId, sourceColumnName, sourceColumnIndex, targetField, createdAt |
| categories | id, name, color, createdAt, updatedAt |
| category_rules | id, matchType, pattern, categoryId, priority, createdAt, updatedAt |
| import_errors | id, sourceFileId, rowNumber, errorType, message, rawColumns, createdAt |
| app_settings | id, key, value, updatedAt |

### 4.3 主なリレーション

```mermaid
erDiagram
  import_files ||--o{ transactions : contains
  import_files ||--o{ import_mappings : uses
  import_files ||--o{ import_errors : has
  categories ||--o{ transactions : classifies
  categories ||--o{ category_rules : target
```

### 4.4 制約・保持方針

- `ImportFile.fileHash` で同一ファイルの再インポートを検出する
- `Transaction.dedupeKey` で明細単位の重複登録を防ぐ
- `dedupeKey` は、初期案として `usageDate + merchantName + cardUser + paymentMethod + billingMonth + usageAmount + billedAmount` から生成する
- `rawColumns` にはCSVの元行をJSONとして保持し、将来の再解釈に備える
- `ImportMapping` は、ヘッダー名または列位置から確定した `usageDate`、`merchantName`、`billingMonth`、`usageAmount`、`billedAmount` などの対応関係を保存する
- 必須項目は `usageDate`、`merchantName`、`billingMonth`、`usageAmount` または `billedAmount` とし、不足時は保存せずプレビューで確認させる
- `CategoryRule.priority` により複数ルール一致時の適用順を制御する
- 同一 `matchType`、`pattern`、`categoryId` の分類ルールはアプリケーション検証とDB複合ユニーク制約で重複登録を防ぐ
- カテゴリ削除時は、紐づく明細の `categoryId` を未分類へ戻す
- ファイル単位削除時は、対象 `ImportFile` に紐づく `Transaction` と `ImportError` を削除対象とする
- ファイル単位削除時は `ImportFile` 自体と `ImportMapping` も削除し、同一 `fileHash` のCSVを再インポート可能にする

## 5. 処理設計

### 5.1 主要ユースケース

| UC | 概要 | 関連画面 | 主な処理 |
|---|---|---|---|
| UC-1 | Vpass CSVをインポートする | インポート | 文字コード変換、パース、検証、プレビュー、保存 |
| UC-2 | 明細を検索・確認する | 明細一覧 | フィルタ、検索、並び替え、ページング |
| UC-3 | 月別支出を確認する | ダッシュボード | 月別集計、前月比、ランキング、グラフ |
| UC-4 | カテゴリと分類ルールを管理する | カテゴリ・ルール管理 | ルール作成、未分類候補のルール化、手動分類、再適用 |
| UC-5 | 分析結果を見る | 分析 | 月次推移、固定費候補、少額高頻度支出候補 |
| UC-6 | データを管理する | データ管理 | 履歴確認、ファイル単位削除、エクスポート |

### 5.2 主要処理フロー

#### CSVインポートフロー

```mermaid
sequenceDiagram
  actor User as 利用者
  participant UI as インポート画面
  participant API as Import API
  participant Parser as Vpass CSV Importer
  participant DB as SQLite

  User->>UI: CSVファイルを選択
  UI->>API: プレビュー要求
  API->>Parser: 文字コード変換・CSVパース
  Parser->>Parser: ヘッダー有無判定・フォーマット検出・項目マッピング
  Parser->>Parser: 必須項目チェック・型変換・エラー検出
  Parser-->>API: mappingCandidates / previewRows / errors / fileHash
  API->>DB: fileHash重複確認
  API-->>UI: マッピング候補、プレビュー、検証結果を返却
  User->>UI: インポート実行
  UI->>API: 保存要求
  API->>DB: ImportFile作成
  API->>DB: ImportMapping作成
  API->>DB: dedupeKey未登録のTransaction保存
  API->>DB: ImportError保存
  API-->>UI: 保存件数・スキップ件数を返却
```

#### 分類ルール適用フロー

1. ルールを `priority` 昇順で取得する
2. 対象明細の `merchantName` に対して `matchType` と `pattern` を評価する
3. 最初に一致したルールの `categoryId` を明細へ設定する
4. 手動設定済みカテゴリを上書きするかどうかは再適用時のオプションとする
5. 適用件数、変更件数、未分類件数を表示する

#### ダッシュボード集計フロー

1. 対象月と集計基準を受け取る
2. `billingMonth` または `usageDate` で対象明細を抽出する
3. `billedAmount` または `usageAmount` を集計対象にする
4. 合計、前月比、利用先別、カテゴリ別、日別推移を算出する
5. グラフ表示用の系列データに変換して返す

### 5.3 API / バックエンド処理方針

| 処理 | 方針 |
|---|---|
| CSVプレビュー | DB保存前に文字コード変換、ヘッダー有無判定、フォーマット検出、項目マッピング、型変換、重複候補検出を行う |
| CSV保存 | `ImportFile` と `Transaction` をトランザクションで保存し、保存済み分類ルールを未分類明細へ自動適用する |
| 重複判定 | ファイル単位は `fileHash`、明細単位は `dedupeKey` を使う |
| 明細検索 | 年月、利用先、カテゴリ、金額範囲、キーワードで絞り込む |
| カテゴリ変更 | 明細単位の `categoryId` を更新する |
| 未分類候補抽出 | `categoryId` が未設定の明細を `merchantName` 単位で集約し、件数順に提示する |
| ルール再適用 | 対象範囲と上書き条件を指定して一括更新する |
| 集計 | DBクエリまたはアプリケーションサービスで集計し、画面向け形式へ整形する |
| エクスポート | 明細をCSVで出力できるようにする |

#### APIエンドポイント案

初期版はローカル実行のGo APIとして、`/api/v1` 配下にエンドポイントを定義する。詳細なリクエスト/レスポンス項目は詳細設計で確定するが、概要設計では画面とバックエンド責務が分かる粒度まで定義する。

| 区分 | Method | Path | 概要 |
|---|---|---|---|
| ヘルスチェック | GET | `/api/v1/health` | Go APIの起動状態を確認する |
| CSVインポート | POST | `/api/v1/imports/preview` | CSVファイルを受け取り、文字コード変換・パース・マッピング候補・検証結果を返す。DB保存はしない |
| CSVインポート | POST | `/api/v1/imports` | プレビュー済みCSVを保存し、ImportFile/Transaction/ImportErrorを作成する |
| インポート履歴 | GET | `/api/v1/imports` | インポート済みファイル一覧を取得する |
| インポート履歴 | GET | `/api/v1/imports/{importFileId}` | インポートファイル単位の件数、エラー、取り込み結果を取得する |
| インポート履歴 | DELETE | `/api/v1/imports/{importFileId}` | 指定ファイル由来の明細、マッピング、エラー、インポート履歴を削除する |
| 明細 | GET | `/api/v1/transactions` | 年月、利用先、カテゴリ、金額範囲、キーワードで明細を検索する |
| 明細 | GET | `/api/v1/transactions/{transactionId}` | 明細1件の詳細を取得する |
| 明細 | PATCH | `/api/v1/transactions/{transactionId}` | 明細のカテゴリなど、編集可能項目を更新する |
| ダッシュボード | GET | `/api/v1/summary/monthly` | 対象月の支出合計、前月比、日別推移を取得する |
| ダッシュボード | GET | `/api/v1/summary/merchants` | 対象期間の利用先別ランキングを取得する |
| ダッシュボード | GET | `/api/v1/summary/categories` | 対象期間のカテゴリ別内訳を取得する |
| 分析 | GET | `/api/v1/analysis/monthly-trends` | 月別支出推移を取得する |
| 分析 | GET | `/api/v1/analysis/merchant-trends` | 利用先別の月次推移を取得する |
| 分析 | GET | `/api/v1/analysis/category-trends` | カテゴリ別の月次推移を取得する |
| 分析 | GET | `/api/v1/analysis/recurring-candidates` | 固定費候補を抽出する |
| 分析 | GET | `/api/v1/analysis/small-frequent` | 少額高頻度支出候補を抽出する |
| カテゴリ | GET | `/api/v1/categories` | カテゴリ一覧を取得する |
| カテゴリ | POST | `/api/v1/categories` | カテゴリを作成する |
| カテゴリ | PATCH | `/api/v1/categories/{categoryId}` | カテゴリ名・色を更新する |
| カテゴリ | DELETE | `/api/v1/categories/{categoryId}` | カテゴリを削除し、紐づく明細を未分類へ戻す |
| 分類ルール | GET | `/api/v1/category-rules` | 分類ルール一覧を取得する |
| 分類ルール | POST | `/api/v1/category-rules` | 利用先名に基づく分類ルールを作成する |
| 分類ルール | PATCH | `/api/v1/category-rules/{ruleId}` | 分類ルールの条件、カテゴリ、優先度を更新する |
| 分類ルール | DELETE | `/api/v1/category-rules/{ruleId}` | 分類ルールを削除する |
| 分類ルール | GET | `/api/v1/classification-candidates` | 未分類の利用先を件数順に集約して取得する |
| 分類ルール | POST | `/api/v1/category-rules/reapply` | 既存明細に分類ルールを再適用する |
| データ管理 | GET | `/api/v1/exports/transactions` | 明細データをCSVまたはJSONでエクスポートする |
| 設定 | GET | `/api/v1/settings` | 集計基準などのアプリ設定を取得する |
| 設定 | PATCH | `/api/v1/settings` | 集計基準などのアプリ設定を更新する |

API設計上の補足:

- 初期版はローカル利用のため認証なしとするが、クラウド化時に `/api/v1` 配下へ認証ミドルウェアを追加できる構成にする
- 一覧取得APIは `page`、`pageSize`、`sort`、`order` を共通クエリとして扱う
- 集計APIは `month`、`from`、`to`、`basis` を共通クエリとして扱う。`basis` は `billingMonth` / `usageDate`、金額基準は `billedAmount` / `usageAmount` を想定する
- インポート実行APIは、プレビューで確定したマッピングを受け取り、サーバー側で再検証してから保存する
- エラー応答は `{ code, message, details }` の共通形式にする
- CSVアップロード系APIでは、実明細データをログに出力しない

### 5.4 エラーハンドリング

| ケース | 処理方針 |
|---|---|
| 文字コード変換失敗 | CSV形式・文字コードが想定外であることを表示する |
| 必須項目不足 | 利用日、利用先、支払月、金額など必須項目にマッピングできない場合、保存せずプレビュー上に表示する |
| マッピング不確定 | ヘッダー名や列位置から項目を確定できない場合、ユーザーが画面上で対応項目を選べるようにする |
| 日付変換エラー | 対象行を保存対象外にし、修正不能行として表示する |
| 金額変換エラー | カンマ、空文字、符号を考慮し、それでも失敗した場合はエラー行にする |
| 同一ファイル再インポート | 既に取り込み済みであることを表示し、保存を止める |
| 明細重複 | 保存時にスキップし、スキップ件数を表示する |
| DB保存失敗 | インポート全体をロールバックし、再試行可能な状態にする |

## 6. 外部インターフェース

### 6.1 外部入力

| 入力 | 内容 | 設計方針 |
|---|---|---|
| Vpass CSV | CP932/Shift_JIS系。ヘッダー有無や列数は固定しない | Vpass CSV Importerで文字コード、ヘッダー有無、フォーマット、項目マッピングを検出する |
| ユーザー操作 | カテゴリ変更、ルール作成、フィルタ条件 | UIからAPIへ送信する |

### 6.2 外部出力

| 出力 | 内容 |
|---|---|
| 明細エクスポート | 保存済み明細をCSVまたはJSONで出力 |

### 6.3 外部サービス連携

初期版では外部サービス連携を行わない。

将来拡張時は、PayPay、銀行CSV、他カードCSVをそれぞれ独立したインポーターとして追加する。Vpassへの自動ログインやスクレイピングは初期スコープ外であり、将来検討時も利用規約・セキュリティ・保守性を再評価する。

## 7. 非機能設計

### 7.1 セキュリティ

- 明細データを外部サーバーへ送信しない
- 初期版ではローカル環境のみで動作させる
- 認証は初期スコープ外とする
- 将来的にクラウド化する場合は、認証、暗号化、バックアップ、アクセス制御を再設計する
- サンプルやログに実明細データを不用意に出力しない
- Go APIはローカルホストで起動し、初期版では外部ネットワーク公開しない

### 7.2 パフォーマンス

- 数千件程度の明細を快適に表示・検索できることを目標とする
- CSVインポートは数秒以内の完了を目標とする
- 明細一覧はページングを基本とし、必要に応じて仮想スクロールを検討する
- 検索・集計で使う `usageDate`、`billingMonth`、`merchantName`、`categoryId` にはインデックスを検討する

### 7.3 保守性

- Vpass固有のフォーマット検出、項目マッピング、文字コード、日付/金額変換はインポーターに閉じ込める
- 分類ルールは明細データとは独立して管理する
- 集計ロジックは画面コンポーネントから分離し、分析サービスとして再利用可能にする
- 将来のCSV形式追加に備え、インポーターの共通インターフェースを用意する

### 7.4 運用・バックアップ

- SQLite DBファイルをバックアップ対象とする
- 明細をCSVエクスポート可能にする
- インポート履歴により、不要データをファイル単位で削除できるようにする
- DBスキーマ変更はGORM AutoMigrateまたはGo製マイグレーションツールで管理する

## 8. ディレクトリ構成案

```text
vpass-statement-analyzer/
  frontend/
    src/
      pages/
        ImportPage.vue
        DashboardPage.vue
        TransactionsPage.vue
        CategoriesPage.vue
        AnalysisPage.vue
        DataManagementPage.vue
      components/
        charts/
        tables/
        forms/
      api/
        client.ts
      types/
  backend/
    cmd/
      server/
        main.go
    internal/
      api/
      importer/
        vpass_csv_importer.go
      encoding/
      analytics/
      category/
      repository/
      model/
      database/
    migrations/
  data/
    vpass-statement-analyzer.sqlite
  docs/
  requirements.md
  overview-design.md
```

## 9. 設計上の判断

| 判断 | 内容 | 理由 |
|---|---|---|
| ローカルWebアプリ | 初期版は外部公開しない | 金融系明細データを外部送信せずに扱うため |
| Go + Vue構成 | フロントエンドとバックエンドを分離する | 実務スタックに寄せつつ、CSV処理と画面責務を明確に分けるため |
| SQLite採用 | 明細、カテゴリ、ルール、履歴をローカルDBへ保存 | 本人利用・数千件規模に対して軽量で十分なため |
| GORM採用 | Go側でSQLiteアクセスとモデル管理を行う | CRUD実装を簡潔にし、Go API内でDB責務を完結させるため |
| Vpass専用Importer | 文字コード、ヘッダー有無判定、フォーマット検出、項目マッピングを専用モジュールに分離 | CSVの列構成を固定せず、他CSV追加時にも影響範囲を限定するため |
| プレビュー後保存 | CSVを即保存せず、検証結果を見せてから保存する | 不正行や文字化け、列ズレを早期発見するため |
| 二段階重複判定 | ファイル単位と明細単位で重複を防ぐ | 同じCSV再投入と複数CSV間の重複の両方を避けるため |
| 分類ルール分離 | CategoryRuleをTransactionから独立管理 | ルール改善と再適用を可能にするため |
| 外部AI自動分類は初期対象外 | ルールベース分類と未分類候補支援をMVPにする | ローカル・軽量・説明可能な分類を優先するため |

## 10. 未決事項

| 項目 | 確認内容 | 設計への影響 |
|---|---|---|
| 主集計金額 | 利用金額と請求金額のどちらを主集計にするか | ダッシュボード、分析、ランキングの基準が変わる |
| 月次集計基準 | 支払月ベースか利用日ベースか | 月別推移、前月比、固定費判定の基準が変わる |
| 初期カテゴリ | コンビニ、スーパー、交通、外食、サブスクなどをどう分けるか | 初期データ、ルール例、画面表示が変わる |
| Suicaチャージ | 交通費か電子マネーチャージか | カテゴリ分類ルールと分析解釈が変わる |
| デスクトップアプリ化 | ローカルWebアプリのままか、将来デスクトップ化するか | 配布方法、DB保存場所、バックアップ導線が変わる |
| 手動分類の優先度 | ルール再適用時に手動カテゴリを上書きするか | 再適用処理とユーザー確認UIが変わる |
| CSVマッピングUI | マッピング不確定時にユーザーが列対応を修正できるようにするか | インポート画面の複雑さと対応可能CSV形式が変わる |

## 11. 要件トレーサビリティ

| 要件 | 設計での反映 |
|---|---|
| CP932/Shift_JIS CSV読み込み | Vpass CSV Importer、Encoding処理、CSVプレビュー |
| 列構成を固定しないCSV取り込み | ヘッダー有無判定、フォーマット検出、ImportMapping、`rawColumns`保持 |
| 必須項目不足・金額変換エラー表示 | ImportError、プレビュー画面、エラーハンドリング |
| 重複登録防止 | `fileHash`、`dedupeKey`、保存時スキップ |
| 明細一覧 | 明細一覧画面、transactionsテーブル、検索/フィルタAPI |
| ダッシュボード | 集計サービス、月別合計、前月比、ランキング、グラフ |
| カテゴリ管理 | Category、CategoryRule、カテゴリ・ルール管理画面 |
| ルール再適用 | 分類ルール適用フロー、上書きオプション |
| 分析機能 | 分析画面、月次推移、固定費候補、少額高頻度候補 |
| ローカルDB保存 | SQLite、GORM、バックアップ/エクスポート方針 |
| ファイル単位削除 | ImportFile、ファイル単位削除処理 |
| 外部送信しない | ローカルWebアプリ、外部サービス連携なし |
| 将来CSV追加 | インポーター分離、共通インターフェース |

## 12. 次の開発ステップ

1. 集計基準を決める: 請求金額/利用金額、支払月/利用日
2. Go APIのプロジェクト構成とGORMモデルを作成する
3. Vue 3 + TypeScript + Viteのフロントエンドを作成する
4. Vpass CSV Importerの最小実装を作る
5. CSVプレビューAPIとインポート画面を作る
6. SQLite保存と重複判定を実装する
7. 明細一覧APIと明細一覧画面を作る
8. 月別合計と利用先ランキングを実装する
9. カテゴリ管理と手動分類を追加する
10. 分類ルール作成と再適用を実装する
11. グラフと分析画面を追加する
