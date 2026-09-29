package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	catalogclient "github.com/code-corhuila/lms-circulation-api/internal/adapter/out/catalogclient"
	membershipclient "github.com/code-corhuila/lms-circulation-api/internal/adapter/out/membershipclient"

	in "github.com/code-corhuila/lms-circulation-api/internal/application/port/in"
	"github.com/code-corhuila/lms-circulation-api/internal/application/service"
	"github.com/code-corhuila/lms-circulation-api/internal/domain/circulation"
	"github.com/code-corhuila/lms-circulation-api/internal/adapter/in/httpapi/middleware"
	"github.com/code-corhuila/lms-circulation-api/internal/adapter/in/httpapi/response"
)

const loanTimeFormat = "2006-01-02T15:04:05Z07:00"

// LoanHandler implements the /loans endpoints (HU-06 register, HU-07
// return/history, HU-08 overdue report + suspension trigger). It depends on
// application/port/in interfaces, not the concrete usecase.X structs
// (rules/2-anexos/C-api-hexagonal.md).
type LoanHandler struct {
	registerLoan in.RegisterLoanUseCase
	returnLoan   in.ReturnLoanUseCase
	searchLoans  in.SearchLoansUseCase
	overdueLoans in.OverdueLoansUseCase
}

func NewLoanHandler(registerLoan in.RegisterLoanUseCase, returnLoan in.ReturnLoanUseCase, searchLoans in.SearchLoansUseCase, overdueLoans in.OverdueLoansUseCase) *LoanHandler {
	return &LoanHandler{registerLoan: registerLoan, returnLoan: returnLoan, searchLoans: searchLoans, overdueLoans: overdueLoans}
}

type createLoanRequest struct {
	StudentID string `json:"studentId"`
	BookID    string `json:"bookId"`
}

type loanResponse struct {
	ID         string  `json:"id"`
	StudentID  string  `json:"studentId"`
	BookID     string  `json:"bookId"`
	LoanDate   string  `json:"loanDate"`
	DueDate    string  `json:"dueDate"`
	ReturnDate *string `json:"returnDate,omitempty"`
	Status     string  `json:"status"`
	WasLate    *bool   `json:"wasLate,omitempty"`
}

func toLoanResponse(l *circulation.Loan) loanResponse {
	resp := loanResponse{
		ID:        l.ID,
		StudentID: l.StudentID,
		BookID:    l.BookID,
		LoanDate:  l.LoanDate.Format(loanTimeFormat),
		DueDate:   l.DueDate.Format(loanTimeFormat),
		Status:    l.Status,
		WasLate:   l.WasLate,
	}
	if l.ReturnDate != nil {
		v := l.ReturnDate.Format(loanTimeFormat)
		resp.ReturnDate = &v
	}
	return resp
}

// clampPage and clampLimit mirror the same bounds the search/overdue use
// cases apply server-side, so List/Overdue can echo the values actually
// served — see rules/2-anexos/C-api-hexagonal.md, "Listados".
func clampPage(page int) int {
	if page < 1 {
		return 1
	}
	return page
}

func clampLimit(limit int) int {
	if limit < 1 || limit > 100 {
		return 20
	}
	return limit
}

func paginatedMeta(total, page, limit int) map[string]any {
	totalPages := 0
	if total > 0 {
		totalPages = (total + limit - 1) / limit
	}
	return map[string]any{
		"total":      total,
		"page":       page,
		"limit":      limit,
		"totalPages": totalPages,
	}
}

// Create — POST /loans (HU-06, FR-013, FR-014, FR-015). Idempotent by the
// Idempotency-Key header (rules/2-anexos/C-api-hexagonal.md, numeral 5.3.8): a
// retried request with the same key returns the original loan and 200, not a
// second loan and 201 (and does not call catalog-service's loan-copy again).
func (h *LoanHandler) Create(w http.ResponseWriter, r *http.Request) {
	traceID := middleware.FromContext(r.Context())
	idempotencyKey := r.Header.Get("Idempotency-Key")

	var req createLoanRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request body", traceID)
		return
	}

	// STUDENT_SUSPENDED, LOAN_LIMIT_REACHED, and NO_COPIES_AVAILABLE are all
	// 422, not 409: the request is well-formed and collides with no existing
	// resource — a domain rule (INV-002/INV-003/INV-004) forbids it outright
	// (rules/2-anexos/C-api-hexagonal.md, numeral 5.3.11 / D-G08).
	loan, replayed, err := h.registerLoan.Execute(r.Context(), req.StudentID, req.BookID, idempotencyKey)
	switch {
	case errors.Is(err, service.ErrStudentSuspended):
		response.Error(w, http.StatusUnprocessableEntity, "STUDENT_SUSPENDED",
			"This student cannot receive a new loan until their suspension ends", traceID)
		return
	case errors.Is(err, service.ErrLoanLimitReached):
		response.Error(w, http.StatusUnprocessableEntity, "LOAN_LIMIT_REACHED",
			"This student already has the maximum of 2 active loans", traceID)
		return
	case errors.Is(err, catalogclient.ErrNoCopiesAvailable):
		response.Error(w, http.StatusUnprocessableEntity, "NO_COPIES_AVAILABLE",
			"There are no available copies of this book", traceID)
		return
	case errors.Is(err, membershipclient.ErrStudentNotFound):
		response.ValidationError(w, traceID, response.FieldDetail{Field: "studentId", Message: "student not found"})
		return
	case errors.Is(err, catalogclient.ErrBookNotFound):
		response.ValidationError(w, traceID, response.FieldDetail{Field: "bookId", Message: "book not found"})
		return
	case err != nil:
		response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error", traceID)
		return
	}

	if replayed {
		response.JSON(w, http.StatusOK, toLoanResponse(loan))
		return
	}

	w.Header().Set("Location", fmt.Sprintf("/api/v1/loans/%s", loan.ID))
	response.JSON(w, http.StatusCreated, toLoanResponse(loan))
}

// Return — POST /loans/{id}/return (HU-07, FR-016, FR-017, FR-018; triggers
// HU-08's suspension policy when the return is late — FR-019).
func (h *LoanHandler) Return(w http.ResponseWriter, r *http.Request) {
	traceID := middleware.FromContext(r.Context())
	id := chi.URLParam(r, "id")

	loan, err := h.returnLoan.Execute(r.Context(), id)
	switch {
	case errors.Is(err, circulation.ErrLoanNotFound):
		response.Error(w, http.StatusNotFound, "NOT_FOUND", "Loan not found", traceID)
		return
	case errors.Is(err, circulation.ErrLoanAlreadyReturned):
		response.Error(w, http.StatusConflict, "LOAN_ALREADY_RETURNED", "This loan was already returned", traceID)
		return
	case err != nil:
		response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error", traceID)
		return
	}

	response.JSON(w, http.StatusOK, toLoanResponse(loan))
}

// List — GET /loans (HU-07 history; also supports ?overdue=true). meta
// carries the full {total, page, limit, totalPages} envelope.
func (h *LoanHandler) List(w http.ResponseWriter, r *http.Request) {
	traceID := middleware.FromContext(r.Context())

	rawPage, _ := strconv.Atoi(r.URL.Query().Get("page"))
	rawLimit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	status := r.URL.Query().Get("status")
	overdue := r.URL.Query().Get("overdue") == "true"
	studentID := r.URL.Query().Get("studentId")
	bookID := r.URL.Query().Get("bookId")

	page := clampPage(rawPage)
	limit := clampLimit(rawLimit)

	loans, total, err := h.searchLoans.Execute(r.Context(), status, overdue, studentID, bookID, page, limit)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error", traceID)
		return
	}

	items := make([]loanResponse, 0, len(loans))
	for _, l := range loans {
		items = append(items, toLoanResponse(l))
	}

	response.JSON(w, http.StatusOK, map[string]any{
		"data": items,
		"meta": paginatedMeta(total, page, limit),
	})
}

// Overdue — GET /loans/overdue (HU-08, FR-020, Scenario 2). meta carries the
// full {total, page, limit, totalPages} envelope.
func (h *LoanHandler) Overdue(w http.ResponseWriter, r *http.Request) {
	traceID := middleware.FromContext(r.Context())

	rawPage, _ := strconv.Atoi(r.URL.Query().Get("page"))
	rawLimit, _ := strconv.Atoi(r.URL.Query().Get("limit"))

	page := clampPage(rawPage)
	limit := clampLimit(rawLimit)

	loans, total, err := h.overdueLoans.Execute(r.Context(), page, limit)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error", traceID)
		return
	}

	items := make([]loanResponse, 0, len(loans))
	for _, l := range loans {
		items = append(items, toLoanResponse(l))
	}

	response.JSON(w, http.StatusOK, map[string]any{
		"data": items,
		"meta": paginatedMeta(total, page, limit),
	})
}
