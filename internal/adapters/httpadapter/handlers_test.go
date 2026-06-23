package httpadapter

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"merchant-transactions-api/internal/app"
	"merchant-transactions-api/internal/app/mocks"
	"merchant-transactions-api/internal/apperrors"
	"merchant-transactions-api/internal/domain"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestHandlerCases(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name                 string
		method               string
		url                  string
		body                 string
		setup                func(*mocks.TransactionRepository, *mocks.ReceivableRepository, *mocks.Numerator) *app.Orchestrator
		expectedStatus       int
		expectedResponseType string
	}{
		{
			name:                 "Given a health request when handler is invoked then returns ok",
			method:               http.MethodGet,
			url:                  "/health",
			expectedStatus:       http.StatusOK,
			expectedResponseType: "health",
		},
		{
			name:                 "Given an OpenAPI specification request when handler is invoked then returns YAML",
			method:               http.MethodGet,
			url:                  "/openapi.yaml",
			expectedStatus:       http.StatusOK,
			expectedResponseType: "openapi",
		},
		{
			name:                 "Given a Swagger UI request when handler is invoked then returns HTML",
			method:               http.MethodGet,
			url:                  "/docs",
			expectedStatus:       http.StatusOK,
			expectedResponseType: "swagger",
		},
		{
			name:                 "Given a transaction list request when handler is invoked then returns the transaction list",
			method:               http.MethodGet,
			url:                  "/transactions",
			expectedStatus:       http.StatusOK,
			expectedResponseType: "transactions",
			setup: func(txRepo *mocks.TransactionRepository, rcRepo *mocks.ReceivableRepository, _ *mocks.Numerator) *app.Orchestrator {
				txRepo.On("ListTransactions", mock.Anything).Return([]domain.Transaction{{ID: "1", Value: "100", Description: "ok"}}, nil).Once()
				return nil
			},
		},
		{
			name:           "Given a missing transaction request when handler is invoked then returns not found",
			method:         http.MethodGet,
			url:            "/transactions/1",
			expectedStatus: http.StatusNotFound,
			setup: func(txRepo *mocks.TransactionRepository, rcRepo *mocks.ReceivableRepository, _ *mocks.Numerator) *app.Orchestrator {
				txRepo.On("GetTransaction", mock.Anything, "1").Return((*domain.Transaction)(nil), apperrors.ErrNotFound).Once()
				return nil
			},
		},
		{
			name:                 "Given an invalid create transaction request when handler is invoked then returns bad request",
			method:               http.MethodPost,
			url:                  "/transactions",
			expectedStatus:       http.StatusBadRequest,
			expectedResponseType: "error",
		},
		{
			name:                 "Given a valid create transaction request when handler is invoked then returns created",
			method:               http.MethodPost,
			url:                  "/transactions",
			body:                 `{"value":"100","description":"test","method":"debit_card","cardNumber":"1234 5678 9012 3456","cardHolderName":"Test","cardExpirationDate":"04/28","cardCvv":"123"}`,
			expectedStatus:       http.StatusCreated,
			expectedResponseType: "createdTransaction",
			setup: func(txRepo *mocks.TransactionRepository, rcRepo *mocks.ReceivableRepository, numerator *mocks.Numerator) *app.Orchestrator {
				numerator.On("NextID", mock.Anything).Return("101", nil).Once()
				numerator.On("NextID", mock.Anything).Return("102", nil).Once()
				txRepo.On("SaveTransaction", mock.Anything, mock.Anything).Return(nil).Once()
				rcRepo.On("SaveReceivable", mock.Anything, mock.Anything).Return(nil).Once()
				return app.NewOrchestrator(txRepo, rcRepo, numerator)
			},
		},
		{
			name:           "Given an unsupported transaction method when handler is invoked then returns method not allowed",
			method:         http.MethodPut,
			url:            "/transactions",
			expectedStatus: http.StatusMethodNotAllowed,
		},
		{
			name:                 "Given a receivable list request when handler is invoked then returns receivables",
			method:               http.MethodGet,
			url:                  "/receivables",
			expectedStatus:       http.StatusOK,
			expectedResponseType: "receivables",
			setup: func(txRepo *mocks.TransactionRepository, rcRepo *mocks.ReceivableRepository, _ *mocks.Numerator) *app.Orchestrator {
				rcRepo.On("ListReceivables", mock.Anything).Return([]domain.Receivable{{ID: "1", TransactionID: "1", Status: "paid"}}, nil).Once()
				return nil
			},
		},
		{
			name:           "Given a missing receivable request when handler is invoked then returns not found",
			method:         http.MethodGet,
			url:            "/receivables/1",
			expectedStatus: http.StatusNotFound,
			setup: func(txRepo *mocks.TransactionRepository, rcRepo *mocks.ReceivableRepository, _ *mocks.Numerator) *app.Orchestrator {
				rcRepo.On("GetReceivable", mock.Anything, "1").Return((*domain.Receivable)(nil), apperrors.ErrNotFound).Once()
				return nil
			},
		},
		{
			name:           "Given a delete transaction request when handler is invoked then returns no content",
			method:         http.MethodDelete,
			url:            "/transactions/1",
			expectedStatus: http.StatusNoContent,
			setup: func(txRepo *mocks.TransactionRepository, rcRepo *mocks.ReceivableRepository, _ *mocks.Numerator) *app.Orchestrator {
				txRepo.On("DeleteTransaction", mock.Anything, "1").Return(nil).Once()
				return nil
			},
		},
		{
			name:           "Given an unsupported receivables method when handler is invoked then returns method not allowed",
			method:         http.MethodPost,
			url:            "/receivables",
			expectedStatus: http.StatusMethodNotAllowed,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			txRepo := &mocks.TransactionRepository{}
			rcRepo := &mocks.ReceivableRepository{}
			numerator := &mocks.Numerator{}
			var orchestrator *app.Orchestrator
			if tc.setup != nil {
				orchestrator = tc.setup(txRepo, rcRepo, numerator)
			}
			h := NewHandler(orchestrator, txRepo, rcRepo)

			var bodyReader io.Reader
			if tc.body != "" {
				bodyReader = strings.NewReader(tc.body)
			}
			req := httptest.NewRequest(tc.method, tc.url, bodyReader)
			rr := httptest.NewRecorder()

			switch {
			case tc.url == "/health":
				h.handleHealth(rr, req)
			case tc.url == "/transactions" && tc.method == http.MethodGet:
				h.handleTransactions(rr, req)
			case tc.url == "/transactions" && tc.method == http.MethodPost:
				h.handleTransactions(rr, req)
			case tc.url == "/transactions" && tc.method == http.MethodPut:
				h.handleTransactions(rr, req)
			case tc.url == "/transactions/1" && tc.method == http.MethodGet:
				h.handleTransactionByID(rr, req)
			case tc.url == "/transactions/1" && tc.method == http.MethodDelete:
				h.handleTransactionByID(rr, req)
			case tc.url == "/receivables" && tc.method == http.MethodGet:
				h.handleReceivables(rr, req)
			case tc.url == "/receivables" && tc.method == http.MethodPost:
				h.handleReceivables(rr, req)
			case tc.url == "/receivables/1" && tc.method == http.MethodGet:
				h.handleReceivableByID(rr, req)
			case tc.url == "/openapi.yaml" && tc.method == http.MethodGet:
				h.serveOpenAPISpec(rr, req)
			case tc.url == "/docs" && tc.method == http.MethodGet:
				h.serveSwaggerUI(rr, req)
			default:
				t.Fatalf("unexpected test route %s %s", tc.method, tc.url)
			}

			require.Equal(t, tc.expectedStatus, rr.Code)

			switch tc.expectedResponseType {
			case "health":
				var body map[string]string
				require.NoError(t, json.NewDecoder(rr.Body).Decode(&body))
				require.Equal(t, "ok", body["status"])
			case "transactions":
				var list []domain.Transaction
				require.NoError(t, json.NewDecoder(rr.Body).Decode(&list))
				require.Len(t, list, 1)
				require.Equal(t, "1", list[0].ID)
				txRepo.AssertExpectations(t)
			case "receivables":
				var list []domain.Receivable
				require.NoError(t, json.NewDecoder(rr.Body).Decode(&list))
				require.Len(t, list, 1)
				require.Equal(t, "1", list[0].ID)
				rcRepo.AssertExpectations(t)
			case "createdTransaction":
				var resp map[string]json.RawMessage
				require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
				require.Contains(t, resp, "transaction")
				require.Contains(t, resp, "receivable")
				txRepo.AssertExpectations(t)
				rcRepo.AssertExpectations(t)
				numerator.AssertExpectations(t)
			case "openapi":
				require.Contains(t, rr.Body.String(), "openapi:")
			case "swagger":
				require.Contains(t, rr.Body.String(), "SwaggerUIBundle")
			case "error":
				var body map[string]any
				require.NoError(t, json.NewDecoder(rr.Body).Decode(&body))
				require.Equal(t, float64(tc.expectedStatus), body["code"])
				require.Contains(t, body["error"], "invalid")
			default:
				// no additional assertions required
			}
		})
	}
}
