package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	catalogclient "github.com/code-corhuila/lms-circulation-api/internal/infrastructure/catalog"
	membershipclient "github.com/code-corhuila/lms-circulation-api/internal/infrastructure/membership"

	"github.com/code-corhuila/lms-circulation-api/internal/application/usecase"
	"github.com/code-corhuila/lms-circulation-api/internal/domain/circulation"
	"github.com/code-corhuila/lms-circulation-api/internal/domain/service"
	"github.com/code-corhuila/lms-circulation-api/internal/infrastructure/http/middleware"
	"github.com/code-corhuila/lms-circulation-api/internal/infrastructure/http/response"
)

const loanTimeFormat = "2006-01-02T15:04:05Z07:00"

// LoanHandler implements the /loans endpoints (HU-06 register, HU-07
// return/history, HU-08 overdue report + suspension trigger).
type LoanHandler struct {
	registerLoan *usecase.RegisterLoan
	returnLoan   *usecase.ReturnLoan
	searchLoans  *usecase.SearchLoans
	overdueLoans *usecase.OverdueLoans
}

func NewLoanHandler(registerLoan *usecase.RegisterLoan, returnLoan *usecase.ReturnLoan, searchLoans *usecase.SearchLoans, overdueLoans *usecase.OverdueLoans) *LoanHandler {
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

// Create — POST /loans (HU-06, FR-013, FR-014, FR-015).
func (h *LoanHandler) Create(w http.ResponseWriter, r *http.Request) {
	correlationID := middleware.FromContext(r.Context())

	var req createLoanRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request body", correlationID)
		return
	}

	loan, err := h.registerLoan.Execute(r.Context(), req.StudentID, req.BookID)
	switch {
	case errors.Is(err, service.ErrStudentSuspended):
		response.Error(w, http.StatusConflict, "STUDENT_SUSPENDED",
			"This student cannot receive a new loan until their suspension ends", correlationID)
		return
	case errors.Is(err, service.ErrLoanLimitReached):
		response.Error(w, http.StatusConflict, "LOAN_LIMIT_REACHED",
			"This student already has the maximum of 2 active loans", correlationID)
		return
	case errors.Is(err, catalogclient.ErrNoCopiesAvailable):
		response.Error(w, http.StatusConflict, "NO_COPIES_AVAILABLE",
			"There are no available copies of this book", correlationID)
		return
	case errors.Is(err, membershipclient.ErrStudentNotFound):
		response.Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "Student not found", correlationID)
		return
	case errors.Is(err, catalogclient.ErrBookNotFound):
		response.Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "Book not found", correlationID)
		return
	case err != nil:
		response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error", correlationID)
		return
	}

	response.JSON(w, http.StatusCreated, toLoanResponse(loan))
}

// Return — POST /loans/{id}/return (HU-07, FR-016, FR-017, FR-018; triggers
// HU-08's suspension policy when the return is late — FR-019).
func (h *LoanHandler) Return(w http.ResponseWriter, r *http.Request) {
	correlationID := middleware.FromContext(r.Context())
	id := chi.URLParam(r, "id")

	loan, err := h.returnLoan.Execute(r.Context(), id)
	switch {
	case errors.Is(err, circulation.ErrLoanNotFound):
		response.Error(w, http.StatusNotFound, "NOT_FOUND", "Loan not found", correlationID)
		return
	case errors.Is(err, circulation.ErrLoanAlreadyReturned):
		response.Error(w, http.StatusConflict, "LOAN_ALREADY_RETURNED", "This loan was already returned", correlationID)
		return
	case err != nil:
		response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error", correlationID)
		return
	}

	response.JSON(w, http.StatusOK, toLoanResponse(loan))
}

// List — GET /loans (HU-07 history; also supports ?overdue=true).
func (h *LoanHandler) List(w http.ResponseWriter, r *http.Request) {
	correlationID := middleware.FromContext(r.Context())

	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	status := r.URL.Query().Get("status")
	overdue := r.URL.Query().Get("overdue") == "true"
	studentID := r.URL.Query().Get("studentId")
	bookID := r.URL.Query().Get("bookId")

	loans, total, err := h.searchLoans.Execute(r.Context(), status, overdue, studentID, bookID, page, limit)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error", correlationID)
		return
	}

	items := make([]loanResponse, 0, len(loans))
	for _, l := range loans {
		items = append(items, toLoanResponse(l))
	}

	response.JSON(w, http.StatusOK, map[string]any{
		"data": items,
		"meta": map[string]any{"total": total},
	})
}

// Overdue — GET /loans/overdue (HU-08, FR-020, Scenario 2).
func (h *LoanHandler) Overdue(w http.ResponseWriter, r *http.Request) {
	correlationID := middleware.FromContext(r.Context())

	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))

	loans, total, err := h.overdueLoans.Execute(r.Context(), page, limit)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error", correlationID)
		return
	}

	items := make([]loanResponse, 0, len(loans))
	for _, l := range loans {
		items = append(items, toLoanResponse(l))
	}

	response.JSON(w, http.StatusOK, map[string]any{
		"data": items,
		"meta": map[string]any{"total": total},
	})
}
