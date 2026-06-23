package numerator

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNextIDWithRetryAndFailureCases(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name        string
		handler     http.HandlerFunc
		expectedID  string
		expectedErr bool
		maxRetries  int
	}{
		{
			name: "Given a numerator mismatch on the first attempt when NextID is called then retries and succeeds",
			handler: func() http.HandlerFunc {
				var testAndSetCalls int32
				return func(w http.ResponseWriter, r *http.Request) {
					switch r.URL.Path {
					case "/numerator":
						_ = json.NewEncoder(w).Encode(map[string]int64{"numerator": int64(100)})
					case "/numerator/test-and-set":
						if atomic.AddInt32(&testAndSetCalls, 1) == 1 {
							w.WriteHeader(http.StatusBadRequest)
							_ = json.NewEncoder(w).Encode(map[string]interface{}{"error": "Numerator does not match the expected old value."})
							return
						}
						w.WriteHeader(http.StatusOK)
					default:
						http.NotFound(w, r)
					}
				}
			}(),
			expectedID:  "101",
			expectedErr: false,
			maxRetries:  3,
		},
		{
			name: "Given the numerator service fails when NextID is called then returns an error",
			handler: func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/numerator" {
					w.WriteHeader(http.StatusInternalServerError)
					_, _ = w.Write([]byte("boom"))
					return
				}
				http.NotFound(w, r)
			},
			expectedErr: true,
			maxRetries:  1,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			server := httptest.NewServer(http.HandlerFunc(tc.handler))
			defer server.Close()

			client, err := NewClient(server.URL, nil, tc.maxRetries)
			require.NoError(t, err)
			id, err := client.NextID(context.Background())

			switch tc.expectedErr {
			case true:
				require.Error(t, err)
			default:
				require.NoError(t, err)
				require.Equal(t, tc.expectedID, id)
			}
		})
	}
}

func TestNewClientReturnsInvalidURLError(t *testing.T) {
	t.Parallel()

	client, err := NewClient("http://%zz", nil, 1)

	require.Nil(t, client)
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid numerator base URL")
}

func TestSleepWithBackoffRespectsCanceledContext(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := sleepWithBackoff(ctx, 0)

	require.ErrorIs(t, err, context.Canceled)
}
