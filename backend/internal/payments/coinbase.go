package payments

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// coinbaseAPI is the Coinbase Commerce base URL. Coinbase Commerce provides a
// hosted checkout that accepts BTC, ETH, SOL and more.
const coinbaseAPI = "https://api.commerce.coinbase.com"

// SupportedCoins is advertised to the frontend.
var SupportedCoins = []string{"BTC", "ETH", "SOL"}

// coinbaseClient talks to the Coinbase Commerce Charges API.
type coinbaseClient struct {
	apiKey        string
	webhookSecret string
	http          *http.Client
}

func newCoinbaseClient(apiKey, webhookSecret string) *coinbaseClient {
	return &coinbaseClient{
		apiKey:        apiKey,
		webhookSecret: webhookSecret,
		http:          &http.Client{Timeout: 15 * time.Second},
	}
}

type coinbaseCharge struct {
	Code      string
	HostedURL string
	Status    string
}

// createCharge creates a fixed-price hosted charge for an order.
func (c *coinbaseClient) createCharge(ctx context.Context, name, description, amount, currency, orderID, redirectURL, cancelURL string) (*coinbaseCharge, error) {
	body := map[string]any{
		"name":         name,
		"description":  description,
		"pricing_type": "fixed_price",
		"local_price":  map[string]string{"amount": amount, "currency": currency},
		"metadata":     map[string]string{"order_id": orderID},
		"redirect_url": redirectURL,
		"cancel_url":   cancelURL,
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, coinbaseAPI+"/charges", bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-CC-Api-Key", c.apiKey)
	req.Header.Set("X-CC-Version", "2018-03-22")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("coinbase request: %w", err)
	}
	defer resp.Body.Close()
	payload, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode/100 != 2 {
		return nil, fmt.Errorf("coinbase charge failed (%d): %s", resp.StatusCode, string(payload))
	}

	var parsed struct {
		Data struct {
			Code      string `json:"code"`
			HostedURL string `json:"hosted_url"`
			Timeline  []struct {
				Status string `json:"status"`
			} `json:"timeline"`
		} `json:"data"`
	}
	if err := json.Unmarshal(payload, &parsed); err != nil {
		return nil, fmt.Errorf("coinbase decode: %w", err)
	}
	status := "NEW"
	if n := len(parsed.Data.Timeline); n > 0 {
		status = parsed.Data.Timeline[n-1].Status
	}
	return &coinbaseCharge{Code: parsed.Data.Code, HostedURL: parsed.Data.HostedURL, Status: status}, nil
}

// verifyWebhook validates the X-CC-Webhook-Signature header (hex HMAC-SHA256 of
// the raw body under the shared secret) in constant time.
func (c *coinbaseClient) verifyWebhook(payload []byte, sigHeader string) bool {
	mac := hmac.New(sha256.New, []byte(c.webhookSecret))
	mac.Write(payload)
	expected := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(expected), []byte(sigHeader))
}

// coinbaseEvent is the shape we parse from a verified webhook body.
type coinbaseEvent struct {
	Event struct {
		ID   string `json:"id"`
		Type string `json:"type"`
		Data struct {
			Code     string            `json:"code"`
			Metadata map[string]string `json:"metadata"`
		} `json:"data"`
	} `json:"event"`
}

func parseCoinbaseEvent(payload []byte) (*coinbaseEvent, error) {
	var e coinbaseEvent
	if err := json.Unmarshal(payload, &e); err != nil {
		return nil, err
	}
	return &e, nil
}
