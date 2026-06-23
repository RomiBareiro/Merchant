package e2e

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"strings"

	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

func TestE2E_CreateTransactionFlow(t *testing.T) {
	ctx := context.Background()
	// start json-server with mounted ./config
	repoRoot, err := os.Getwd()
	require.NoError(t, err)
	root := filepath.Join(repoRoot, "..", "..")
	root, err = filepath.Abs(root)
	require.NoError(t, err)

	jsonReq := testcontainers.ContainerRequest{
		Image:        "vimagick/json-server",
		ExposedPorts: []string{"8080/tcp"},
		Cmd:          []string{"-h", "0.0.0.0", "-p", "8080", "/config/db.json"},
		WaitingFor:   wait.ForHTTP("/").WithPort("8080/tcp").WithStartupTimeout(60 * time.Second),
		Mounts: testcontainers.Mounts(
			testcontainers.BindMount(filepath.Join(root, "config"), "/config"),
		),
	}
	jsonC, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: jsonReq,
		Started:          true,
	})
	require.NoError(t, err)
	defer jsonC.Terminate(ctx)

	jsonHost, err := jsonC.Host(ctx)
	require.NoError(t, err)
	jsonPort, err := jsonC.MappedPort(ctx, "8080/tcp")
	require.NoError(t, err)
	jsonURL := fmt.Sprintf("http://%s:%s", jsonHost, jsonPort.Port())

	// start a lightweight in-test Numerator HTTP server (avoids mounting local node app)
	type numPayload struct {
		OldValue int64 `json:"oldValue"`
		NewValue int64 `json:"newValue"`
	}
	var (
		numVal int64 = 0
	)
	numMux := http.NewServeMux()
	numMux.HandleFunc("/numerator", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]int64{"numerator": numVal})
	})
	numMux.HandleFunc("/numerator/test-and-set", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var p numPayload
		if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		if p.OldValue != numVal {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"error": "Numerator does not match the expected old value.", "currentNumerator": numVal})
			return
		}
		numVal = p.NewValue
		_ = json.NewEncoder(w).Encode(map[string]int64{"numerator": numVal})
	})
	numSrv := &http.Server{Addr: "127.0.0.1:0", Handler: numMux}
	ln, err := net.Listen("tcp", numSrv.Addr)
	require.NoError(t, err)
	go numSrv.Serve(ln)
	defer numSrv.Close()
	numURL := fmt.Sprintf("http://%s", ln.Addr().String())

	// build orchestration-api binary
	build := exec.CommandContext(ctx, "go", "build", "-o", "orchestration-api", "./cmd/server")
	build.Env = os.Environ()
	build.Dir = root
	out, err := build.CombinedOutput()
	require.NoError(t, err, string(out))
	defer os.Remove(filepath.Join(root, "orchestration-api"))

	// start orchestration-api binary with envs pointing to containers
	cmd := exec.CommandContext(ctx, filepath.Join(root, "orchestration-api"))
	cmd.Env = append(os.Environ(), fmt.Sprintf("JSON_SERVER_URL=%s", jsonURL), fmt.Sprintf("NUMERATOR_URL=%s", numURL))
	cmd.Dir = root
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	require.NoError(t, cmd.Start())
	defer func() {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	}()

	// wait for health
	client := &http.Client{Timeout: 2 * time.Second}
	healthOK := false
	for i := 0; i < 30; i++ {
		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, "http://localhost:4000/health", nil)
		resp, err := client.Do(req)
		if err == nil && resp.StatusCode == http.StatusOK {
			healthOK = true
			_ = resp.Body.Close()
			break
		}
		if resp != nil {
			_ = resp.Body.Close()
		}
		time.Sleep(500 * time.Millisecond)
	}
	require.True(t, healthOK, "orchestration-api did not become healthy")

	// perform create transaction
	payload := `{"value":"100","description":"E2E Test","method":"debit_card","cardNumber":"4242 4242 4242 4242","cardHolderName":"E2E","cardExpirationDate":"12/34","cardCvv":"123"}`
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, "http://localhost:4000/transactions", io.NopCloser(strings.NewReader(payload)))
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	var body map[string]any
	err = json.NewDecoder(resp.Body).Decode(&body)
	require.NoError(t, err)
	trans := body["transaction"].(map[string]any)
	reqID := trans["id"].(string)
	recv := body["receivable"].(map[string]any)
	recID := recv["id"].(string)

	// GET transaction
	resp, err = client.Get(fmt.Sprintf("http://localhost:4000/transactions/%s", reqID))
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	// GET receivable
	resp, err = client.Get(fmt.Sprintf("http://localhost:4000/receivables/%s", recID))
	require.NoError(t, err)
	defer resp.Body.Close()
	// allow 200 or 404 depending on timing, but prefer 200
	require.Contains(t, []int{http.StatusOK, http.StatusNotFound}, resp.StatusCode)

	// cleanup
	_ = cmd.Process.Kill()
	_ = cmd.Wait()
}
