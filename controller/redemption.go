package controller

import (
	"net/http"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/i18n"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting"
	"github.com/QuantumNous/new-api/setting/operation_setting"

	"github.com/gin-gonic/gin"
)

// computeQuotaFromCurrencyAmount converts a (currency, amount) pair into
// internal quota units using the current effective FX rates. currency must
// already be normalized to one of "usd"/"cny"/"cad" (empty is treated as USD).
func computeQuotaFromCurrencyAmount(currency string, amount float64) int {
	currency = strings.ToLower(strings.TrimSpace(currency))
	if amount <= 0 {
		return 0
	}
	var fxPerUSD float64
	switch currency {
	case setting.BillingCurrencyCNY:
		fxPerUSD = setting.EffectiveUSDCNYRate(operation_setting.USDExchangeRate)
	case setting.BillingCurrencyCAD:
		fxPerUSD = operation_setting.GetGeneralSetting().CADExchangeRate
	default: // "usd" or "" (legacy)
		fxPerUSD = 1
	}
	if fxPerUSD <= 0 {
		fxPerUSD = 1
	}
	usd := amount / fxPerUSD
	return common.QuotaFromFloat(usd * common.QuotaPerUnit)
}

// normalizeRedemptionCurrency validates and normalizes the currency code on
// incoming create/update requests. Empty currency is treated as USD for
// backward compatibility with legacy single-currency clients.
func normalizeRedemptionCurrency(currency string) (string, bool) {
	currency = strings.ToLower(strings.TrimSpace(currency))
	if currency == "" {
		return setting.BillingCurrencyUSD, true
	}
	return currency, setting.IsSupportedBillingCurrency(currency)
}

func GetAllRedemptions(c *gin.Context) {
	pageInfo := common.GetPageQuery(c)
	redemptions, total, err := model.GetAllRedemptions(pageInfo.GetStartIdx(), pageInfo.GetPageSize())
	if err != nil {
		common.ApiError(c, err)
		return
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(redemptions)
	common.ApiSuccess(c, pageInfo)
	return
}

func SearchRedemptions(c *gin.Context) {
	keyword := c.Query("keyword")
	status := c.Query("status")
	pageInfo := common.GetPageQuery(c)
	redemptions, total, err := model.SearchRedemptions(keyword, status, pageInfo.GetStartIdx(), pageInfo.GetPageSize())
	if err != nil {
		common.ApiError(c, err)
		return
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(redemptions)
	common.ApiSuccess(c, pageInfo)
	return
}

func GetRedemption(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	redemption, err := model.GetRedemptionById(id)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    redemption,
	})
	return
}

func AddRedemption(c *gin.Context) {
	redemption := model.Redemption{}
	err := c.ShouldBindJSON(&redemption)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if utf8.RuneCountInString(redemption.Name) == 0 || utf8.RuneCountInString(redemption.Name) > 20 {
		common.ApiErrorI18n(c, i18n.MsgRedemptionNameLength)
		return
	}
	if redemption.Count <= 0 {
		common.ApiErrorI18n(c, i18n.MsgRedemptionCountPositive)
		return
	}
	if redemption.Count > 100 {
		common.ApiErrorI18n(c, i18n.MsgRedemptionCountMax)
		return
	}
	if valid, msg := validateExpiredTime(c, redemption.ExpiredTime); !valid {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": msg})
		return
	}
	// 货币选择：cny/usd/cad。空串视为 USD（兼容旧客户端）。
	currency, ok := normalizeRedemptionCurrency(redemption.Currency)
	if !ok {
		common.ApiErrorI18n(c, i18n.MsgRedemptionCurrencyInvalid)
		return
	}
	redemption.Currency = currency
	// 价格必须为正数。允许旧的"直接传 quota"调用方继续工作：当 amount <= 0
	// 但 quota > 0 时按 USD 反推 amount。
	if redemption.Amount <= 0 {
		if redemption.Quota <= 0 {
			common.ApiErrorI18n(c, i18n.MsgRedemptionAmountPositive)
			return
		}
		// 兼容旧调用：从 quota 反推 USD 价格
		redemption.Amount = float64(redemption.Quota) / common.QuotaPerUnit
		redemption.Currency = setting.BillingCurrencyUSD
	}
	// 总兑换次数：0 或 1 都按 1 次处理；上限 100000 防滥用。
	if redemption.MaxRedeems < 0 {
		common.ApiErrorI18n(c, i18n.MsgRedemptionMaxRedeemsInvalid)
		return
	}
	if redemption.MaxRedeems == 0 {
		redemption.MaxRedeems = 1
	}
	if redemption.MaxRedeems > 100000 {
		common.ApiErrorI18n(c, i18n.MsgRedemptionMaxRedeemsInvalid)
		return
	}
	// 用当前汇率把 (currency, amount) 换算为内部 quota
	redemption.Quota = computeQuotaFromCurrencyAmount(redemption.Currency, redemption.Amount)

	var keys []string
	for i := 0; i < redemption.Count; i++ {
		key := common.GetUUID()
		cleanRedemption := model.Redemption{
			UserId:      c.GetInt("id"),
			Name:        redemption.Name,
			Key:         key,
			CreatedTime: common.GetTimestamp(),
			Quota:       redemption.Quota,
			Currency:    redemption.Currency,
			Amount:      redemption.Amount,
			MaxRedeems:  redemption.MaxRedeems,
			ExpiredTime: redemption.ExpiredTime,
		}
		err = cleanRedemption.Insert()
		if err != nil {
			common.SysError("failed to insert redemption: " + err.Error())
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": i18n.T(c, i18n.MsgRedemptionCreateFailed),
				"data":    keys,
			})
			return
		}
		keys = append(keys, key)
	}
	recordManageAudit(c, "redemption.create", map[string]interface{}{
		"name":        redemption.Name,
		"count":       redemption.Count,
		"currency":    redemption.Currency,
		"amount":      redemption.Amount,
		"max_redeems": redemption.MaxRedeems,
		"quota":       logger.LogQuota(redemption.Quota),
	})
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    keys,
	})
	return
}

func DeleteRedemption(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	err := model.DeleteRedemptionById(id)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
	})
	return
}

func UpdateRedemption(c *gin.Context) {
	statusOnly := c.Query("status_only")
	redemption := model.Redemption{}
	err := c.ShouldBindJSON(&redemption)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	cleanRedemption, err := model.GetRedemptionById(redemption.Id)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if statusOnly == "" {
		if valid, msg := validateExpiredTime(c, redemption.ExpiredTime); !valid {
			c.JSON(http.StatusOK, gin.H{"success": false, "message": msg})
			return
		}
		// 货币选择：cny/usd/cad。空串视为 USD（兼容旧客户端）。
		currency, ok := normalizeRedemptionCurrency(redemption.Currency)
		if !ok {
			common.ApiErrorI18n(c, i18n.MsgRedemptionCurrencyInvalid)
			return
		}
		cleanRedemption.Currency = currency
		// 价格优先；如果 amount <= 0 但 quota > 0，保留原 quota 并按 USD 反推 amount。
		if redemption.Amount > 0 {
			cleanRedemption.Amount = redemption.Amount
			cleanRedemption.Quota = computeQuotaFromCurrencyAmount(currency, redemption.Amount)
		} else if redemption.Quota > 0 {
			// 兼容旧客户端：直接传 quota
			cleanRedemption.Quota = redemption.Quota
			cleanRedemption.Amount = float64(redemption.Quota) / common.QuotaPerUnit
			cleanRedemption.Currency = setting.BillingCurrencyUSD
		}
		// 总兑换次数：0 表示不更新，1+ 表示新上限。不允许低于已兑换次数。
		if redemption.MaxRedeems > 0 {
			if redemption.MaxRedeems > 100000 {
				common.ApiErrorI18n(c, i18n.MsgRedemptionMaxRedeemsInvalid)
				return
			}
			if redemption.MaxRedeems < cleanRedemption.RedeemedCount {
				common.ApiErrorI18n(c, i18n.MsgRedemptionMaxRedeemsBelowUsed)
				return
			}
			cleanRedemption.MaxRedeems = redemption.MaxRedeems
		}
		cleanRedemption.Name = redemption.Name
		cleanRedemption.ExpiredTime = redemption.ExpiredTime
	}
	if statusOnly != "" {
		cleanRedemption.Status = redemption.Status
	}
	err = cleanRedemption.Update()
	if err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    cleanRedemption,
	})
	return
}

func DeleteInvalidRedemption(c *gin.Context) {
	rows, err := model.DeleteInvalidRedemptions()
	if err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    rows,
	})
	return
}

func validateExpiredTime(c *gin.Context, expired int64) (bool, string) {
	if expired != 0 && expired < common.GetTimestamp() {
		return false, i18n.T(c, i18n.MsgRedemptionExpireTimeInvalid)
	}
	return true, ""
}
