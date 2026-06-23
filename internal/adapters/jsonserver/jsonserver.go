package jsonserver

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"merchant-transactions-api/internal/apperrors"
	"merchant-transactions-api/internal/domain"
)

type Client struct {
	baseURL *url.URL
	client  *http.Client
}

func NewClient(baseURL string, client *http.Client) *Client {
	parsed, _ := url.Parse(baseURL)
	if client == nil {
		client = http.DefaultClient
	}
	return &Client{baseURL: parsed, client: client}
}

func (c *Client) SaveTransaction(ctx context.Context, transaction domain.Transaction) error {
	return c.postJSON(ctx, "/transactions", transaction)
}

func (c *Client) SaveReceivable(ctx context.Context, receivable domain.Receivable) error {
	return c.postJSON(ctx, "/receivables", receivable)
}

func (c *Client) GetTransaction(ctx context.Context, id string) (*domain.Transaction, error) {
	var transaction domain.Transaction
	err := c.getJSON(ctx, "/transactions/"+id, &transaction)
	if err != nil {
		return nil, err
	}
	return &transaction, nil
}

func (c *Client) ListTransactions(ctx context.Context) ([]domain.Transaction, error) {
	var transactions []domain.Transaction
	err := c.getJSON(ctx, "/transactions", &transactions)
	return transactions, err
}

func (c *Client) DeleteTransaction(ctx context.Context, id string) error {
	return c.delete(ctx, "/transactions/"+id)
}

func (c *Client) GetReceivable(ctx context.Context, id string) (*domain.Receivable, error) {
	var receivable domain.Receivable
	err := c.getJSON(ctx, "/receivables/"+id, &receivable)
	if err != nil {
		return nil, err
	}
	return &receivable, nil
}

func (c *Client) ListReceivables(ctx context.Context) ([]domain.Receivable, error) {
	var receivables []domain.Receivable
	err := c.getJSON(ctx, "/receivables", &receivables)
	return receivables, err
}

func (c *Client) DeleteReceivable(ctx context.Context, id string) error {
	return c.delete(ctx, "/receivables/"+id)
}

func (c *Client) postJSON(ctx context.Context, relativePath string, payload interface{}) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, err := c.newRequest(ctx, http.MethodPost, relativePath, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		content, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("%w: status %d: %s", apperrors.ErrJSONServerService, resp.StatusCode, string(content))
	}
	return nil
}

func (c *Client) getJSON(ctx context.Context, relativePath string, out interface{}) error {
	req, err := c.newRequest(ctx, http.MethodGet, relativePath, nil)
	if err != nil {
		return err
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return apperrors.ErrNotFound
	}
	if resp.StatusCode >= 400 {
		content, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("%w: status %d: %s", apperrors.ErrJSONServerService, resp.StatusCode, string(content))
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

func (c *Client) delete(ctx context.Context, relativePath string) error {
	req, err := c.newRequest(ctx, http.MethodDelete, relativePath, nil)
	if err != nil {
		return err
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		content, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("%w: status %d: %s", apperrors.ErrJSONServerService, resp.StatusCode, string(content))
	}
	return nil
}

func (c *Client) newRequest(ctx context.Context, method, relativePath string, body io.Reader) (*http.Request, error) {
	fullPath := *c.baseURL
	if !strings.HasPrefix(relativePath, "/") {
		relativePath = "/" + relativePath
	}
	fullPath.Path = strings.TrimRight(fullPath.Path, "/") + relativePath
	return http.NewRequestWithContext(ctx, method, fullPath.String(), body)
}
