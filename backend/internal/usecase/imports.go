package usecase

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"golang.org/x/text/encoding/japanese"
	"golang.org/x/text/transform"

	"vpass-statement-analyzer/backend/internal/domain"
)

var previewStore = struct {
	sync.RWMutex
	items map[string]storedPreview
}{items: map[string]storedPreview{}}

type storedPreview struct {
	Preview ImportPreview
	Rows    [][]string
	Header  []string
}

func (a *App) CreateImportPreview(ctx context.Context, fileName string, r io.Reader) (ImportPreview, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return ImportPreview{}, BadRequest("CSVファイルを読み込めません", nil)
	}
	if len(bytes.TrimSpace(data)) == 0 {
		return ImportPreview{}, Validation("CSVファイルが空です", nil)
	}

	hashBytes := sha256.Sum256(data)
	fileHash := hex.EncodeToString(hashBytes[:])
	duplicate := false
	if existing, err := a.repos.Imports().FindByHash(ctx, fileHash); err != nil {
		return ImportPreview{}, err
	} else if existing != nil {
		duplicate = true
	}

	decoded, encodingName, err := decodeCSV(data)
	if err != nil {
		return ImportPreview{}, Validation("文字コードを変換できません", nil)
	}

	reader := csv.NewReader(strings.NewReader(decoded))
	reader.FieldsPerRecord = -1
	records, err := reader.ReadAll()
	if err != nil {
		return ImportPreview{}, Validation("CSVとして解析できません", map[string]any{"reason": err.Error()})
	}
	if len(records) == 0 {
		return ImportPreview{}, Validation("CSVに明細行がありません", nil)
	}

	hasHeader := detectHeader(records[0])
	header := []string{}
	rows := records
	if hasHeader {
		header = records[0]
		rows = records[1:]
	} else {
		rows = dropLeadingNonTransactionRows(rows)
	}

	if month := billingMonthFromFileName(fileName); month != "" {
		candidates := inferMapping(header, rows)
		if !hasMappingTarget(candidates, "billingMonth") {
			rows = appendColumn(rows, month)
		}
	}
	candidates := inferMapping(header, rows)
	errors := validateRows(rows, candidates)
	previewRows := buildPreviewRows(rows, candidates, 50)
	previewID := fmt.Sprintf("%x", sha256.Sum256([]byte(fileHash+time.Now().String())))[:24]

	preview := ImportPreview{
		PreviewID:         previewID,
		FileName:          fileName,
		FileHash:          fileHash,
		DetectedFormat:    "vpass",
		Encoding:          encodingName,
		HasHeader:         hasHeader,
		MappingCandidates: candidates,
		PreviewRows:       previewRows,
		Errors:            errors,
		DuplicateFile:     duplicate,
	}

	previewStore.Lock()
	previewStore.items[previewID] = storedPreview{Preview: preview, Rows: rows, Header: header}
	previewStore.Unlock()

	return preview, nil
}

func (a *App) CreateImport(ctx context.Context, in CreateImportInput) (CreateImportResult, error) {
	if in.PreviewID == "" || in.FileHash == "" {
		return CreateImportResult{}, BadRequest("previewId と fileHash は必須です", nil)
	}

	previewStore.RLock()
	stored, ok := previewStore.items[in.PreviewID]
	previewStore.RUnlock()
	if !ok {
		return CreateImportResult{}, Validation("プレビューが見つかりません。再度CSVを選択してください", nil)
	}
	if stored.Preview.FileHash != in.FileHash {
		return CreateImportResult{}, Conflict("プレビューとfileHashが一致しません", nil)
	}
	if existing, err := a.repos.Imports().FindByHash(ctx, in.FileHash); err != nil {
		return CreateImportResult{}, err
	} else if existing != nil {
		return CreateImportResult{}, Conflict("同一ファイルは既にインポート済みです", map[string]any{"fileHash": in.FileHash})
	}

	mapping := confirmedMapping(stored.Preview.MappingCandidates, in.ConfirmedMapping)
	if missing := missingRequired(mapping); len(missing) > 0 {
		return CreateImportResult{}, Validation("必須項目のマッピングが不足しています", map[string]any{"missing": missing})
	}

	var result CreateImportResult
	err := a.tx.WithinTx(ctx, func(ctx context.Context, repos TxRepositories) error {
		file := domain.ImportFile{
			FileName:       stored.Preview.FileName,
			FileHash:       stored.Preview.FileHash,
			DetectedFormat: stored.Preview.DetectedFormat,
			HasHeader:      stored.Preview.HasHeader,
			RowCount:       len(stored.Rows),
			ImportedAt:     time.Now(),
		}
		mappings := toDomainMappings(mapping)
		errs := toDomainErrors(stored.Preview.Errors)
		createdFile, err := repos.Imports().CreateImport(ctx, file, mappings, errs)
		if err != nil {
			return err
		}
		txs, rowErrors := rowsToTransactions(createdFile.ID, stored.Rows, mapping)
		if len(rowErrors) > 0 {
			errs = append(errs, rowErrors...)
		}
		created, skipped, err := repos.Transactions().CreateMany(ctx, txs)
		if err != nil {
			return err
		}
		result = CreateImportResult{
			ImportFile:            createdFile,
			ImportedCount:         created,
			DuplicateSkippedCount: skipped,
			ErrorCount:            len(errs),
		}
		return nil
	})
	if err != nil {
		return CreateImportResult{}, err
	}

	previewStore.Lock()
	delete(previewStore.items, in.PreviewID)
	previewStore.Unlock()

	return result, nil
}

func (a *App) ListImports(ctx context.Context, page, pageSize int) ([]domain.ImportFile, Pagination, error) {
	page, pageSize = normalizePage(page, pageSize)
	items, total, err := a.repos.Imports().List(ctx, page, pageSize)
	if err != nil {
		return nil, Pagination{}, err
	}
	return items, pagination(page, pageSize, total), nil
}

func (a *App) GetImport(ctx context.Context, id int64) (*domain.ImportFile, error) {
	return a.repos.Imports().FindByID(ctx, id)
}

func (a *App) DeleteImport(ctx context.Context, id int64) error {
	return a.tx.WithinTx(ctx, func(ctx context.Context, repos TxRepositories) error {
		if _, err := repos.Imports().FindByID(ctx, id); err != nil {
			return err
		}
		if err := repos.Transactions().DeleteByImportID(ctx, id); err != nil {
			return err
		}
		return repos.Imports().Delete(ctx, id)
	})
}

func decodeCSV(data []byte) (string, string, error) {
	if utf8.Valid(data) {
		return string(data), "UTF-8", nil
	}
	reader := transform.NewReader(bytes.NewReader(data), japanese.ShiftJIS.NewDecoder())
	decoded, err := io.ReadAll(reader)
	if err != nil {
		return "", "", err
	}
	return string(decoded), "CP932/Shift_JIS", nil
}

func detectHeader(row []string) bool {
	joined := strings.Join(row, ",")
	headerHints := []string{"利用日", "利用先", "支払", "請求", "金額", "カード"}
	for _, h := range headerHints {
		if strings.Contains(joined, h) {
			return true
		}
	}
	return false
}

func inferMapping(header []string, rows [][]string) []ImportMappingCandidate {
	width := 0
	if len(header) > 0 {
		width = len(header)
	}
	for _, row := range rows {
		if len(row) > width {
			width = len(row)
		}
	}
	candidates := make([]ImportMappingCandidate, 0, width)
	compact := isCompactVpassRows(rows)
	for i := 0; i < width; i++ {
		name := fmt.Sprintf("列 %d", i+1)
		if i < len(header) && strings.TrimSpace(header[i]) != "" {
			name = strings.TrimSpace(header[i])
		}
		target := inferTarget(name, i, compact)
		candidates = append(candidates, ImportMappingCandidate{
			SourceColumnName:  name,
			SourceColumnIndex: i,
			TargetField:       target,
			SampleValues:      sampleValues(rows, i),
			Required:          isRequiredTarget(target),
		})
	}
	return candidates
}

func inferTarget(name string, index int, compact bool) string {
	n := strings.ToLower(name)
	switch {
	case strings.Contains(name, "利用日") || strings.Contains(n, "date"):
		return "usageDate"
	case strings.Contains(name, "利用先") || strings.Contains(name, "店") || strings.Contains(n, "merchant"):
		return "merchantName"
	case strings.Contains(name, "支払区分") || strings.Contains(name, "支払方法"):
		return "paymentMethod"
	case strings.Contains(name, "支払") || strings.Contains(name, "請求月"):
		return "billingMonth"
	case strings.Contains(name, "利用金額"):
		return "usageAmount"
	case strings.Contains(name, "請求金額") || strings.Contains(name, "支払金額"):
		return "billedAmount"
	case strings.Contains(name, "本人") || strings.Contains(name, "家族") || strings.Contains(name, "利用者"):
		return "cardUser"
	}
	if compact {
		known := map[int]string{0: "usageDate", 1: "merchantName", 2: "usageAmount", 3: "paymentMethod", 5: "billedAmount", 7: "billingMonth"}
		if target, ok := known[index]; ok {
			return target
		}
		return ""
	}
	known := map[int]string{0: "usageDate", 1: "merchantName", 2: "cardUser", 3: "paymentMethod", 5: "billingMonth", 6: "usageAmount", 7: "billedAmount"}
	if target, ok := known[index]; ok {
		return target
	}
	return ""
}

func dropLeadingNonTransactionRows(rows [][]string) [][]string {
	for i, row := range rows {
		if len(row) == 0 {
			continue
		}
		if _, err := parseDate(row[0]); err == nil {
			return rows[i:]
		}
	}
	return rows
}

func billingMonthFromFileName(fileName string) string {
	base := fileName
	if idx := strings.LastIndexAny(base, `/\`); idx >= 0 {
		base = base[idx+1:]
	}
	digits := ""
	for _, r := range base {
		if r >= '0' && r <= '9' {
			digits += string(r)
		}
		if len(digits) >= 6 {
			break
		}
	}
	if len(digits) < 6 {
		return ""
	}
	return normalizeMonth(digits[:4] + "-" + digits[4:6])
}

func appendColumn(rows [][]string, value string) [][]string {
	out := make([][]string, 0, len(rows))
	for _, row := range rows {
		next := append([]string{}, row...)
		next = append(next, value)
		out = append(out, next)
	}
	return out
}

func hasMappingTarget(candidates []ImportMappingCandidate, target string) bool {
	for _, candidate := range candidates {
		if candidate.TargetField == target {
			return true
		}
	}
	return false
}

func isCompactVpassRows(rows [][]string) bool {
	for _, row := range rows {
		if len(row) < 6 || len(row) > 8 {
			continue
		}
		if _, err := parseDate(row[0]); err != nil {
			continue
		}
		if parseOptionalAmount(row[2]) != nil && parseOptionalAmount(row[5]) != nil {
			return true
		}
	}
	return false
}

func sampleValues(rows [][]string, index int) []string {
	values := []string{}
	for _, row := range rows {
		if index < len(row) && strings.TrimSpace(row[index]) != "" {
			values = append(values, row[index])
		}
		if len(values) >= 3 {
			break
		}
	}
	return values
}

func isRequiredTarget(target string) bool {
	return target == "usageDate" || target == "merchantName" || target == "billingMonth" || target == "usageAmount" || target == "billedAmount"
}

func validateRows(rows [][]string, candidates []ImportMappingCandidate) []ImportRowError {
	mapping := confirmedMapping(candidates, nil)
	if missing := missingRequired(mapping); len(missing) > 0 {
		return []ImportRowError{{RowNumber: 0, ErrorType: "MAPPING_REQUIRED", Message: "必須項目のマッピングが不足しています"}}
	}
	_, errs := rowsToTransactions(0, rows, mapping)
	out := make([]ImportRowError, 0, len(errs))
	for _, e := range errs {
		out = append(out, ImportRowError{RowNumber: e.RowNumber, ErrorType: e.ErrorType, Message: e.Message, RawColumns: e.RawColumns})
	}
	return out
}

func buildPreviewRows(rows [][]string, candidates []ImportMappingCandidate, limit int) []ImportPreviewRow {
	mapping := confirmedMapping(candidates, nil)
	out := []ImportPreviewRow{}
	for i, row := range rows {
		if i >= limit {
			break
		}
		out = append(out, ImportPreviewRow{RowNumber: i + 1, Normalized: normalizeRow(row, mapping), RawColumns: row})
	}
	return out
}

func confirmedMapping(candidates []ImportMappingCandidate, overrides map[string]string) map[int]string {
	mapping := map[int]string{}
	for _, c := range candidates {
		if c.TargetField != "" {
			mapping[c.SourceColumnIndex] = c.TargetField
		}
	}
	for k, v := range overrides {
		idx, err := strconv.Atoi(k)
		if err == nil {
			mapping[idx] = v
		}
	}
	return mapping
}

func missingRequired(mapping map[int]string) []string {
	found := map[string]bool{}
	for _, target := range mapping {
		found[target] = true
	}
	missing := []string{}
	for _, req := range []string{"usageDate", "merchantName", "billingMonth"} {
		if !found[req] {
			missing = append(missing, req)
		}
	}
	if !found["usageAmount"] && !found["billedAmount"] {
		missing = append(missing, "usageAmount or billedAmount")
	}
	return missing
}

func toDomainMappings(mapping map[int]string) []domain.ImportMapping {
	out := []domain.ImportMapping{}
	for idx, target := range mapping {
		if target == "" {
			continue
		}
		out = append(out, domain.ImportMapping{SourceColumnIndex: idx, SourceColumnName: fmt.Sprintf("列 %d", idx+1), TargetField: target, Confidence: 1})
	}
	return out
}

func toDomainErrors(errs []ImportRowError) []domain.ImportError {
	out := make([]domain.ImportError, 0, len(errs))
	for _, e := range errs {
		out = append(out, domain.ImportError{RowNumber: e.RowNumber, ErrorType: e.ErrorType, Message: e.Message, RawColumns: e.RawColumns})
	}
	return out
}

func rowsToTransactions(importID int64, rows [][]string, mapping map[int]string) ([]domain.Transaction, []domain.ImportError) {
	txs := []domain.Transaction{}
	errs := []domain.ImportError{}
	for i, row := range rows {
		normalized := normalizeRow(row, mapping)
		tx, err := normalizedToTransaction(importID, row, normalized)
		if err != nil {
			errs = append(errs, domain.ImportError{SourceFileID: importID, RowNumber: i + 1, ErrorType: "ROW_VALIDATION", Message: err.Error(), RawColumns: row})
			continue
		}
		txs = append(txs, tx)
	}
	return txs, errs
}

func normalizeRow(row []string, mapping map[int]string) map[string]any {
	out := map[string]any{}
	for idx, target := range mapping {
		if idx < len(row) && target != "" {
			out[target] = strings.TrimSpace(row[idx])
		}
	}
	return out
}

func normalizedToTransaction(importID int64, raw []string, normalized map[string]any) (domain.Transaction, error) {
	usageDate, err := parseDate(fmt.Sprint(normalized["usageDate"]))
	if err != nil {
		return domain.Transaction{}, fmt.Errorf("利用日を変換できません")
	}
	merchant := strings.TrimSpace(fmt.Sprint(normalized["merchantName"]))
	if merchant == "" {
		return domain.Transaction{}, fmt.Errorf("利用先が空です")
	}
	billingMonth := normalizeMonth(fmt.Sprint(normalized["billingMonth"]))
	if billingMonth == "" {
		return domain.Transaction{}, fmt.Errorf("請求月を変換できません")
	}
	usageAmount := parseOptionalAmount(fmt.Sprint(normalized["usageAmount"]))
	billedAmount := parseOptionalAmount(fmt.Sprint(normalized["billedAmount"]))
	if usageAmount == nil && billedAmount == nil {
		return domain.Transaction{}, fmt.Errorf("利用金額または請求金額が必要です")
	}
	rawJSON, _ := json.Marshal(raw)
	dedupe := fmt.Sprintf("%s|%s|%s|%s|%s|%s|%s", usageDate.Format("2006-01-02"), merchant, normalized["cardUser"], normalized["paymentMethod"], billingMonth, intPtrString(usageAmount), intPtrString(billedAmount))
	hash := sha256.Sum256([]byte(dedupe))
	return domain.Transaction{
		SourceFileID:  importID,
		UsageDate:     usageDate,
		MerchantName:  merchant,
		CardUser:      fmt.Sprint(normalized["cardUser"]),
		PaymentMethod: fmt.Sprint(normalized["paymentMethod"]),
		BillingMonth:  billingMonth,
		UsageAmount:   usageAmount,
		BilledAmount:  billedAmount,
		RawColumns:    []string{string(rawJSON)},
		DedupeKey:     hex.EncodeToString(hash[:]),
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}, nil
}

func parseDate(s string) (time.Time, error) {
	s = strings.TrimSpace(strings.ReplaceAll(s, "/", "-"))
	for _, layout := range []string{"2006-01-02", "2006-1-2", "2006年1月2日"} {
		if t, err := time.ParseInLocation(layout, s, time.Local); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("invalid date")
}

func normalizeMonth(s string) string {
	s = strings.TrimSpace(strings.ReplaceAll(strings.TrimPrefix(s, "'"), "/", "-"))
	if strings.Count(s, "-") == 1 {
		parts := strings.Split(s, "-")
		if len(parts) == 2 && len(parts[0]) == 2 {
			s = "20" + parts[0] + "-" + parts[1]
		}
	}
	if t, err := time.Parse("2006-01", s); err == nil {
		return t.Format("2006-01")
	}
	if t, err := time.Parse("2006-1", s); err == nil {
		return t.Format("2006-01")
	}
	if t, err := parseDate(s); err == nil {
		return t.Format("2006-01")
	}
	return ""
}

func parseOptionalAmount(s string) *int64 {
	s = strings.TrimSpace(strings.ReplaceAll(strings.ReplaceAll(s, ",", ""), "円", ""))
	if s == "" || s == "<nil>" {
		return nil
	}
	v, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return nil
	}
	return &v
}

func intPtrString(v *int64) string {
	if v == nil {
		return ""
	}
	return strconv.FormatInt(*v, 10)
}
