package numerator

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"time"

	"merchant-transactions-api/internal/apperrors"
)

type Client struct {
	baseURL    *url.URL
	client     *http.Client
	maxRetries int
}

type numeratorResponse struct {
	Numerator int64 `json:"numerator"`
}

type testAndSetPayload struct {
	OldValue int64 `json:"oldValue"`
	NewValue int64 `json:"newValue"`
}

type errorResponse struct {
	Error            string `json:"error"`
	CurrentNumerator int64  `json:"currentNumerator,omitempty"`
}

// use centralized errors defined in internal/apperrors

func NewClient(baseURL string, client *http.Client, maxRetries int) (*Client, error) {
	parsed, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("invalid numerator base URL: %w", err)
	}
	if client == nil {
		client = http.DefaultClient
	}
	return &Client{baseURL: parsed, client: client, maxRetries: maxRetries}, nil
}

func (c *Client) NextID(ctx context.Context) (string, error) {
	for attempt := 0; attempt < c.maxRetries; attempt++ {
		current, err := c.currentNumerator(ctx)
		if err != nil {
			return "", err
		}

		nextValue := current + 1
		if err := c.testAndSet(ctx, current, nextValue); err == nil {
			log.Printf("numerator: allocated id %d (attempt %d)", nextValue, attempt+1)
			return fmt.Sprintf("%d", nextValue), nil
		} else if err == apperrors.ErrNumeratorMismatch {
			log.Printf("numerator: mismatch on attempt %d (current=%d), retrying", attempt+1, current)
			if sleepErr := sleepWithBackoff(ctx, attempt); sleepErr != nil {
				return "", sleepErr
			}
			continue
		} else {
			log.Printf("numerator: error on test-and-set: %v", err)
			return "", err
		}
	}
	return "", fmt.Errorf("%w after %d retries", apperrors.ErrUnableToObtainUniqueNumerator, c.maxRetries)
}

func sleepWithBackoff(ctx context.Context, attempt int) error {
	base := 100 * time.Millisecond
	jitter := time.Duration(rand.Intn(50)) * time.Millisecond
	delay := base*time.Duration(attempt+1) + jitter

	timer := time.NewTimer(delay)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func (c *Client) currentNumerator(ctx context.Context) (int64, error) {
	req, err := c.newRequest(ctx, http.MethodGet, "/numerator", nil)
	if err != nil {
		return 0, err
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		content, _ := io.ReadAll(resp.Body)
		return 0, fmt.Errorf("%w: status %d: %s", apperrors.ErrNumeratorService, resp.StatusCode, string(content))
	}

	var parsed numeratorResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return 0, err
	}
	return parsed.Numerator, nil
}

func (c *Client) testAndSet(ctx context.Context, oldValue, newValue int64) error {
	payload := testAndSetPayload{OldValue: oldValue, NewValue: newValue}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, err := c.newRequest(ctx, http.MethodPut, "/numerator/test-and-set", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusBadRequest {
		var parsed errorResponse
		if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
			return err
		}
		if parsed.Error == "Numerator does not match the expected old value." {
			return apperrors.ErrNumeratorMismatch
		}
		return fmt.Errorf("%w: %s", apperrors.ErrNumeratorService, parsed.Error)
	}
	if resp.StatusCode >= 400 {
		content, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("%w: status %d: %s", apperrors.ErrNumeratorService, resp.StatusCode, string(content))
	}
	return nil
}

func (c *Client) newRequest(ctx context.Context, method, relativePath string, body io.Reader) (*http.Request, error) {
	full := *c.baseURL
	full.Path = relativePath
	return http.NewRequestWithContext(ctx, method, full.String(), body)
}
