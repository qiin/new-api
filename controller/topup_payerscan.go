package controller

import (
	"crypto/subtle"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting"
	"github.com/QuantumNous/new-api/setting/operation_setting"

	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
	"github.com/thanhpk/randstr"
)

// payerScanTradeNoPrefix namespaces our order numbers inside PayerScan's
// `request_id`, so a callback can be attributed before touching the database.
const payerScanTradeNoPrefix = "PAYERSCAN-"

func isPayerScanTopUpEnabled() bool {
	if !isPaymentComplianceConfirmed() {
		return false
	}
	if !setting.PayerScanEnabled {
		return false
	}
	return isPayerScanWebhookConfigured()
}

func isPayerScanWebhookConfigured() bool {
	return strings.TrimSpace(setting.PayerScanMerchantID) != "" &&
		strings.TrimSpace(setting.PayerScanApiKey) != ""
}

func isPayerScanWebhookEnabled() bool {
	return isPayerScanTopUpEnabled()
}

type PayerScanPayRequest struct {
	Amount int64 `json:"amount"`
}

// getPayerScanPayMoney converts a requested top-up quantity into the USD amount
// PayerScan invoices, applying the top-up group ratio and the preset discount.
func getPayerScanPayMoney(amount int64, group string) decimal.Decimal {
	dAmount := decimal.NewFromInt(amount)
	if operation_setting.GetQuotaDisplayType() == operation_setting.QuotaDisplayTypeTokens {
		dAmount = dAmount.Div(decimal.NewFromFloat(common.QuotaPerUnit))
	}

	topupGroupRatio := common.GetTopupGroupRatio(group)
	if topupGroupRatio == 0 {
		topupGroupRatio = 1
	}

	discount := 1.0
	if ds, ok := operation_setting.GetPaymentSetting().AmountDiscount[int(amount)]; ok && ds > 0 {
		discount = ds
	}

	return dAmount.
		Mul(decimal.NewFromFloat(setting.PayerScanUnitPrice)).
		Mul(decimal.NewFromFloat(topupGroupRatio)).
		Mul(decimal.NewFromFloat(discount))
}

func RequestPayerScanAmount(c *gin.Context) {
	var req PayerScanPayRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "参数错误"})
		return
	}
	if req.Amount < int64(setting.PayerScanMinTopUp) {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": fmt.Sprintf("充值数量不能小于 %d", setting.PayerScanMinTopUp)})
		return
	}
	id := c.GetInt("id")
	if rejectInvalidTopUpQuota(c, id, req.Amount) {
		return
	}

	group, err := model.GetUserGroup(id, true)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "获取用户分组失败"})
		return
	}

	payMoney := getPayerScanPayMoney(req.Amount, group)
	if payMoney.LessThan(decimal.NewFromFloat(0.01)) {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "充值金额过低"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "success", "data": payMoney.StringFixed(2)})
}

func RequestPayerScanPay(c *gin.Context) {
	if !isPayerScanTopUpEnabled() {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "PayerScan 配置不完整"})
		return
	}

	var req PayerScanPayRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "参数错误"})
		return
	}
	if req.Amount < int64(setting.PayerScanMinTopUp) {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": fmt.Sprintf("充值数量不能小于 %d", setting.PayerScanMinTopUp)})
		return
	}
	id := c.GetInt("id")
	if rejectInvalidTopUpQuota(c, id, req.Amount) {
		return
	}

	user, err := model.GetUserById(id, false)
	if err != nil || user == nil {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "用户不存在"})
		return
	}

	group, err := model.GetUserGroup(id, true)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "获取用户分组失败"})
		return
	}

	payMoney := getPayerScanPayMoney(req.Amount, group)
	if payMoney.LessThan(decimal.NewFromFloat(0.01)) {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "充值金额过低"})
		return
	}
	// The invoice is raised for the rounded amount, so the order must store the
	// same figure; otherwise settlement would compare against an amount the
	// buyer was never charged.
	payMoney = payMoney.Round(2)

	// Orders are settled from Amount * QuotaPerUnit, so store the quantity in
	// wallet units even when the console displays tokens.
	storedAmount := req.Amount
	if operation_setting.GetQuotaDisplayType() == operation_setting.QuotaDisplayTypeTokens {
		storedAmount = decimal.NewFromInt(req.Amount).Div(decimal.NewFromFloat(common.QuotaPerUnit)).IntPart()
		if storedAmount < 1 {
			c.JSON(http.StatusOK, gin.H{"message": "error", "data": "充值数量无效"})
			return
		}
	}

	// Settlement only ever happens through the callback, so an order raised
	// without a reachable callback URL would take money and never credit it.
	callbackAddress := strings.TrimRight(strings.TrimSpace(service.GetCallbackAddress()), "/")
	if callbackAddress == "" {
		logger.LogError(c.Request.Context(), fmt.Sprintf("PayerScan 回调地址未配置 user_id=%d", id))
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "当前管理员未配置支付回调地址"})
		return
	}

	tradeNo := fmt.Sprintf("%s%d-%d-%s", payerScanTradeNoPrefix, id, time.Now().UnixMilli(), randstr.String(6))
	topUp := &model.TopUp{
		UserId:          id,
		Amount:          storedAmount,
		Money:           payMoney.InexactFloat64(),
		TradeNo:         tradeNo,
		PaymentMethod:   model.PaymentMethodPayerScan,
		PaymentProvider: model.PaymentProviderPayerScan,
		CreateTime:      time.Now().Unix(),
		Status:          common.TopUpStatusPending,
	}
	if err := topUp.Insert(); err != nil {
		logger.LogError(c.Request.Context(), fmt.Sprintf("PayerScan 创建充值订单失败 user_id=%d trade_no=%s amount=%d error=%q", id, tradeNo, req.Amount, err.Error()))
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "创建订单失败"})
		return
	}

	invoice, err := service.CreatePayerScanInvoice(c.Request.Context(), &service.PayerScanCreateInvoiceParams{
		Amount:       payMoney,
		RequestID:    tradeNo,
		Name:         "Top-up balance",
		Description:  fmt.Sprintf("Top-up the balance for username: %s", user.Username),
		CallbackURL:  callbackAddress + "/api/payerscan/webhook",
		CompletedURL: paymentReturnPath("/wallet?show_history=true"),
		ExpiredURL:   paymentReturnPath("/wallet"),
	})
	if err != nil {
		logger.LogError(c.Request.Context(), fmt.Sprintf("PayerScan 创建支付订单失败 user_id=%d trade_no=%s error=%q", id, tradeNo, err.Error()))
		topUp.Status = common.TopUpStatusFailed
		_ = topUp.Update()
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "拉起支付失败"})
		return
	}

	logger.LogInfo(c.Request.Context(), fmt.Sprintf("PayerScan 充值订单创建成功 user_id=%d trade_no=%s trans_id=%s amount=%d money=%s", id, tradeNo, invoice.TransID, req.Amount, payMoney.StringFixed(2)))
	c.JSON(http.StatusOK, gin.H{
		"message": "success",
		"data": gin.H{
			"checkout_url": invoice.URLPayment,
			"trans_id":     invoice.TransID,
			"order_id":     tradeNo,
		},
	})
}

// payerScanWebhookPayload is the notification PayerScan POSTs to callback_url.
// It is a hint only: nothing in it is trusted for settlement beyond locating
// the order, because the amounts and status are re-read from the API.
type payerScanWebhookPayload struct {
	MerchantID string `json:"merchant_id"`
	ApiKey     string `json:"api_key"`
	RequestID  string `json:"request_id"`
	TransID    string `json:"trans_id"`
	Status     string `json:"status"`
}

// PayerScanWebhook settles or expires a top-up order.
//
// PayerScan signs nothing: the callback only carries the store's API key for
// completed invoices. Matching that key in constant time keeps forged bodies
// from reaching the API, and every state change is then confirmed by re-reading
// the invoice from PayerScan, so a leaked callback URL cannot credit quota.
func PayerScanWebhook(c *gin.Context) {
	if !isPayerScanWebhookEnabled() {
		logger.LogWarn(c.Request.Context(), fmt.Sprintf("PayerScan webhook 被拒绝 reason=webhook_disabled path=%q client_ip=%s", c.Request.RequestURI, c.ClientIP()))
		c.String(http.StatusForbidden, "webhook disabled")
		return
	}

	bodyBytes, err := io.ReadAll(c.Request.Body)
	if err != nil {
		logger.LogError(c.Request.Context(), fmt.Sprintf("PayerScan webhook 读取请求体失败 path=%q client_ip=%s error=%q", c.Request.RequestURI, c.ClientIP(), err.Error()))
		c.String(http.StatusBadRequest, "bad request")
		return
	}

	var payload payerScanWebhookPayload
	if err := common.Unmarshal(bodyBytes, &payload); err != nil {
		logger.LogWarn(c.Request.Context(), fmt.Sprintf("PayerScan webhook 解析失败 path=%q client_ip=%s error=%q", c.Request.RequestURI, c.ClientIP(), err.Error()))
		c.String(http.StatusBadRequest, "bad request")
		return
	}

	// Never log the body: completed callbacks embed the store API key.
	status := strings.ToLower(strings.TrimSpace(payload.Status))
	tradeNo := strings.TrimSpace(payload.RequestID)
	transID := strings.TrimSpace(payload.TransID)
	logger.LogInfo(c.Request.Context(), fmt.Sprintf("PayerScan webhook 收到请求 client_ip=%s status=%s trade_no=%s trans_id=%s", c.ClientIP(), status, tradeNo, transID))

	if subtle.ConstantTimeCompare([]byte(strings.TrimSpace(payload.MerchantID)), []byte(strings.TrimSpace(setting.PayerScanMerchantID))) != 1 {
		logger.LogWarn(c.Request.Context(), fmt.Sprintf("PayerScan webhook 商户号不匹配 client_ip=%s trade_no=%s trans_id=%s", c.ClientIP(), tradeNo, transID))
		c.String(http.StatusUnauthorized, "invalid merchant")
		return
	}
	// Expired callbacks omit api_key by design; completed ones must carry it.
	if status == service.PayerScanStatusCompleted &&
		subtle.ConstantTimeCompare([]byte(strings.TrimSpace(payload.ApiKey)), []byte(strings.TrimSpace(setting.PayerScanApiKey))) != 1 {
		logger.LogWarn(c.Request.Context(), fmt.Sprintf("PayerScan webhook API Key 不匹配 client_ip=%s trade_no=%s trans_id=%s", c.ClientIP(), tradeNo, transID))
		c.String(http.StatusUnauthorized, "invalid api key")
		return
	}

	if !strings.HasPrefix(tradeNo, payerScanTradeNoPrefix) || transID == "" {
		logger.LogWarn(c.Request.Context(), fmt.Sprintf("PayerScan webhook 订单号无效 client_ip=%s trade_no=%s trans_id=%s", c.ClientIP(), tradeNo, transID))
		c.String(http.StatusOK, "OK")
		return
	}
	if status != service.PayerScanStatusCompleted && status != service.PayerScanStatusExpired {
		logger.LogInfo(c.Request.Context(), fmt.Sprintf("PayerScan webhook 忽略事件 client_ip=%s status=%s trade_no=%s", c.ClientIP(), status, tradeNo))
		c.String(http.StatusOK, "OK")
		return
	}

	// Resolve the order before calling PayerScan. Expired callbacks legitimately
	// arrive without an API key, so without this check an unauthenticated POST
	// could make the gateway issue an outbound request on demand.
	topUp := model.GetTopUpByTradeNo(tradeNo)
	if topUp == nil || topUp.PaymentProvider != model.PaymentProviderPayerScan {
		logger.LogWarn(c.Request.Context(), fmt.Sprintf("PayerScan 回调订单不存在 trade_no=%s trans_id=%s client_ip=%s", tradeNo, transID, c.ClientIP()))
		c.String(http.StatusOK, "OK")
		return
	}
	if topUp.Status != common.TopUpStatusPending {
		// A repeated callback on a settled order is routine; any other closed
		// state paired with a completion needs an operator to reconcile it.
		if status == service.PayerScanStatusCompleted && topUp.Status != common.TopUpStatusSuccess {
			logger.LogError(c.Request.Context(), fmt.Sprintf("PayerScan 已关闭订单收到支付完成回调 trade_no=%s trans_id=%s order_status=%s client_ip=%s", tradeNo, transID, topUp.Status, c.ClientIP()))
		} else {
			logger.LogInfo(c.Request.Context(), fmt.Sprintf("PayerScan 回调订单已结束 trade_no=%s trans_id=%s order_status=%s client_ip=%s", tradeNo, transID, topUp.Status, c.ClientIP()))
		}
		c.String(http.StatusOK, "OK")
		return
	}

	// Authoritative state. A transient lookup failure returns 500 so PayerScan
	// retries rather than dropping a paid invoice.
	invoice, err := service.FetchPayerScanInvoice(c.Request.Context(), transID)
	if err != nil {
		logger.LogError(c.Request.Context(), fmt.Sprintf("PayerScan 校验订单失败 trade_no=%s trans_id=%s client_ip=%s error=%q", tradeNo, transID, c.ClientIP(), err.Error()))
		c.String(http.StatusInternalServerError, "retry")
		return
	}
	if strings.TrimSpace(invoice.RequestID) != tradeNo {
		logger.LogWarn(c.Request.Context(), fmt.Sprintf("PayerScan 回调订单号与发票不符 trade_no=%s invoice_request_id=%s trans_id=%s client_ip=%s", tradeNo, invoice.RequestID, transID, c.ClientIP()))
		c.String(http.StatusOK, "OK")
		return
	}
	invoiceStatus := strings.ToLower(strings.TrimSpace(invoice.Status))
	if invoiceStatus != status {
		logger.LogWarn(c.Request.Context(), fmt.Sprintf("PayerScan 回调状态与发票不符 trade_no=%s callback_status=%s invoice_status=%s client_ip=%s", tradeNo, status, invoiceStatus, c.ClientIP()))
		c.String(http.StatusOK, "OK")
		return
	}

	// The in-process lock is an optimisation; correctness comes from the row
	// lock and in-transaction status check inside the model layer.
	LockOrder(tradeNo)
	defer UnlockOrder(tradeNo)

	if invoiceStatus == service.PayerScanStatusExpired {
		if err := model.UpdatePendingTopUpStatus(tradeNo, model.PaymentProviderPayerScan, common.TopUpStatusExpired); err != nil {
			// A settled or missing order is a permanent outcome, not a retry.
			logger.LogWarn(c.Request.Context(), fmt.Sprintf("PayerScan 订单过期处理跳过 trade_no=%s trans_id=%s client_ip=%s reason=%q", tradeNo, transID, c.ClientIP(), err.Error()))
		} else {
			logger.LogInfo(c.Request.Context(), fmt.Sprintf("PayerScan 订单已过期 trade_no=%s trans_id=%s client_ip=%s", tradeNo, transID, c.ClientIP()))
		}
		c.String(http.StatusOK, "OK")
		return
	}

	alreadyDone, err := model.RechargePayerScan(tradeNo, invoice.Amount.Decimal(), c.ClientIP())
	if err != nil {
		switch {
		case errors.Is(err, model.ErrTopUpNotFound):
			logger.LogWarn(c.Request.Context(), fmt.Sprintf("PayerScan 回调订单不存在 trade_no=%s trans_id=%s client_ip=%s", tradeNo, transID, c.ClientIP()))
		case errors.Is(err, model.ErrPaymentMethodMismatch):
			logger.LogWarn(c.Request.Context(), fmt.Sprintf("PayerScan 订单支付网关不匹配 trade_no=%s trans_id=%s client_ip=%s", tradeNo, transID, c.ClientIP()))
		case errors.Is(err, model.ErrTopUpStatusInvalid):
			logger.LogWarn(c.Request.Context(), fmt.Sprintf("PayerScan 订单状态非法 trade_no=%s trans_id=%s client_ip=%s", tradeNo, transID, c.ClientIP()))
		case errors.Is(err, model.ErrTopUpAmountMismatch):
			logger.LogError(c.Request.Context(), fmt.Sprintf("PayerScan 实付金额低于订单金额 trade_no=%s trans_id=%s paid=%s client_ip=%s", tradeNo, transID, invoice.Amount.Decimal().String(), c.ClientIP()))
		default:
			logger.LogError(c.Request.Context(), fmt.Sprintf("PayerScan 充值处理失败 trade_no=%s trans_id=%s client_ip=%s error=%q", tradeNo, transID, c.ClientIP(), err.Error()))
			c.String(http.StatusInternalServerError, "retry")
			return
		}
		c.String(http.StatusOK, "OK")
		return
	}

	if alreadyDone {
		logger.LogInfo(c.Request.Context(), fmt.Sprintf("PayerScan 重复回调幂等忽略 trade_no=%s trans_id=%s client_ip=%s", tradeNo, transID, c.ClientIP()))
	} else {
		logger.LogInfo(c.Request.Context(), fmt.Sprintf("PayerScan 充值成功 trade_no=%s trans_id=%s client_ip=%s", tradeNo, transID, c.ClientIP()))
	}
	c.String(http.StatusOK, "OK")
}
