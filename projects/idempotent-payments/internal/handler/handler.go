package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"tour_of_go/projects/idempotent-payments/internal/ports"
	"tour_of_go/projects/idempotent-payments/internal/service"
)

type PaymentRequest struct {
	FromAccountID int64   `json:"from_account_id"`
	ToAccountID   int64   `json:"to_account_id"`
	Amount        float64 `json:"amount"`
}

type PaymentResponse struct {
	PaymentID string  `json:"payment_id"`
	Status    string  `json:"status"`
	Amount    float64 `json:"amount"`
}

func Payment(svc *service.LedgerService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req PaymentRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}

		key := r.Header.Get("Idempotency-Key")
		p, err := svc.ProcessPayment(r.Context(), req.FromAccountID, req.ToAccountID, req.Amount, key)
		if err != nil {
			msg := err.Error()
			if strings.Contains(msg, "insufficient funds") {
				http.Error(w, msg, http.StatusPaymentRequired)
				return
			}
			http.Error(w, msg, http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(PaymentResponse{ //nolint:errcheck
			PaymentID: p.ID,
			Status:    p.Status,
			Amount:    p.Amount,
		})
	}
}

func GetAccount(repo ports.LedgerDB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idStr := chi.URLParam(r, "id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			http.Error(w, "invalid account id", http.StatusBadRequest)
			return
		}
		acc, err := repo.GetAccount(r.Context(), id)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(acc) //nolint:errcheck
	}
}
