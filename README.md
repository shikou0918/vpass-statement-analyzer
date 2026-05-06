# Vpass Statement Analyzer

Vpass明細CSVをローカルで取り込み、明細検索、カテゴリ分類、集計、分析を行うWebアプリです。

## 構成

- `backend/`: Go + GORM + SQLite API
- `frontend/`: Vue 3 + TypeScript + Vite UI
- `requirements.md`, `overview-design.md`, `api-design.md`, `openapi.yaml`, `ui-design.md`: 設計書

## 起動

```sh
cd backend
go mod download
go run ./cmd/server
```

```sh
cd frontend
npm install
npm run dev
```

APIの既定URLは `http://localhost:8080` です。

