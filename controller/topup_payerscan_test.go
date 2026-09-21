package controller

import (
	"context"
	"maps"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting"
	"github.com/QuantumNous/new-api/setting/operation_setting"

	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func configurePayerScanForTest(t *testing.T, baseURL string) {
	t.Helper()
	confirmPaymentComplianceForTest(t)
	originalEnabled := setting.PayerScanEnabled
	originalMerchantID := setting.PayerScanMerchantID
	originalApiKey := setting.PayerScanApiKey
	originalBaseURL := setting.PayerScanBaseURL
	t.Cleanup(func() {
		setting.PayerScanEnabled = originalEnabled
		setting.PayerScanMerchantID = originalMerchantID
		setting.PayerScanApiKey = originalApiKey
		setting.PayerScanBaseURL = originalBaseURL
	})
	setting.PayerScanEnabled = true
	setting.PayerScanMerchantID = "MID-TEST"
	setting.PayerScanApiKey = "psk_live_secret"
	setting.PayerScanBaseURL = baseURL
}

func TestIsPayerScanTopUpEnabledRequiresToggleAndCredentials(t *testing.T) {
	configurePayerScanForTest(t, "")
	require.True(t, isPayerScanTopUpEnabled())

	setting.PayerScanEnabled = false
	assert.False(t, isPayerScanTopUpEnabled(), "disabled gateway must not be offered")

	setting.PayerScanEnabled = true
	setting.PayerScanApiKey = "   "
	assert.False(t, isPayerScanTopUpEnabled(), "blank API key must not count as configured")

	setting.PayerScanApiKey = "psk_live_secret"
	setting.PayerScanMerchantID = ""
	assert.False(t, isPayerScanTopUpEnabled(), "missing merchant id must not count as configured")

	setting.PayerScanMerchantID = "MID-TEST"
	operation_setting.GetPaymentSetting().ComplianceConfirmed = false
	assert.False(t, isPayerScanTopUpEnabled(), "unconfirmed compliance must disable the gateway")
}

func TestGetPayerScanPayMoney(t *testing.T) {
	originalUnitPrice := setting.PayerScanUnitPrice
	originalQuotaDisplayType := operation_setting.GetGeneralSetting().QuotaDisplayType
	originalDiscounts := make(map[int]float64, len(operation_setting.GetPaymentSetting().AmountDiscount))
	maps.Copy(originalDiscounts, operation_setting.GetPaymentSetting().AmountDiscount)
	originalTopupGroupRatio := common.TopupGroupRatio2JSONString()

	t.Cleanup(func() {
		setting.PayerScanUnitPrice = originalUnitPrice
		operation_setting.GetGeneralSetting().QuotaDisplayType = originalQuotaDisplayType
		operation_setting.GetPaymentSetting().AmountDiscount = originalDiscounts
		require.NoError(t, common.UpdateTopupGroupRatioByJSONString(originalTopupGroupRatio))
	})

	setting.PayerScanUnitPrice = 2.5
	operation_setting.GetPaymentSetting().AmountDiscount = map[int]float64{
		10:                           0.8,
		int(common.QuotaPerUnit * 3): 0.5,
		20:                           0,
	}
	require.NoError(t, common.UpdateTopupGroupRatioByJSONString(`{"default":1,"vip":1.2}`))

	testCases := []struct {
		name             string
		amount           int64
		group            string
		quotaDisplayType string
		expected         string
	}{
		{
			name:             "currency display applies unit price group ratio and discount",
			amount:           10,
			group:            "vip",
			quotaDisplayType: operation_setting.QuotaDisplayTypeUSD,
			expected:         "24.00",
		},
		{
			name:             "tokens display converts quota to display units before pricing",
			amount:           int64(common.QuotaPerUnit * 3),
			group:            "vip",
			quotaDisplayType: operation_setting.QuotaDisplayTypeTokens,
			expected:         "4.50",
		},
		{
			name:             "non-positive discount falls back to no discount",
			amount:           20,
			group:            "default",
			quotaDisplayType: operation_setting.QuotaDisplayTypeUSD,
			expected:         "50.00",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			operation_setting.GetGeneralSetting().QuotaDisplayType = tc.quotaDisplayType
			require.Equal(t, tc.expected, getPayerScanPayMoney(tc.amount, tc.group).StringFixed(2))
		})
	}
}

func postPayerScanWebhook(t *testing.T, body string) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/payerscan/webhook", strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	PayerScanWebhook(c)
	return recorder
}

// The callback carries no signature, so merchant id and API key are the only
// gate before the handler talks to PayerScan. A settled order must never be
// reachable from a body that fails either check.
func TestPayerScanWebhookRejectsUnauthenticatedCallbacks(t *testing.T) {
	configurePayerScanForTest(t, "http://127.0.0.1:0")

	testCases := []struct {
		name           string
		body           string
		expectedStatus int
	}{
		{
			name:           "wrong merchant id",
			body:           `{"merchant_id":"MID-ATTACKER","api_key":"psk_live_secret","request_id":"PAYERSCAN-1-2-abcdef","trans_id":"TID-1","status":"completed"}`,
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "completed callback without api key",
			body:           `{"merchant_id":"MID-TEST","request_id":"PAYERSCAN-1-2-abcdef","trans_id":"TID-1","status":"completed"}`,
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "completed callback with wrong api key",
			body:           `{"merchant_id":"MID-TEST","api_key":"psk_live_guess","request_id":"PAYERSCAN-1-2-abcdef","trans_id":"TID-1","status":"completed"}`,
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "malformed body",
			body:           `not json`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "foreign order number is ignored without retry",
			body:           `{"merchant_id":"MID-TEST","api_key":"psk_live_secret","request_id":"ORDER_123","trans_id":"TID-1","status":"completed"}`,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "unknown status is ignored without retry",
			body:           `{"merchant_id":"MID-TEST","api_key":"psk_live_secret","request_id":"PAYERSCAN-1-2-abcdef","trans_id":"TID-1","status":"processing"}`,
			expectedStatus: http.StatusOK,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.expectedStatus, postPayerScanWebhook(t, tc.body).Code)
		})
	}
}

func TestPayerScanWebhookRejectedWhenGatewayDisabled(t *testing.T) {
	configurePayerScanForTest(t, "")
	setting.PayerScanEnabled = false

	recorder := postPayerScanWebhook(t, `{"merchant_id":"MID-TEST","api_key":"psk_live_secret","request_id":"PAYERSCAN-1-2-abcdef","trans_id":"TID-1","status":"completed"}`)
	require.Equal(t, http.StatusForbidden, recorder.Code)
}

func TestCreatePayerScanInvoiceSendsDocumentedRequest(t *testing.T) {
	var (
		gotPath   string
		gotKey    string
		gotBody   map[string]any
		gotMethod string
	)
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		gotKey = r.Header.Get("x-api-key")
		require.NoError(t, common.DecodeJson(r.Body, &gotBody))
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"success","data":{"request_id":"PAYERSCAN-1","trans_id":"TID-ABC","status":"waiting","amount":"24.00","url_payment":"https://app.payerscan.com/bill/TID-ABC","expires_in_minutes":15}}`))
	}))
	t.Cleanup(upstream.Close)
	configurePayerScanForTest(t, upstream.URL)

	invoice, err := service.CreatePayerScanInvoice(context.Background(), &service.PayerScanCreateInvoiceParams{
		Amount:      decimal.NewFromFloat(24),
		RequestID:   "PAYERSCAN-1",
		Name:        "Top-up balance",
		CallbackURL: "https://gateway.example.com/api/payerscan/webhook",
	})
	require.NoError(t, err)

	assert.Equal(t, http.MethodPost, gotMethod)
	assert.Equal(t, "/payment/crypto", gotPath)
	assert.Equal(t, "psk_live_secret", gotKey)
	assert.Equal(t, "MID-TEST", gotBody["merchant_id"])
	assert.Equal(t, "24.00", gotBody["amount"])
	assert.Equal(t, "PAYERSCAN-1", gotBody["request_id"])
	assert.Equal(t, "https://gateway.example.com/api/payerscan/webhook", gotBody["callback_url"])
	assert.NotContains(t, gotBody, "completed_url", "blank optional fields must be omitted")

	assert.Equal(t, "TID-ABC", invoice.TransID)
	assert.Equal(t, "https://app.payerscan.com/bill/TID-ABC", invoice.URLPayment)
	assert.Equal(t, "24", invoice.Amount.Decimal().String())
}

func TestFetchPayerScanInvoiceSurfacesUpstreamState(t *testing.T) {
	testCases := []struct {
		name           string
		status         int
		body           string
		expectError    bool
		expectedStatus string
		expectedAmount string
	}{
		{
			name:           "completed invoice with string amount",
			status:         http.StatusOK,
			body:           `{"status":"success","data":{"trans_id":"TID-ABC","request_id":"PAYERSCAN-1","status":"completed","amount":"24.00","transaction_hash":"0xabc"}}`,
			expectedStatus: "completed",
			expectedAmount: "24",
		},
		{
			name:           "completed invoice with numeric amount",
			status:         http.StatusOK,
			body:           `{"status":"success","data":{"trans_id":"TID-ABC","request_id":"PAYERSCAN-1","status":"completed","amount":24.5}}`,
			expectedStatus: "completed",
			expectedAmount: "24.5",
		},
		{
			name:        "error envelope",
			status:      http.StatusNotFound,
			body:        `{"status":"error","message":"Invoice not found","error_code":"INVOICE_NOT_FOUND"}`,
			expectError: true,
		},
		{
			name:        "success envelope without data",
			status:      http.StatusOK,
			body:        `{"status":"success"}`,
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/invoice/TID-ABC", r.URL.Path)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tc.status)
				_, _ = w.Write([]byte(tc.body))
			}))
			t.Cleanup(upstream.Close)
			configurePayerScanForTest(t, upstream.URL)

			invoice, err := service.FetchPayerScanInvoice(context.Background(), "TID-ABC")
			if tc.expectError {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.expectedStatus, invoice.Status)
			assert.Equal(t, tc.expectedAmount, invoice.Amount.Decimal().String())
		})
	}
}

func TestValidatePayerScanBaseURL(t *testing.T) {
	testCases := []struct {
		name        string
		value       string
		expectError bool
	}{
		{name: "blank falls back to production", value: ""},
		{name: "https endpoint", value: "https://api.payerscan.com"},
		{name: "http endpoint", value: "http://127.0.0.1:8080"},
		{name: "unsupported scheme", value: "file:///etc/passwd", expectError: true},
		{name: "missing host", value: "https://", expectError: true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := service.ValidatePayerScanBaseURL(tc.value)
			if tc.expectError {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestPayerScanBaseURLFallsBackToProduction(t *testing.T) {
	configurePayerScanForTest(t, "  https://api.staging.example.com/  ")
	require.Equal(t, "https://api.staging.example.com", service.PayerScanBaseURL())

	setting.PayerScanBaseURL = "   "
	require.Equal(t, service.PayerScanProductionBaseURL, service.PayerScanBaseURL())
}
