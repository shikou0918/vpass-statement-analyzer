package httpadapter

import (
	"encoding/csv"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"vpass-statement-analyzer/backend/internal/domain"
	"vpass-statement-analyzer/backend/internal/usecase"
)

type Handler struct {
	app           *usecase.App
	allowedOrigin string
}

func NewRouter(app *usecase.App, allowedOrigin string) http.Handler {
	h := &Handler{app: app, allowedOrigin: allowedOrigin}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /import-previews", h.createImportPreview)
	mux.HandleFunc("GET /imports", h.listImports)
	mux.HandleFunc("POST /imports", h.createImport)
	mux.HandleFunc("GET /imports/", h.getImport)
	mux.HandleFunc("DELETE /imports/", h.deleteImport)
	mux.HandleFunc("GET /transactions", h.listTransactions)
	mux.HandleFunc("GET /transactions/", h.getTransaction)
	mux.HandleFunc("PATCH /transactions/", h.updateTransaction)
	mux.HandleFunc("GET /summaries/monthly", h.getMonthlySummary)
	mux.HandleFunc("GET /summaries/merchants", h.getMerchantSummary)
	mux.HandleFunc("GET /summaries/categories", h.getCategorySummary)
	mux.HandleFunc("GET /analytics/monthly-trends", h.getMonthlyTrends)
	mux.HandleFunc("GET /analytics/merchant-trends", h.getMerchantTrends)
	mux.HandleFunc("GET /analytics/category-trends", h.getCategoryTrends)
	mux.HandleFunc("GET /analytics/recurring-candidates", h.getRecurringCandidates)
	mux.HandleFunc("GET /analytics/small-frequent-transactions", h.getSmallFrequentTransactions)
	mux.HandleFunc("GET /categories", h.listCategories)
	mux.HandleFunc("POST /categories", h.createCategory)
	mux.HandleFunc("PATCH /categories/", h.updateCategory)
	mux.HandleFunc("DELETE /categories/", h.deleteCategory)
	mux.HandleFunc("GET /category-rules", h.listCategoryRules)
	mux.HandleFunc("POST /category-rules", h.createCategoryRule)
	mux.HandleFunc("PATCH /category-rules/", h.updateCategoryRule)
	mux.HandleFunc("DELETE /category-rules/", h.deleteCategoryRule)
	mux.HandleFunc("POST /category-rule-applications", h.applyCategoryRules)
	mux.HandleFunc("GET /exports/transactions", h.exportTransactions)
	mux.HandleFunc("GET /exports/categories", h.exportCategories)
	mux.HandleFunc("GET /exports/category-rules", h.exportCategoryRules)
	mux.HandleFunc("GET /settings", h.getSettings)
	mux.HandleFunc("PATCH /settings", h.updateSettings)
	return h.cors(mux)
}

func (h *Handler) cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", h.allowedOrigin)
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.Header().Set("Access-Control-Allow-Methods", "GET,POST,PATCH,DELETE,OPTIONS")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (h *Handler) createImportPreview(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		writeError(w, usecase.BadRequest("multipart/form-data を解析できません", nil))
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		writeError(w, usecase.BadRequest("file は必須です", map[string]any{"field": "file"}))
		return
	}
	defer file.Close()
	preview, err := h.app.CreateImportPreview(r.Context(), header.Filename, file)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, preview)
}

func (h *Handler) createImport(w http.ResponseWriter, r *http.Request) {
	var in usecase.CreateImportInput
	if !decodeJSON(w, r, &in) {
		return
	}
	result, err := h.app.CreateImport(r.Context(), in)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, createImportToResponse(result))
}

func (h *Handler) listImports(w http.ResponseWriter, r *http.Request) {
	items, p, err := h.app.ListImports(r.Context(), intQuery(r, "page", 1), intQuery(r, "pageSize", 50))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": mapResponses(items, importFileToResponse), "pagination": p})
}

func (h *Handler) getImport(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "/imports/")
	if !ok {
		return
	}
	item, err := h.app.GetImport(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, importFileToResponse(*item))
}

func (h *Handler) deleteImport(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "/imports/")
	if !ok {
		return
	}
	if err := h.app.DeleteImport(r.Context(), id); err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) listTransactions(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	filter := usecase.TransactionFilter{
		BillingMonth:    q.Get("billingMonth"),
		UsageDateFrom:   q.Get("usageDateFrom"),
		UsageDateTo:     q.Get("usageDateTo"),
		MerchantName:    q.Get("merchantName"),
		CategoryID:      q.Get("categoryId"),
		Keyword:         q.Get("keyword"),
		IncludeExcluded: q.Get("includeExcluded") == "true",
		Page:            intQuery(r, "page", 1),
		PageSize:        intQuery(r, "pageSize", 50),
		Sort:            q.Get("sort"),
		Order:           q.Get("order"),
	}
	if v := int64PtrQuery(r, "amountMin"); v != nil {
		filter.AmountMin = v
	}
	if v := int64PtrQuery(r, "amountMax"); v != nil {
		filter.AmountMax = v
	}
	items, p, err := h.app.ListTransactions(r.Context(), filter)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": mapResponses(items, transactionToResponse), "pagination": p})
}

func (h *Handler) getTransaction(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "/transactions/")
	if !ok {
		return
	}
	item, err := h.app.GetTransaction(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, transactionToResponse(*item))
}

func (h *Handler) updateTransaction(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "/transactions/")
	if !ok {
		return
	}
	var body struct {
		CategoryID            *string `json:"categoryId"`
		Memo                  *string `json:"memo"`
		ExcludedFromAnalytics *bool   `json:"excludedFromAnalytics"`
	}
	if !decodeJSON(w, r, &body) {
		return
	}
	categoryID, set, err := usecase.ParseOptionalID(body.CategoryID)
	if err != nil {
		writeError(w, err)
		return
	}
	item, err := h.app.UpdateTransaction(r.Context(), id, usecase.UpdateTransactionInput{CategoryID: categoryID, CategoryIDSet: set, Memo: body.Memo, ExcludedFromAnalytics: body.ExcludedFromAnalytics})
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, transactionToResponse(*item))
}

func (h *Handler) getMonthlySummary(w http.ResponseWriter, r *http.Request) {
	result, err := h.app.MonthlySummary(r.Context(), summaryFilter(r))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) getMerchantSummary(w http.ResponseWriter, r *http.Request) {
	filter := summaryFilter(r)
	filter.Limit = intQuery(r, "limit", 20)
	result, err := h.app.MerchantSummary(r.Context(), filter)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) getCategorySummary(w http.ResponseWriter, r *http.Request) {
	result, err := h.app.CategorySummary(r.Context(), summaryFilter(r))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) getMonthlyTrends(w http.ResponseWriter, r *http.Request) {
	result, err := h.app.Trend(r.Context(), summaryFilter(r))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) getMerchantTrends(w http.ResponseWriter, r *http.Request) {
	filter := summaryFilter(r)
	filter.Merchant = r.URL.Query().Get("merchantName")
	if filter.Merchant == "" {
		writeError(w, usecase.BadRequest("merchantName は必須です", map[string]any{"field": "merchantName"}))
		return
	}
	result, err := h.app.Trend(r.Context(), filter)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) getCategoryTrends(w http.ResponseWriter, r *http.Request) {
	filter := summaryFilter(r)
	filter.CategoryID = r.URL.Query().Get("categoryId")
	result, err := h.app.Trend(r.Context(), filter)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) getRecurringCandidates(w http.ResponseWriter, r *http.Request) {
	items, err := h.app.RecurringCandidates(r.Context(), summaryFilter(r))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (h *Handler) getSmallFrequentTransactions(w http.ResponseWriter, r *http.Request) {
	filter := summaryFilter(r)
	filter.MaxAmount = int64(intQuery(r, "maxAmount", 1000))
	items, err := h.app.SmallFrequent(r.Context(), filter)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (h *Handler) listCategories(w http.ResponseWriter, r *http.Request) {
	items, err := h.app.ListCategories(r.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": mapResponses(items, categoryToResponse)})
}

func (h *Handler) createCategory(w http.ResponseWriter, r *http.Request) {
	var in usecase.CategoryInput
	if !decodeJSON(w, r, &in) {
		return
	}
	item, err := h.app.CreateCategory(r.Context(), in)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, categoryToResponse(item))
}

func (h *Handler) updateCategory(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "/categories/")
	if !ok {
		return
	}
	var in usecase.CategoryInput
	if !decodeJSON(w, r, &in) {
		return
	}
	item, err := h.app.UpdateCategory(r.Context(), id, in)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, categoryToResponse(*item))
}

func (h *Handler) deleteCategory(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "/categories/")
	if !ok {
		return
	}
	if err := h.app.DeleteCategory(r.Context(), id); err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) listCategoryRules(w http.ResponseWriter, r *http.Request) {
	items, err := h.app.ListCategoryRules(r.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": mapResponses(items, categoryRuleToResponse)})
}

func (h *Handler) createCategoryRule(w http.ResponseWriter, r *http.Request) {
	var in usecase.CategoryRuleInput
	if !decodeJSON(w, r, &in) {
		return
	}
	item, err := h.app.CreateCategoryRule(r.Context(), in)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, categoryRuleToResponse(item))
}

func (h *Handler) updateCategoryRule(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "/category-rules/")
	if !ok {
		return
	}
	var in usecase.CategoryRuleInput
	if !decodeJSON(w, r, &in) {
		return
	}
	item, err := h.app.UpdateCategoryRule(r.Context(), id, in)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, categoryRuleToResponse(*item))
}

func (h *Handler) deleteCategoryRule(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "/category-rules/")
	if !ok {
		return
	}
	if err := h.app.DeleteCategoryRule(r.Context(), id); err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) applyCategoryRules(w http.ResponseWriter, r *http.Request) {
	var body struct {
		OverwriteManualCategory bool `json:"overwriteManualCategory"`
	}
	if !decodeJSON(w, r, &body) {
		return
	}
	matched, updated, unchanged, uncategorized, err := h.app.ApplyCategoryRules(r.Context(), body.OverwriteManualCategory)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]int{"matchedCount": matched, "updatedCount": updated, "unchangedCount": unchanged, "uncategorizedCount": uncategorized})
}

func (h *Handler) exportTransactions(w http.ResponseWriter, r *http.Request) {
	items, _, err := h.app.ListTransactions(r.Context(), usecase.TransactionFilter{Page: 1, PageSize: 200, IncludeExcluded: true})
	if err != nil {
		writeError(w, err)
		return
	}
	if r.URL.Query().Get("format") == "json" {
		writeJSON(w, http.StatusOK, map[string]any{"items": mapResponses(items, transactionToResponse)})
		return
	}
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", "attachment; filename=\"transactions.csv\"")
	cw := csv.NewWriter(w)
	_ = cw.Write([]string{"id", "usageDate", "merchantName", "billingMonth", "usageAmount", "billedAmount", "categoryId", "memo"})
	for _, item := range items {
		_ = cw.Write([]string{strconv.FormatInt(item.ID, 10), item.UsageDate.Format("2006-01-02"), item.MerchantName, item.BillingMonth, intPtr(item.UsageAmount), intPtr(item.BilledAmount), intPtr(item.CategoryID), item.Memo})
	}
	cw.Flush()
}

func (h *Handler) exportCategories(w http.ResponseWriter, r *http.Request) {
	h.listCategories(w, r)
}

func (h *Handler) exportCategoryRules(w http.ResponseWriter, r *http.Request) {
	h.listCategoryRules(w, r)
}

func (h *Handler) getSettings(w http.ResponseWriter, r *http.Request) {
	item, err := h.app.GetSettings(r.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, settingsToResponse(item))
}

func (h *Handler) updateSettings(w http.ResponseWriter, r *http.Request) {
	var body domain.AppSettings
	if !decodeJSON(w, r, &body) {
		return
	}
	item, err := h.app.UpdateSettings(r.Context(), body)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, settingsToResponse(item))
}

func summaryFilter(r *http.Request) usecase.SummaryFilter {
	q := r.URL.Query()
	return usecase.SummaryFilter{
		Month:       q.Get("month"),
		From:        q.Get("from"),
		To:          q.Get("to"),
		FromMonth:   q.Get("fromMonth"),
		ToMonth:     q.Get("toMonth"),
		BasisDate:   valueOr(q.Get("basisDate"), "billingMonth"),
		BasisAmount: valueOr(q.Get("basisAmount"), "billedAmount"),
	}
}

func decodeJSON(w http.ResponseWriter, r *http.Request, v any) bool {
	defer r.Body.Close()
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		writeError(w, usecase.BadRequest("JSON形式が不正です", nil))
		return false
	}
	return true
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, err error) {
	status := http.StatusInternalServerError
	code := "INTERNAL_ERROR"
	message := "予期しないエラーが発生しました"
	details := map[string]any(nil)
	var appErr *usecase.AppError
	if errors.As(err, &appErr) {
		message = appErr.Message
		details = appErr.Details
		switch {
		case errors.Is(appErr.Kind, usecase.ErrBadRequest):
			status, code = http.StatusBadRequest, "BAD_REQUEST"
		case errors.Is(appErr.Kind, usecase.ErrNotFound):
			status, code = http.StatusNotFound, "NOT_FOUND"
		case errors.Is(appErr.Kind, usecase.ErrConflict):
			status, code = http.StatusConflict, "CONFLICT"
		case errors.Is(appErr.Kind, usecase.ErrValidation):
			status, code = http.StatusUnprocessableEntity, "VALIDATION_ERROR"
		}
	}
	writeJSON(w, status, map[string]any{"code": code, "message": message, "details": details})
}

func pathID(w http.ResponseWriter, r *http.Request, prefix string) (int64, bool) {
	idText := strings.Trim(strings.TrimPrefix(r.URL.Path, prefix), "/")
	id, err := strconv.ParseInt(idText, 10, 64)
	if err != nil || id <= 0 {
		writeError(w, usecase.BadRequest("ID形式が不正です", map[string]any{"id": idText}))
		return 0, false
	}
	return id, true
}

func intQuery(r *http.Request, key string, fallback int) int {
	v, err := strconv.Atoi(r.URL.Query().Get(key))
	if err != nil || v == 0 {
		return fallback
	}
	return v
}

func int64PtrQuery(r *http.Request, key string) *int64 {
	raw := r.URL.Query().Get(key)
	if raw == "" {
		return nil
	}
	v, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return nil
	}
	return &v
}

func intPtr(v *int64) string {
	if v == nil {
		return ""
	}
	return strconv.FormatInt(*v, 10)
}

func valueOr(v, fallback string) string {
	if v == "" {
		return fallback
	}
	return v
}
