# Vpass Statement Analyzer

Vpass明細CSVをローカルで取り込み、明細検索、カテゴリ分類、集計、分析を行うWebアプリです。

## 構成

- `backend/`: Go + GORM + SQLite API
- `frontend/`: Vue 3 + TypeScript + Vite UI
- `requirements.md`, `overview-design.md`, `api-design.md`, `openapi.yaml`, `ui-design.md`: 設計書

## 起動

Docker Composeで起動する場合:

```sh
docker compose up --build
```

Compose v2が入っていない環境では以下を使用する。

```sh
docker-compose up --build
```

フロントエンドは `http://localhost:5173`、APIは `http://localhost:8080` で起動します。SQLite DBはDocker volume `vpass-statement-analyzer_backend-data` に保存されます。

Docker Compose起動中は、`frontend/` の変更はVite HMRでブラウザへ反映され、`backend/` のGoファイル変更は開発用ウォッチャーが検知してAPIサーバを再起動します。

ローカルで個別に起動する場合:

```sh
cd backend
go mod download
go run ./cmd/api
```

```sh
cd frontend
npm install
npm run dev
```

APIの既定URLは `http://localhost:8080` です。
