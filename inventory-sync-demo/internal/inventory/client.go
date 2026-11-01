package inventory

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

type Supplier interface {
	Fetch(context.Context) ([]Product, error)
}

type SupplierClient struct {
	URL      string
	HTTP     *http.Client
	Attempts int
}

type supplierResponse struct {
	Products []Product `json:"products"`
}

func (c SupplierClient) Fetch(ctx context.Context) ([]Product, error) {
	httpClient := c.HTTP
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 3 * time.Second}
	}
	attempts := c.Attempts
	if attempts <= 0 {
		attempts = 3
	}
	var lastErr error
	for attempt := 0; attempt < attempts; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.URL, nil)
		if err != nil {
			return nil, err
		}
		resp, err := httpClient.Do(req)
		retry := false
		if err != nil {
			lastErr = fmt.Errorf("supplier request: %w", err)
			retry = true
		} else {
			if resp.StatusCode == http.StatusOK {
				limited := io.LimitReader(resp.Body, 1<<20+1)
				b, readErr := io.ReadAll(limited)
				resp.Body.Close()
				if readErr != nil {
					return nil, readErr
				}
				if len(b) > 1<<20 {
					return nil, errors.New("supplier response exceeds 1 MiB")
				}
				var decoded supplierResponse
				if err := json.Unmarshal(b, &decoded); err != nil {
					return nil, fmt.Errorf("invalid supplier JSON: %w", err)
				}
				if decoded.Products == nil {
					return nil, errors.New("supplier response missing products array")
				}
				return decoded.Products, nil
			}
			// Retry only rate limiting and transient server errors, not bad input.
			retry = resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= http.StatusInternalServerError
			lastErr = fmt.Errorf("supplier returned HTTP %d", resp.StatusCode)
			io.Copy(io.Discard, io.LimitReader(resp.Body, 1024))
			resp.Body.Close()
		}
		if !retry || attempt == attempts-1 {
			break
		}
		delay := time.Duration(200*(1<<attempt)) * time.Millisecond
		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil, ctx.Err()
		case <-timer.C:
		}
	}
	return nil, lastErr
}
