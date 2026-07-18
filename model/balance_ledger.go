package model

// Balance ledger entry types.
const (
	LedgerTypeTopupPaidCredit       = "topup_paid_credit"
	LedgerTypeTopupPromotionalBonus = "topup_promotional_bonus"
	LedgerTypeRefundPaid            = "refund_paid"
	LedgerTypeRefundPromo           = "refund_promo"
	LedgerTypeAPIUsage              = "api_usage"
	LedgerTypeExpiration            = "expiration"
	LedgerTypeReversal              = "reversal"
	LedgerTypeManualAdjust          = "manual_adjustment"

	// Legacy aliases keep existing call sites and historical readers compatible.
	LedgerTypeTopupPaid  = LedgerTypeTopupPaidCredit
	LedgerTypeTopupPromo = LedgerTypeTopupPromotionalBonus
	LedgerTypeUsageNote  = LedgerTypeAPIUsage
)

// BalanceLedger is an immutable credit/debit audit trail for API credits.
// AmountQuota positive = credit, negative = reverse.
type BalanceLedger struct {
	Id          int64  `json:"id" gorm:"primaryKey"`
	UserId      int    `json:"user_id" gorm:"index;not null"`
	OrderID     string `json:"order_id" gorm:"type:varchar(64);index;default:''"`
	LotID       int64  `json:"lot_id" gorm:"index;not null;default:0"`
	BalanceType string `json:"balance_type" gorm:"type:varchar(16);index;default:''"`
	EntryType   string `json:"entry_type" gorm:"type:varchar(32);index;not null"`
	AmountQuota int    `json:"amount_quota" gorm:"not null"`
	AmountMicro int64  `json:"amount_micro" gorm:"not null;default:0"`
	Currency    string `json:"currency" gorm:"type:varchar(8);default:''"`
	Note        string `json:"note" gorm:"type:varchar(512);default:''"`
	CreatedAt   int64  `json:"created_at" gorm:"index"`
}

func (BalanceLedger) TableName() string {
	return "balance_ledgers"
}

func ListBalanceLedgerByUser(userId int, limit int) ([]*BalanceLedger, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	var list []*BalanceLedger
	err := DB.Where("user_id = ?", userId).Order("id desc").Limit(limit).Find(&list).Error
	return list, err
}

func ListBalanceLedgerByOrder(orderID string) ([]*BalanceLedger, error) {
	var list []*BalanceLedger
	err := DB.Where("order_id = ?", orderID).Order("id asc").Find(&list).Error
	return list, err
}
