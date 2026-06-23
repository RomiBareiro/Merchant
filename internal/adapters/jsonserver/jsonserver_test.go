package jsonserver

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"merchant-transactions-api/internal/apperrors"
	"merchant-transactions-api/internal/domain"

	"github.com/stretchr/testify/require"
)

func TestJSONServerClientCRUDTableDriven(t *testing.T) {
	transactions := make(map[string]domain.Transaction)
	receivables := make(map[string]domain.Receivable)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/transactions":
			var tx domain.Transaction
			_ = json.NewDecoder(r.Body).Decode(&tx)
			transactions[tx.ID] = tx
			w.WriteHeader(http.StatusCreated)
		case r.Method == http.MethodGet && r.URL.Path == "/transactions/1":
			if tx, ok := transactions["1"]; ok {
				_ = json.NewEncoder(w).Encode(tx)
				return
			}
			http.NotFound(w, r)
		case r.Method == http.MethodGet && r.URL.Path == "/transactions":
			var list []domain.Transaction
			for _, tx := range transactions {
				list = append(list, tx)
			}
			_ = json.NewEncoder(w).Encode(list)
		case r.Method == http.MethodDelete && r.URL.Path == "/transactions/1":
			delete(transactions, "1")
			w.WriteHeader(http.StatusOK)
		case r.Method == http.MethodPost && r.URL.Path == "/receivables":
			var rc domain.Receivable
			_ = json.NewDecoder(r.Body).Decode(&rc)
			receivables[rc.ID] = rc
			w.WriteHeader(http.StatusCreated)
		case r.Method == http.MethodGet && r.URL.Path == "/receivables/1":
			if rc, ok := receivables["1"]; ok {
				_ = json.NewEncoder(w).Encode(rc)
				return
			}
			http.NotFound(w, r)
		case r.Method == http.MethodGet && r.URL.Path == "/receivables":
			var list []domain.Receivable
			for _, rc := range receivables {
				list = append(list, rc)
			}
			_ = json.NewEncoder(w).Encode(list)
		case r.Method == http.MethodDelete && r.URL.Path == "/receivables/1":
			delete(receivables, "1")
			w.WriteHeader(http.StatusOK)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	client := NewClient(srv.URL, nil)

	cases := []struct {
		name           string
		action         string
		run            func() (int, error)
		expectedStatus int
	}{
		{
			name:   "Given save and retrieve when storing transaction then it can be retrieved",
			action: "save_tx",
			run: func() (int, error) {
				if err := client.SaveTransaction(context.Background(), domain.Transaction{ID: "1", Value: "100", Description: "test"}); err != nil {
					return 0, err
				}
				tx, err := client.GetTransaction(context.Background(), "1")
				if err != nil {
					return 0, err
				}
				if tx == nil || tx.ID != "1" {
					return 0, nil
				}
				return http.StatusOK, nil
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:   "Given list endpoint when listing transactions then returns one",
			action: "list_tx",
			run: func() (int, error) {
				list, err := client.ListTransactions(context.Background())
				if err != nil {
					return 0, err
				}
				if len(list) != 1 {
					return 0, nil
				}
				return http.StatusOK, nil
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:   "Given delete when deleting transaction then it is removed",
			action: "delete_tx",
			run: func() (int, error) {
				if err := client.DeleteTransaction(context.Background(), "1"); err != nil {
					return 0, err
				}
				_, err := client.GetTransaction(context.Background(), "1")
				if err == nil {
					return 0, nil
				}
				if err != apperrors.ErrNotFound {
					return 0, err
				}
				return http.StatusOK, nil
			},
			expectedStatus: http.StatusOK,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			status, err := tc.run()
			switch tc.expectedStatus {
			case http.StatusOK:
				require.NoError(t, err)
				require.Equal(t, http.StatusOK, status)
			default:
				t.Fatalf("unexpected expected status %d", tc.expectedStatus)
			}
		})
	}
}
