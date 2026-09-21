package model

import (
	"errors"
	"fmt"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

const (
	PaymentMethodPayerScan   = "payerscan"
	PaymentProviderPayerScan = "payerscan"
)

// ErrTopUpAmountMismatch reports a settlement whose confirmed paid amount is
// below the amount the order was created for.
var ErrTopUpAmountMismatch = errors.New("topup paid amount is below the order amount")

// payerScanAmountTolerance absorbs the cent-level rounding PayerScan applies
// when it converts the invoice total to the token amount actually transferred.
var payerScanAmountTolerance = decimal.NewFromFloat(0.01)

// RechargePayerScan atomically settles a PayerScan order: row lock, provider
// and status checks, confirmed-amount check, completion and quota credit all
// happen inside one transaction, so concurrent or repeated callbacks (across
// instances too) credit the order at most once. alreadyDone=true means the
// order had already been settled and this callback was a duplicate.
//
// paidAmount is the amount confirmed by re-reading the invoice from PayerScan,
// never a number taken from the webhook body.
func RechargePayerScan(tradeNo string, paidAmount decimal.Decimal, callerIp string) (alreadyDone bool, err error) {
	if tradeNo == "" {
		return false, errors.New("未提供支付单号")
	}

	refCol := "`trade_no`"
	if common.UsingMainDatabase(common.DatabaseTypePostgreSQL) {
		refCol = `"trade_no"`
	}

	var quotaToAdd int
	topUp := &TopUp{}
	err = DB.Transaction(func(tx *gorm.DB) error {
		if err := lockForUpdate(tx).Where(refCol+" = ?", tradeNo).First(topUp).Error; err != nil {
			return ErrTopUpNotFound
		}
		if topUp.PaymentProvider != PaymentProviderPayerScan {
			return ErrPaymentMethodMismatch
		}
		if topUp.Status == common.TopUpStatusSuccess {
			alreadyDone = true
			return nil
		}
		if topUp.Status != common.TopUpStatusPending {
			return ErrTopUpStatusInvalid
		}
		// Underpayment must never credit the full order. The tolerance only
		// covers PayerScan's own cent-level conversion rounding.
		expected := decimal.NewFromFloat(topUp.Money).Sub(payerScanAmountTolerance)
		if paidAmount.LessThan(expected) {
			return ErrTopUpAmountMismatch
		}

		var quotaErr error
		quotaToAdd, quotaErr = common.WalletQuotaFromDecimalStrict(
			decimal.NewFromInt(topUp.Amount).Mul(decimal.NewFromFloat(common.QuotaPerUnit)),
		)
		if quotaErr != nil || quotaToAdd <= 0 {
			return ErrInvalidTopUpQuota
		}

		topUp.CompleteTime = common.GetTimestamp()
		topUp.Status = common.TopUpStatusSuccess
		if err := tx.Save(topUp).Error; err != nil {
			return err
		}
		return creditTopUpQuota(tx, topUp.UserId, quotaToAdd, nil)
	})
	if err != nil {
		if !errors.Is(err, ErrTopUpNotFound) && !errors.Is(err, ErrPaymentMethodMismatch) &&
			!errors.Is(err, ErrTopUpStatusInvalid) && !errors.Is(err, ErrTopUpAmountMismatch) {
			common.SysError("payerscan topup failed: " + err.Error())
		}
		return false, err
	}
	if alreadyDone {
		return true, nil
	}
	syncCreditUserQuotaCache(topUp.UserId, quotaToAdd, "payerscan topup")

	common.SysLog(fmt.Sprintf("PayerScan 充值成功 trade_no=%s user_id=%d quota_to_add=%d money=%.2f", topUp.TradeNo, topUp.UserId, quotaToAdd, topUp.Money))
	RecordTopupLog(topUp.UserId, fmt.Sprintf("使用 PayerScan 充值成功，充值额度: %v，支付金额：%.2f", logger.FormatQuota(quotaToAdd), topUp.Money), callerIp, topUp.PaymentMethod, PaymentProviderPayerScan)
	return false, nil
}
