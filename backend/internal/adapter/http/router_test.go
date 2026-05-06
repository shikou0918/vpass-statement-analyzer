package httpadapter

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"vpass-statement-analyzer/backend/internal/infra/database"
	"vpass-statement-analyzer/backend/internal/usecase"
)

func newTestServer(t *testing.T) http.Handler {
	t.Helper()
	db, err := database.Open("file:vpass_test?mode=memory&cache=shared")
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	if err := database.Migrate(db); err != nil {
		t.Fatalf("migrate database: %v", err)
	}
	repos := database.NewRepositories(db)
	return NewRouter(usecase.NewApp(repos, database.NewTxManager(db)), "http://localhost:5173")
}

func TestCategoryEndpoints(t *testing.T) {
	router := newTestServer(t)

	createReq := httptest.NewRequest(http.MethodPost, "/categories", strings.NewReader(`{"name":"食費","color":"#22c55e"}`))
	createReq.Header.Set("Content-Type", "application/json")
	createRes := httptest.NewRecorder()
	router.ServeHTTP(createRes, createReq)
	if createRes.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d: %s", createRes.Code, createRes.Body.String())
	}

	var created struct {
		ID    int64  `json:"id"`
		Name  string `json:"name"`
		Color string `json:"color"`
	}
	if err := json.Unmarshal(createRes.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode category: %v", err)
	}
	if created.ID == 0 || created.Name != "食費" || created.Color != "#22c55e" {
		t.Fatalf("unexpected category: %+v", created)
	}

	listReq := httptest.NewRequest(http.MethodGet, "/categories", nil)
	listRes := httptest.NewRecorder()
	router.ServeHTTP(listRes, listReq)
	if listRes.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", listRes.Code)
	}
	if !strings.Contains(listRes.Body.String(), "食費") {
		t.Fatalf("list response should include created category: %s", listRes.Body.String())
	}
}

func TestCategoryValidationError(t *testing.T) {
	router := newTestServer(t)

	req := httptest.NewRequest(http.MethodPost, "/categories", strings.NewReader(`{"name":"","color":"blue"}`))
	req.Header.Set("Content-Type", "application/json")
	res := httptest.NewRecorder()
	router.ServeHTTP(res, req)

	if res.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", res.Code)
	}
	if !strings.Contains(res.Body.String(), "BAD_REQUEST") {
		t.Fatalf("expected BAD_REQUEST error body, got %s", res.Body.String())
	}
}

func TestImportPreviewAndDuplicateImport(t *testing.T) {
	router := newTestServer(t)
	csvBody := "利用日,利用先,支払月,利用金額,請求金額\n2026-05-01,コンビニ,2026-06,1000,1000\n"

	preview := createPreview(t, router, csvBody)
	if preview.PreviewID == "" {
		t.Fatal("previewId should not be empty")
	}
	if preview.DuplicateFile {
		t.Fatal("first preview should not be duplicate")
	}

	mapping := map[string]string{}
	for _, candidate := range preview.MappingCandidates {
		mapping[strconv.Itoa(candidate.SourceColumnIndex)] = candidate.TargetField
	}

	importReqBody, err := json.Marshal(map[string]any{
		"previewId":        preview.PreviewID,
		"fileHash":         preview.FileHash,
		"confirmedMapping": mapping,
		"options": map[string]any{
			"applyCategoryRules": true,
		},
	})
	if err != nil {
		t.Fatalf("marshal import request: %v", err)
	}
	importReq := httptest.NewRequest(http.MethodPost, "/imports", bytes.NewReader(importReqBody))
	importReq.Header.Set("Content-Type", "application/json")
	importRes := httptest.NewRecorder()
	router.ServeHTTP(importRes, importReq)
	if importRes.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d: %s", importRes.Code, importRes.Body.String())
	}
	if !strings.Contains(importRes.Body.String(), `"importedCount":1`) {
		t.Fatalf("expected importedCount 1, got %s", importRes.Body.String())
	}

	duplicatePreview := createPreview(t, router, csvBody)
	if !duplicatePreview.DuplicateFile {
		t.Fatal("second preview should detect duplicate file")
	}
}

func createPreview(t *testing.T, router http.Handler, csvBody string) importPreviewResponse {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", "vpass.csv")
	if err != nil {
		t.Fatalf("create form file: %v", err)
	}
	if _, err := part.Write([]byte(csvBody)); err != nil {
		t.Fatalf("write form file: %v", err)
	}
	if err := writer.WriteField("sourceType", "vpass"); err != nil {
		t.Fatalf("write sourceType: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close writer: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/import-previews", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	res := httptest.NewRecorder()
	router.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", res.Code, res.Body.String())
	}

	var preview importPreviewResponse
	if err := json.Unmarshal(res.Body.Bytes(), &preview); err != nil {
		t.Fatalf("decode preview: %v", err)
	}
	return preview
}

type importPreviewResponse struct {
	PreviewID         string                   `json:"previewId"`
	FileHash          string                   `json:"fileHash"`
	DuplicateFile     bool                     `json:"duplicateFile"`
	MappingCandidates []importMappingCandidate `json:"mappingCandidates"`
}

type importMappingCandidate struct {
	SourceColumnIndex int    `json:"sourceColumnIndex"`
	TargetField       string `json:"targetField"`
}
