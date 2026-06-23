package httpadapter

import (
	_ "embed"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"merchant-transactions-api/internal/app"
	"merchant-transactions-api/internal/apperrors"
	"merchant-transactions-api/internal/domain"
	"merchant-transactions-api/internal/ports"
)

//go:embed openapi.yaml
var openAPISpec []byte

type Handler struct {
	orchestrator    *app.Orchestrator
	transactionRepo ports.TransactionRepository
	receivableRepo  ports.ReceivableRepository
}

func NewHandler(orchestrator *app.Orchestrator, transactionRepo ports.TransactionRepository, receivableRepo ports.ReceivableRepository) *Handler {
	return &Handler{orchestrator: orchestrator, transactionRepo: transactionRepo, receivableRepo: receivableRepo}
}

func (h *Handler) NewMux() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", h.handleHealth)
	mux.HandleFunc("/transactions", h.handleTransactions)
	mux.HandleFunc("/transactions/", h.handleTransactionByID)
	mux.HandleFunc("/receivables", h.handleReceivables)
	mux.HandleFunc("/receivables/", h.handleReceivableByID)
	mux.HandleFunc("/openapi.yaml", h.serveOpenAPISpec)
	mux.HandleFunc("/docs", h.serveSwaggerUI)
	return mux
}

func (h *Handler) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	respondJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) handleTransactions(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		h.createTransaction(w, r)
	case http.MethodGet:
		h.listTransactions(w, r)
	default:
		respondError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (h *Handler) handleTransactionByID(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/transactions/")
	if id == "" {
		respondError(w, http.StatusNotFound, "not found")
		return
	}
	switch r.Method {
	case http.MethodGet:
		h.getTransaction(w, r, id)
	case http.MethodDelete:
		h.deleteTransaction(w, r, id)
	default:
		respondError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (h *Handler) handleReceivables(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.listReceivables(w, r)
	default:
		respondError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (h *Handler) handleReceivableByID(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/receivables/")
	if id == "" {
		respondError(w, http.StatusNotFound, "not found")
		return
	}
	switch r.Method {
	case http.MethodGet:
		h.getReceivable(w, r, id)
	case http.MethodDelete:
		h.deleteReceivable(w, r, id)
	default:
		respondError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (h *Handler) createTransaction(w http.ResponseWriter, r *http.Request) {
	var command domain.CreateTransactionCommand
	if err := json.NewDecoder(r.Body).Decode(&command); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request payload")
		return
	}
	transaction, receivable, err := h.orchestrator.CreateTransaction(r.Context(), command)
	if err != nil {
		// map known errors to appropriate HTTP status codes
		if errors.Is(err, domain.ErrInvalidTransactionValue) || errors.Is(err, domain.ErrInvalidPaymentMethod) || errors.Is(err, domain.ErrInvalidCardNumber) || errors.Is(err, domain.ErrInvalidCardExpirationDate) || errors.Is(err, domain.ErrInvalidCardCvv) || errors.Is(err, domain.ErrInvalidTransactionDescription) {
			respondError(w, http.StatusBadRequest, err.Error())
			return
		}
		if errors.Is(err, apperrors.ErrUnableToObtainUniqueNumerator) || errors.Is(err, apperrors.ErrNumeratorService) || errors.Is(err, apperrors.ErrJSONServerService) {
			respondError(w, http.StatusBadGateway, err.Error())
			return
		}
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusCreated, map[string]interface{}{
		"transaction": transaction,
		"receivable":  receivable,
	})
}

func (h *Handler) listTransactions(w http.ResponseWriter, r *http.Request) {
	transactions, err := h.transactionRepo.ListTransactions(r.Context())
	if err != nil {
		if errors.Is(err, apperrors.ErrJSONServerService) {
			respondError(w, http.StatusBadGateway, err.Error())
			return
		}
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, transactions)
}

func (h *Handler) getTransaction(w http.ResponseWriter, r *http.Request, id string) {
	transaction, err := h.transactionRepo.GetTransaction(r.Context(), id)
	if err != nil {
		if errors.Is(err, apperrors.ErrNotFound) {
			respondError(w, http.StatusNotFound, err.Error())
			return
		}
		if errors.Is(err, apperrors.ErrJSONServerService) {
			respondError(w, http.StatusBadGateway, err.Error())
			return
		}
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if transaction == nil {
		respondError(w, http.StatusNotFound, apperrors.ErrNotFound.Error())
		return
	}
	respondJSON(w, http.StatusOK, transaction)
}

func (h *Handler) deleteTransaction(w http.ResponseWriter, r *http.Request, id string) {
	if err := h.transactionRepo.DeleteTransaction(r.Context(), id); err != nil {
		if errors.Is(err, apperrors.ErrNotFound) {
			respondError(w, http.StatusNotFound, err.Error())
			return
		}
		if errors.Is(err, apperrors.ErrJSONServerService) {
			respondError(w, http.StatusBadGateway, err.Error())
			return
		}
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) listReceivables(w http.ResponseWriter, r *http.Request) {
	receivables, err := h.receivableRepo.ListReceivables(r.Context())
	if err != nil {
		if errors.Is(err, apperrors.ErrJSONServerService) {
			respondError(w, http.StatusBadGateway, err.Error())
			return
		}
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, receivables)
}

func (h *Handler) getReceivable(w http.ResponseWriter, r *http.Request, id string) {
	receivable, err := h.receivableRepo.GetReceivable(r.Context(), id)
	if err != nil {
		if errors.Is(err, apperrors.ErrNotFound) {
			respondError(w, http.StatusNotFound, err.Error())
			return
		}
		if errors.Is(err, apperrors.ErrJSONServerService) {
			respondError(w, http.StatusBadGateway, err.Error())
			return
		}
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if receivable == nil {
		respondError(w, http.StatusNotFound, apperrors.ErrNotFound.Error())
		return
	}
	respondJSON(w, http.StatusOK, receivable)
}

func (h *Handler) deleteReceivable(w http.ResponseWriter, r *http.Request, id string) {
	if err := h.receivableRepo.DeleteReceivable(r.Context(), id); err != nil {
		if errors.Is(err, apperrors.ErrNotFound) {
			respondError(w, http.StatusNotFound, err.Error())
			return
		}
		if errors.Is(err, apperrors.ErrJSONServerService) {
			respondError(w, http.StatusBadGateway, err.Error())
			return
		}
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) serveOpenAPISpec(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	w.Header().Set("Content-Type", "application/x-yaml")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(openAPISpec)
}

func (h *Handler) serveSwaggerUI(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1.0" />
  <title>Merchant Transactions API Docs</title>
  <link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@4/swagger-ui.css" />
</head>
<body>
  <div id="swagger-ui"></div>
  <script src="https://unpkg.com/swagger-ui-dist@4/swagger-ui-bundle.js"></script>
  <script src="https://unpkg.com/swagger-ui-dist@4/swagger-ui-standalone-preset.js"></script>
  <script>
    window.onload = function() {
      SwaggerUIBundle({
        url: '/openapi.yaml',
        dom_id: '#swagger-ui',
        presets: [SwaggerUIBundle.presets.apis, SwaggerUIStandalonePreset],
        layout: 'BaseLayout',
      });
    };
  </script>
</body>
</html>`))
}

func respondJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func respondError(w http.ResponseWriter, status int, errorMsg string) {
	respondJSON(w, status, map[string]interface{}{
		"code":  status,
		"error": errorMsg,
	})
}
