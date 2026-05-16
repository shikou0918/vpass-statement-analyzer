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

	"vpass-statement-analyzer/backend/internal/infrastructure/database"
	"vpass-statement-analyzer/backend/internal/usecase"
)

func newTestServer(t *testing.T) http.Handler {
	t.Helper()
	dbName := strings.NewReplacer("/", "_", " ", "_").Replace(t.Name())
	db, err := database.Open("file:" + dbName + "?mode=memory&cache=shared")
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

func TestDuplicateCategoryRuleRejected(t *testing.T) {
	router := newTestServer(t)

	createCategoryReq := httptest.NewRequest(http.MethodPost, "/categories", strings.NewReader(`{"name":"コンビニ","color":"#22c55e"}`))
	createCategoryReq.Header.Set("Content-Type", "application/json")
	createCategoryRes := httptest.NewRecorder()
	router.ServeHTTP(createCategoryRes, createCategoryReq)
	if createCategoryRes.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d: %s", createCategoryRes.Code, createCategoryRes.Body.String())
	}
	var category struct {
		ID int64 `json:"id"`
	}
	if err := json.Unmarshal(createCategoryRes.Body.Bytes(), &category); err != nil {
		t.Fatalf("decode category: %v", err)
	}

	body := `{"matchType":"contains","pattern":"ローソン","categoryId":` + strconv.FormatInt(category.ID, 10) + `,"priority":1}`
	createRuleReq := httptest.NewRequest(http.MethodPost, "/category-rules", strings.NewReader(body))
	createRuleReq.Header.Set("Content-Type", "application/json")
	createRuleRes := httptest.NewRecorder()
	router.ServeHTTP(createRuleRes, createRuleReq)
	if createRuleRes.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d: %s", createRuleRes.Code, createRuleRes.Body.String())
	}

	duplicateReq := httptest.NewRequest(http.MethodPost, "/category-rules", strings.NewReader(body))
	duplicateReq.Header.Set("Content-Type", "application/json")
	duplicateRes := httptest.NewRecorder()
	router.ServeHTTP(duplicateRes, duplicateReq)
	if duplicateRes.Code != http.StatusConflict {
		t.Fatalf("expected status 409, got %d: %s", duplicateRes.Code, duplicateRes.Body.String())
	}
	if !strings.Contains(duplicateRes.Body.String(), "CONFLICT") {
		t.Fatalf("expected CONFLICT error body, got %s", duplicateRes.Body.String())
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

func TestImportAppliesExistingCategoryRules(t *testing.T) {
	router := newTestServer(t)

	createCategoryReq := httptest.NewRequest(http.MethodPost, "/categories", strings.NewReader(`{"name":"コンビニ","color":"#22c55e"}`))
	createCategoryReq.Header.Set("Content-Type", "application/json")
	createCategoryRes := httptest.NewRecorder()
	router.ServeHTTP(createCategoryRes, createCategoryReq)
	if createCategoryRes.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d: %s", createCategoryRes.Code, createCategoryRes.Body.String())
	}
	var category struct {
		ID int64 `json:"id"`
	}
	if err := json.Unmarshal(createCategoryRes.Body.Bytes(), &category); err != nil {
		t.Fatalf("decode category: %v", err)
	}

	ruleBody := `{"matchType":"contains","pattern":"ローソン","categoryId":` + strconv.FormatInt(category.ID, 10) + `,"priority":1}`
	createRuleReq := httptest.NewRequest(http.MethodPost, "/category-rules", strings.NewReader(ruleBody))
	createRuleReq.Header.Set("Content-Type", "application/json")
	createRuleRes := httptest.NewRecorder()
	router.ServeHTTP(createRuleRes, createRuleReq)
	if createRuleRes.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d: %s", createRuleRes.Code, createRuleRes.Body.String())
	}

	preview := createPreview(t, router, "利用日,利用先,支払月,利用金額,請求金額\n2026-05-01,ローソン,2026-06,1000,1000\n")
	mapping := map[string]string{}
	for _, candidate := range preview.MappingCandidates {
		mapping[strconv.Itoa(candidate.SourceColumnIndex)] = candidate.TargetField
	}
	importReqBody, err := json.Marshal(map[string]any{
		"previewId":        preview.PreviewID,
		"fileHash":         preview.FileHash,
		"confirmedMapping": mapping,
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

	listReq := httptest.NewRequest(http.MethodGet, "/transactions?page=1&pageSize=50", nil)
	listRes := httptest.NewRecorder()
	router.ServeHTTP(listRes, listReq)
	if listRes.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", listRes.Code, listRes.Body.String())
	}
	var list struct {
		Items []struct {
			CategoryID *int64 `json:"categoryId"`
		} `json:"items"`
	}
	if err := json.Unmarshal(listRes.Body.Bytes(), &list); err != nil {
		t.Fatalf("decode transactions: %v", err)
	}
	if len(list.Items) != 1 || list.Items[0].CategoryID == nil || *list.Items[0].CategoryID != category.ID {
		t.Fatalf("imported transaction should be categorized by existing rule: %+v", list.Items)
	}
}

func TestDeleteImportRemovesRelatedTransactionsAndAllowsReimport(t *testing.T) {
	router := newTestServer(t)
	csvBody := "利用日,利用先,支払月,利用金額,請求金額\n2026-05-01,コンビニ,2026-06,1000,1000\n"

	preview := createPreview(t, router, csvBody)
	imported := createImportFromPreview(t, router, preview)
	if imported.ImportFile.ID == 0 {
		t.Fatal("import id should not be empty")
	}

	deleteReq := httptest.NewRequest(http.MethodDelete, "/imports/"+strconv.FormatInt(imported.ImportFile.ID, 10), nil)
	deleteRes := httptest.NewRecorder()
	router.ServeHTTP(deleteRes, deleteReq)
	if deleteRes.Code != http.StatusNoContent {
		t.Fatalf("expected status 204, got %d: %s", deleteRes.Code, deleteRes.Body.String())
	}

	listReq := httptest.NewRequest(http.MethodGet, "/transactions?page=1&pageSize=50", nil)
	listRes := httptest.NewRecorder()
	router.ServeHTTP(listRes, listReq)
	if listRes.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", listRes.Code, listRes.Body.String())
	}
	var list struct {
		Items []struct {
			ID int64 `json:"id"`
		} `json:"items"`
	}
	if err := json.Unmarshal(listRes.Body.Bytes(), &list); err != nil {
		t.Fatalf("decode transactions: %v", err)
	}
	if len(list.Items) != 0 {
		t.Fatalf("expected related transactions to be deleted, got %d", len(list.Items))
	}

	nextPreview := createPreview(t, router, csvBody)
	if nextPreview.DuplicateFile {
		t.Fatal("deleted import should not be treated as duplicate")
	}
}

func TestHeaderlessVpassImportFormat(t *testing.T) {
	router := newTestServer(t)
	csvBody := strings.Join([]string{
		"2026/4/30,セブン－イレブン,ご本人,1回払い,,'26/05,213,213,,,,,",
		"2026/4/30,キャッシュバック（ポイント交換）,ご本人,,,'26/05,-30000,-30000,,,,,",
	}, "\n") + "\n"

	preview := createPreview(t, router, csvBody)
	if preview.DuplicateFile {
		t.Fatal("first preview should not be duplicate")
	}
	if len(preview.PreviewRows) != 2 {
		t.Fatalf("expected 2 preview rows, got %d", len(preview.PreviewRows))
	}
	if got := strings.Join(preview.PreviewRows[0].RawColumns, ","); got != "2026/4/30,セブン－イレブン,ご本人,1回払い,,'26/05,213,213,,,,," {
		t.Fatalf("unexpected raw preview row: %s", got)
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
	if !strings.Contains(importRes.Body.String(), `"importedCount":2`) {
		t.Fatalf("expected importedCount 2, got %s", importRes.Body.String())
	}

	listReq := httptest.NewRequest(http.MethodGet, "/transactions?page=1&pageSize=50&billingMonth=2026-05&sort=usageDate&order=desc", nil)
	listRes := httptest.NewRecorder()
	router.ServeHTTP(listRes, listReq)
	if listRes.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", listRes.Code, listRes.Body.String())
	}
	var list struct {
		Items []struct {
			UsageDate    string `json:"usageDate"`
			MerchantName string `json:"merchantName"`
			BillingMonth string `json:"billingMonth"`
			BilledAmount *int64 `json:"billedAmount"`
		} `json:"items"`
	}
	if err := json.Unmarshal(listRes.Body.Bytes(), &list); err != nil {
		t.Fatalf("decode transactions: %v", err)
	}
	if len(list.Items) < 2 {
		t.Fatalf("expected at least 2 transactions, got %d: %s", len(list.Items), listRes.Body.String())
	}
	if list.Items[0].UsageDate == "" || list.Items[0].MerchantName == "" || list.Items[0].BillingMonth == "" || list.Items[0].BilledAmount == nil {
		t.Fatalf("transaction response should use camelCase fields: %+v", list.Items[0])
	}
}

func TestClassificationCandidatesEndpoint(t *testing.T) {
	router := newTestServer(t)
	csvBody := "利用日,利用先,支払月,利用金額,請求金額\n2026-05-01,ローソン,2026-06,1000,1000\n2026-05-02,ローソン,2026-06,500,500\n2026-05-03,大東ガス,2026-06,3000,3000\n"
	preview := createPreview(t, router, csvBody)
	createImportFromPreview(t, router, preview)

	req := httptest.NewRequest(http.MethodGet, "/classification-candidates?limit=10", nil)
	res := httptest.NewRecorder()
	router.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", res.Code, res.Body.String())
	}
	var body struct {
		Items []struct {
			MerchantName     string `json:"merchantName"`
			TransactionCount int64  `json:"transactionCount"`
		} `json:"items"`
	}
	if err := json.Unmarshal(res.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode candidates: %v", err)
	}
	if len(body.Items) < 2 {
		t.Fatalf("expected at least 2 candidates, got %d: %s", len(body.Items), res.Body.String())
	}
	if body.Items[0].MerchantName != "ローソン" || body.Items[0].TransactionCount != 2 {
		t.Fatalf("expected candidates ordered by count, got %+v", body.Items[0])
	}
}

func TestCompactVpassImportSkipsMetadataAndUsesFileNameBillingMonth(t *testing.T) {
	router := newTestServer(t)
	csvBody := strings.Join([]string{
		"市川　志功　様,4980-00**-****-****,Ｏｌｉｖｅゴールド／クレジット",
		"2026/03/01,カブシキガイシャドミノピザジャパン,3190,１,１,3190,",
		"2026/03/02,ローソン,254,１,１,254,",
	}, "\n") + "\n"

	preview := createPreviewWithFileName(t, router, "202604.csv", csvBody)
	if len(preview.PreviewRows) != 2 {
		t.Fatalf("expected metadata row to be skipped, got %d preview rows", len(preview.PreviewRows))
	}
	if got := preview.PreviewRows[0].RawColumns[0]; got != "2026/03/01" {
		t.Fatalf("first preview row should be first transaction row, got %s", got)
	}
	mapping := map[string]string{}
	for _, candidate := range preview.MappingCandidates {
		mapping[strconv.Itoa(candidate.SourceColumnIndex)] = candidate.TargetField
	}
	for _, target := range []string{"usageDate", "merchantName", "usageAmount", "billedAmount", "billingMonth"} {
		if !mappingHasTarget(mapping, target) {
			t.Fatalf("mapping should include %s: %+v", target, mapping)
		}
	}

	imported := createImportFromPreview(t, router, preview)
	if imported.ImportedCount != 2 {
		t.Fatalf("expected importedCount 2, got %d", imported.ImportedCount)
	}

	listReq := httptest.NewRequest(http.MethodGet, "/transactions?page=1&pageSize=50&billingMonth=2026-04&sort=usageDate&order=asc", nil)
	listRes := httptest.NewRecorder()
	router.ServeHTTP(listRes, listReq)
	if listRes.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", listRes.Code, listRes.Body.String())
	}
	var list struct {
		Items []struct {
			MerchantName string `json:"merchantName"`
			BillingMonth string `json:"billingMonth"`
			UsageAmount  *int64 `json:"usageAmount"`
			BilledAmount *int64 `json:"billedAmount"`
		} `json:"items"`
	}
	if err := json.Unmarshal(listRes.Body.Bytes(), &list); err != nil {
		t.Fatalf("decode transactions: %v", err)
	}
	if len(list.Items) != 2 {
		t.Fatalf("expected 2 transactions for billingMonth 2026-04, got %d: %s", len(list.Items), listRes.Body.String())
	}
	if list.Items[0].BillingMonth != "2026-04" || list.Items[0].UsageAmount == nil || list.Items[0].BilledAmount == nil {
		t.Fatalf("unexpected imported compact row: %+v", list.Items[0])
	}
}

func TestMonthlySummaryIncludesPreviousMonthAmount(t *testing.T) {
	router := newTestServer(t)
	csvBody := strings.Join([]string{
		"利用日,利用先,支払月,利用金額,請求金額",
		"2026-04-01,前月店,2026-04,1000,1000",
		"2026-05-01,当月店,2026-05,2500,2500",
	}, "\n") + "\n"

	preview := createPreview(t, router, csvBody)
	createImportFromPreview(t, router, preview)

	req := httptest.NewRequest(http.MethodGet, "/summaries/monthly?month=2026-05&basisDate=billingMonth&basisAmount=billedAmount", nil)
	res := httptest.NewRecorder()
	router.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", res.Code, res.Body.String())
	}
	var body struct {
		TotalAmount    int64 `json:"totalAmount"`
		PreviousAmount int64 `json:"previousAmount"`
		DiffAmount     int64 `json:"diffAmount"`
	}
	if err := json.Unmarshal(res.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode monthly summary: %v", err)
	}
	if body.TotalAmount != 2500 || body.PreviousAmount != 1000 || body.DiffAmount != 1500 {
		t.Fatalf("unexpected monthly summary: %+v", body)
	}
}

func TestVpassImportsCanBeSeparatedByCreditCard(t *testing.T) {
	router := newTestServer(t)
	cardACSV := strings.Join([]string{
		"市川　志功　様,4980-00**-****-****,Ｏｌｉｖｅゴールド／クレジット",
		"2026/04/01,ローソン,1000,１,１,1000,",
	}, "\n") + "\n"
	cardBCSV := strings.Join([]string{
		"市川　志功　様,1111-22**-****-****,別カード／クレジット",
		"2026/04/01,ローソン,1000,１,１,1000,",
	}, "\n") + "\n"

	cardAPreview := createPreviewWithFileName(t, router, "202605-a.csv", cardACSV)
	if !strings.Contains(cardAPreview.DetectedCreditCardName, "Ｏｌｉｖｅゴールド") {
		t.Fatalf("expected credit card name to be detected, got %s", cardAPreview.DetectedCreditCardName)
	}
	cardAImport := createImportFromPreviewWithCardName(t, router, cardAPreview, cardAPreview.DetectedCreditCardName)
	cardBPreview := createPreviewWithFileName(t, router, "202605-b.csv", cardBCSV)
	cardBImport := createImportFromPreviewWithCardName(t, router, cardBPreview, cardBPreview.DetectedCreditCardName)

	if cardAImport.ImportedCount != 1 || cardBImport.ImportedCount != 1 {
		t.Fatalf("same transaction on different cards should import separately: %+v %+v", cardAImport, cardBImport)
	}
	if cardAImport.ImportFile.CreditCardID == nil || cardBImport.ImportFile.CreditCardID == nil || *cardAImport.ImportFile.CreditCardID == *cardBImport.ImportFile.CreditCardID {
		t.Fatalf("imports should be linked to different credit cards: %+v %+v", cardAImport.ImportFile, cardBImport.ImportFile)
	}

	listReq := httptest.NewRequest(http.MethodGet, "/transactions?page=1&pageSize=50&billingMonth=2026-05&creditCardId="+strconv.FormatInt(*cardAImport.ImportFile.CreditCardID, 10), nil)
	listRes := httptest.NewRecorder()
	router.ServeHTTP(listRes, listReq)
	if listRes.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", listRes.Code, listRes.Body.String())
	}
	var list struct {
		Items []struct {
			CreditCardID *int64 `json:"creditCardId"`
		} `json:"items"`
	}
	if err := json.Unmarshal(listRes.Body.Bytes(), &list); err != nil {
		t.Fatalf("decode transactions: %v", err)
	}
	if len(list.Items) != 1 || list.Items[0].CreditCardID == nil || *list.Items[0].CreditCardID != *cardAImport.ImportFile.CreditCardID {
		t.Fatalf("expected only card A transactions, got %+v", list.Items)
	}
}

func createPreview(t *testing.T, router http.Handler, csvBody string) importPreviewResponse {
	return createPreviewWithFileName(t, router, "vpass.csv", csvBody)
}

func createPreviewWithFileName(t *testing.T, router http.Handler, fileName string, csvBody string) importPreviewResponse {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", fileName)
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

func mappingHasTarget(mapping map[string]string, target string) bool {
	for _, value := range mapping {
		if value == target {
			return true
		}
	}
	return false
}

func createImportFromPreview(t *testing.T, router http.Handler, preview importPreviewResponse) testCreateImportResponse {
	return createImportFromPreviewWithCardName(t, router, preview, "")
}

func createImportFromPreviewWithCardName(t *testing.T, router http.Handler, preview importPreviewResponse, creditCardName string) testCreateImportResponse {
	t.Helper()
	mapping := map[string]string{}
	for _, candidate := range preview.MappingCandidates {
		mapping[strconv.Itoa(candidate.SourceColumnIndex)] = candidate.TargetField
	}
	importReqBody, err := json.Marshal(map[string]any{
		"previewId":        preview.PreviewID,
		"fileHash":         preview.FileHash,
		"creditCardName":   creditCardName,
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
	var result testCreateImportResponse
	if err := json.Unmarshal(importRes.Body.Bytes(), &result); err != nil {
		t.Fatalf("decode import response: %v", err)
	}
	return result
}

type importPreviewResponse struct {
	PreviewID              string                   `json:"previewId"`
	FileHash               string                   `json:"fileHash"`
	DetectedCreditCardName string                   `json:"detectedCreditCardName"`
	DuplicateFile          bool                     `json:"duplicateFile"`
	MappingCandidates      []importMappingCandidate `json:"mappingCandidates"`
	PreviewRows            []importPreviewRow       `json:"previewRows"`
}

type importMappingCandidate struct {
	SourceColumnIndex int    `json:"sourceColumnIndex"`
	TargetField       string `json:"targetField"`
}

type importPreviewRow struct {
	RawColumns []string `json:"rawColumns"`
}

type testCreateImportResponse struct {
	ImportFile    testImportFileResponse `json:"importFile"`
	ImportedCount int                    `json:"importedCount"`
}

type testImportFileResponse struct {
	ID           int64  `json:"id"`
	CreditCardID *int64 `json:"creditCardId"`
}
