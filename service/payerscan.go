package service

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting"

	"github.com/shopspring/decimal"
)

// PayerScanProductionBaseURL is the documented production host. Operators may
// override it through the PayerScanBaseURL option when PayerScan issues them a
// different endpoint.
const PayerScanProductionBaseURL = "https://api.payerscan.com"

const (
	payerScanRequestTimeout = 30 * time.Second
	// payerScanMaxResponseBytes bounds the upstream body we buffer. Invoice
	// payloads are a few hundred bytes; anything larger is a malfunction.
	payerScanMaxResponseBytes = 1 << 20
)

// PayerScan invoice lifecycle values.
const (
	PayerScanStatusWaiting    = "waiting"
	PayerScanStatusProcessing = "processing"
	PayerScanStatusCompleted  = "completed"
	PayerScanStatusExpired    = "expired"
)

// payerScanNumber accepts both the quoted and unquoted JSON forms PayerScan
// uses for monetary fields (`"amount": "100"` when creating an invoice,
// `"amount": 100` in some error envelopes) and keeps the raw text so the value
// can be parsed as an exact decimal rather than a float.
type payerScanNumber string

func (n *payerScanNumber) UnmarshalJSON(data []byte) error {
	raw := strings.TrimSpace(string(data))
	if raw == "" || raw == "null" {
		*n = ""
		return nil
	}
	if unquoted, err := strconv.Unquote(raw); err == nil {
		*n = payerScanNumber(strings.TrimSpace(unquoted))
		return nil
	}
	*n = payerScanNumber(raw)
	return nil
}

// Decimal returns the amount as an exact decimal. A blank or unparsable value
// yields zero, which every caller treats as "amount not confirmed".
func (n payerScanNumber) Decimal() decimal.Decimal {
	value, err := decimal.NewFromString(strings.TrimSpace(string(n)))
	if err != nil {
		return decimal.Zero
	}
	return value
}

// PayerScanInvoice is the `data` object returned by both the create and the
// fetch endpoints. Fields absent from a given response stay zero-valued.
type PayerScanInvoice struct {
	TransID         string          `json:"trans_id"`
	RequestID       string          `json:"request_id"`
	Status          string          `json:"status"`
	Amount          payerScanNumber `json:"amount"`
	URLPayment      string          `json:"url_payment"`
	TokenSymbol     string          `json:"token_symbol"`
	NetworkSymbol   string          `json:"network_symbol"`
	TokenAmount     payerScanNumber `json:"token_amount"`
	TransactionHash string          `json:"transaction_hash"`
}

// PayerScanCreateInvoiceParams mirrors the documented POST /payment/crypto body.
// RequestID carries our trade number and is echoed back in webhooks.
type PayerScanCreateInvoiceParams struct {
	Amount       decimal.Decimal
	RequestID    string
	Name         string
	Description  string
	CallbackURL  string
	CompletedURL string
	ExpiredURL   string
}

type payerScanEnvelope struct {
	Status    string            `json:"status"`
	Message   string            `json:"message"`
	ErrorCode string            `json:"error_code"`
	Data      *PayerScanInvoice `json:"data"`
}

// PayerScanBaseURL returns the configured host without a trailing slash,
// falling back to the documented production host.
func PayerScanBaseURL() string {
	base := strings.TrimRight(strings.TrimSpace(setting.PayerScanBaseURL), "/")
	if base == "" {
		return PayerScanProductionBaseURL
	}
	return base
}

// ValidatePayerScanBaseURL rejects endpoints the gateway must never be pointed
// at. An empty value is valid and means "use the production host".
func ValidatePayerScanBaseURL(raw string) error {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil
	}
	parsed, err := url.Parse(trimmed)
	if err != nil {
		return fmt.Errorf("invalid PayerScan base URL: %w", err)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return fmt.Errorf("PayerScan base URL must use http or https")
	}
	if parsed.Host == "" {
		return fmt.Errorf("PayerScan base URL must include a host")
	}
	return nil
}

func payerScanHTTPClient() *http.Client {
	return &http.Client{Timeout: payerScanRequestTimeout}
}

// CreatePayerScanInvoice opens a hosted checkout and returns the invoice to
// redirect the buyer to.
func CreatePayerScanInvoice(ctx context.Context, params *PayerScanCreateInvoiceParams) (*PayerScanInvoice, error) {
	if params == nil {
		return nil, fmt.Errorf("missing invoice params")
	}
	merchantID := strings.TrimSpace(setting.PayerScanMerchantID)
	apiKey := strings.TrimSpace(setting.PayerScanApiKey)
	if merchantID == "" || apiKey == "" {
		return nil, fmt.Errorf("PayerScan credentials are not configured")
	}
	if strings.TrimSpace(params.RequestID) == "" {
		return nil, fmt.Errorf("missing request id")
	}
	if params.Amount.LessThanOrEqual(decimal.Zero) {
		return nil, fmt.Errorf("invoice amount must be positive")
	}

	body := map[string]any{
		"merchant_id": merchantID,
		"amount":      params.Amount.StringFixed(2),
		"request_id":  params.RequestID,
	}
	for key, value := range map[string]string{
		"name":          params.Name,
		"description":   params.Description,
		"callback_url":  params.CallbackURL,
		"completed_url": params.CompletedURL,
		"expired_url":   params.ExpiredURL,
	} {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			body[key] = trimmed
		}
	}

	payload, err := common.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("encode PayerScan invoice request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, PayerScanBaseURL()+"/payment/crypto", bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("build PayerScan invoice request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", apiKey)

	invoice, err := sendPayerScanRequest(req)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(invoice.URLPayment) == "" {
		return nil, fmt.Errorf("PayerScan returned no checkout URL")
	}
	return invoice, nil
}

// FetchPayerScanInvoice re-reads an invoice from PayerScan. Webhook bodies are
// only a notification hint: settlement always confirms the state here, so a
// forged callback cannot credit an order.
func FetchPayerScanInvoice(ctx context.Context, transID string) (*PayerScanInvoice, error) {
	apiKey := strings.TrimSpace(setting.PayerScanApiKey)
	if apiKey == "" {
		return nil, fmt.Errorf("PayerScan credentials are not configured")
	}
	trimmed := strings.TrimSpace(transID)
	if trimmed == "" {
		return nil, fmt.Errorf("missing trans id")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, PayerScanBaseURL()+"/invoice/"+url.PathEscape(trimmed), nil)
	if err != nil {
		return nil, fmt.Errorf("build PayerScan invoice lookup: %w", err)
	}
	req.Header.Set("x-api-key", apiKey)

	return sendPayerScanRequest(req)
}

func sendPayerScanRequest(req *http.Request) (*PayerScanInvoice, error) {
	resp, err := payerScanHTTPClient().Do(req)
	if err != nil {
		return nil, fmt.Errorf("call PayerScan: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(io.LimitReader(resp.Body, payerScanMaxResponseBytes))
	if err != nil {
		return nil, fmt.Errorf("read PayerScan response: %w", err)
	}

	var envelope payerScanEnvelope
	if err := common.Unmarshal(raw, &envelope); err != nil {
		return nil, fmt.Errorf("decode PayerScan response (http %d): %w", resp.StatusCode, err)
	}
	if resp.StatusCode != http.StatusOK || envelope.Status != "success" || envelope.Data == nil {
		// error_code / message are PayerScan's own diagnostics; neither echoes
		// the API key, so both are safe to surface to the operator log.
		return nil, fmt.Errorf("PayerScan request failed (http %d, code %q): %s",
			resp.StatusCode, envelope.ErrorCode, envelope.Message)
	}
	return envelope.Data, nil
}
