package model

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"

	"gorm.io/gorm"
)

type Redemption struct {
	Id            int            `json:"id"`
	UserId        int            `json:"user_id"`
	Key           string         `json:"key" gorm:"type:varchar(64);uniqueIndex"`
	Status        int            `json:"status" gorm:"default:1"`
	Name          string         `json:"name" gorm:"index"`
	Quota         int            `json:"quota" gorm:"default:100"` // 内部 quota（由 Currency × Amount 换算得出）
	Currency      string         `json:"currency" gorm:"type:varchar(8);default:''"` // 原币种代码：usd/cny/cad，空串视为旧数据 USD
	Amount        float64        `json:"amount" gorm:"default:0"` // 原币种价格，例如 currency=cny amount=10 表示 10 元人民币
	MaxRedeems    int            `json:"max_redeems" gorm:"default:0"` // 总兑换次数上限，0 表示按 1 次（兼容旧数据），>1 为多次
	RedeemedCount int            `json:"redeemed_count" gorm:"default:0"` // 已兑换次数
	CreatedTime   int64          `json:"created_time" gorm:"bigint"`
	RedeemedTime  int64          `json:"redeemed_time" gorm:"bigint"`
	Count         int            `json:"count" gorm:"-:all"` // only for api request
	UsedUserId    int            `json:"used_user_id"`
	DeletedAt     gorm.DeletedAt `gorm:"index"`
	ExpiredTime   int64          `json:"expired_time" gorm:"bigint"` // 过期时间，0 表示不过期
}

// EffectiveMaxRedeems returns the effective total redeem limit.
// 0 is treated as 1 for backward compatibility with legacy single-use codes.
func (r *Redemption) EffectiveMaxRedeems() int {
	if r == nil {
		return 1
	}
	if r.MaxRedeems <= 0 {
		return 1
	}
	return r.MaxRedeems
}

func GetAllRedemptions(startIdx int, num int) (redemptions []*Redemption, total int64, err error) {
	// 开始事务
	tx := DB.Begin()
	if tx.Error != nil {
		return nil, 0, tx.Error
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 获取总数
	err = tx.Model(&Redemption{}).Count(&total).Error
	if err != nil {
		tx.Rollback()
		return nil, 0, err
	}

	// 获取分页数据
	err = tx.Order("id desc").Limit(num).Offset(startIdx).Find(&redemptions).Error
	if err != nil {
		tx.Rollback()
		return nil, 0, err
	}

	// 提交事务
	if err = tx.Commit().Error; err != nil {
		return nil, 0, err
	}

	return redemptions, total, nil
}

func SearchRedemptions(keyword string, status string, startIdx int, num int) (redemptions []*Redemption, total int64, err error) {
	tx := DB.Begin()
	if tx.Error != nil {
		return nil, 0, tx.Error
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	query := tx.Model(&Redemption{})

	if keyword != "" {
		if id, err := strconv.Atoi(keyword); err == nil {
			query = query.Where("id = ? OR name LIKE ?", id, keyword+"%")
		} else {
			query = query.Where("name LIKE ?", keyword+"%")
		}
	}

	if status != "" {
		now := common.GetTimestamp()
		switch status {
		case "expired":
			query = query.Where(
				"status = ? AND expired_time != 0 AND expired_time < ?",
				common.RedemptionCodeStatusEnabled,
				now,
			)
		case strconv.Itoa(common.RedemptionCodeStatusEnabled):
			query = query.Where(
				"status = ? AND (expired_time = 0 OR expired_time >= ?)",
				common.RedemptionCodeStatusEnabled,
				now,
			)
		case strconv.Itoa(common.RedemptionCodeStatusDisabled):
			query = query.Where("status = ?", common.RedemptionCodeStatusDisabled)
		case strconv.Itoa(common.RedemptionCodeStatusUsed):
			query = query.Where("status = ?", common.RedemptionCodeStatusUsed)
		}
	}

	// Get total count
	err = query.Count(&total).Error
	if err != nil {
		tx.Rollback()
		return nil, 0, err
	}

	// Get paginated data
	err = query.Order("id desc").Limit(num).Offset(startIdx).Find(&redemptions).Error
	if err != nil {
		tx.Rollback()
		return nil, 0, err
	}

	if err = tx.Commit().Error; err != nil {
		return nil, 0, err
	}

	return redemptions, total, nil
}

func GetRedemptionById(id int) (*Redemption, error) {
	if id == 0 {
		return nil, errors.New("id 为空！")
	}
	redemption := Redemption{Id: id}
	var err error = nil
	err = DB.First(&redemption, "id = ?", id).Error
	return &redemption, err
}

func Redeem(key string, userId int) (quota int, err error) {
	if key == "" {
		return 0, errors.New("未提供兑换码")
	}
	if userId == 0 {
		return 0, errors.New("无效的 user id")
	}
	redemption := &Redemption{}

	keyCol := "`key`"
	if common.UsingMainDatabase(common.DatabaseTypePostgreSQL) {
		keyCol = `"key"`
	}
	common.RandomSleep()
	err = DB.Transaction(func(tx *gorm.DB) error {
		err := lockForUpdate(tx).Where(keyCol+" = ?", key).First(redemption).Error
		if err != nil {
			return errors.New("无效的兑换码")
		}
		if redemption.Status != common.RedemptionCodeStatusEnabled {
			return errors.New("该兑换码已被使用")
		}
		if redemption.ExpiredTime != 0 && redemption.ExpiredTime < common.GetTimestamp() {
			return errors.New("该兑换码已过期")
		}
		// Per-user-once: insert a usage row; the unique index
		// (redemption_id, user_id) rejects duplicate redemptions by the
		// same user even under concurrent transactions.
		usage := &RedemptionUsage{
			RedemptionId: redemption.Id,
			UserId:       userId,
			CreatedTime:   common.GetTimestamp(),
			Quota:        redemption.Quota,
		}
		if insertErr := tx.Create(usage).Error; insertErr != nil {
			return ErrRedeemAlreadyUsedByUser
		}

		maxRedeems := redemption.EffectiveMaxRedeems()
		newCount := redemption.RedeemedCount + 1
		now := common.GetTimestamp()
		// Mark as Used when this redeem consumes the last slot. For
		// single-use codes (legacy default), this preserves the old
		// enabled -> used transition via compare-and-swap.
		if newCount >= maxRedeems {
			result := tx.Model(&Redemption{}).
				Where("id = ? AND status = ?", redemption.Id, common.RedemptionCodeStatusEnabled).
				Updates(map[string]interface{}{
					"redeemed_time":  now,
					"status":         common.RedemptionCodeStatusUsed,
					"used_user_id":  userId,
					"redeemed_count": newCount,
				})
			if result.Error != nil {
				return result.Error
			}
			if result.RowsAffected == 0 {
				return errors.New("该兑换码已被使用")
			}
		} else {
			// Increment redeemed_count without flipping status yet.
			result := tx.Model(&Redemption{}).
				Where("id = ? AND status = ?", redemption.Id, common.RedemptionCodeStatusEnabled).
				Updates(map[string]interface{}{
					"redeemed_time":  now,
					"used_user_id":  userId,
					"redeemed_count": newCount,
				})
			if result.Error != nil {
				return result.Error
			}
			if result.RowsAffected == 0 {
				return errors.New("该兑换码已被使用")
			}
		}
		// Redemption codes are gift/promo, not cash top-up.
		return tx.Model(&User{}).Where("id = ?", userId).Updates(map[string]any{
			"quota":       gorm.Expr("quota + ?", redemption.Quota),
			"promo_quota": gorm.Expr("promo_quota + ?", redemption.Quota),
		}).Error
	})
	if err != nil {
		common.SysError("redemption failed: " + err.Error())
		return 0, ErrRedeemFailed
	}
	RecordLog(userId, LogTypeTopup, fmt.Sprintf("通过兑换码充值 %s，兑换码ID %d", logger.LogQuota(redemption.Quota), redemption.Id))
	return redemption.Quota, nil
}

// IsRedemptionKeyTaken reports whether any redemption (including soft-deleted
// ones) already uses the given key. The unique index on `key` covers soft-deleted
// rows too, so we must check with Unscoped to avoid a duplicate-key insert when
// re-creating a key that was previously deleted.
func IsRedemptionKeyTaken(key string) (bool, error) {
	if key == "" {
		return false, nil
	}
	keyCol := "`key`"
	if common.UsingMainDatabase(common.DatabaseTypePostgreSQL) {
		keyCol = `"key"`
	}
	var count int64
	err := DB.Unscoped().Model(&Redemption{}).Where(keyCol+" = ?", key).Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (redemption *Redemption) Insert() error {
	var err error
	err = DB.Create(redemption).Error
	return err
}

func (redemption *Redemption) SelectUpdate() error {
	// This can update zero values
	return DB.Model(redemption).Select("redeemed_time", "status").Updates(redemption).Error
}

// Update Make sure your token's fields is completed, because this will update non-zero values
func (redemption *Redemption) Update() error {
	var err error
	err = DB.Model(redemption).Select("name", "status", "quota", "currency", "amount", "max_redeems", "redeemed_time", "expired_time").Updates(redemption).Error
	return err
}

func (redemption *Redemption) Delete() error {
	var err error
	err = DB.Delete(redemption).Error
	return err
}

func DeleteRedemptionById(id int) (err error) {
	if id == 0 {
		return errors.New("id 为空！")
	}
	redemption := Redemption{Id: id}
	err = DB.Where(redemption).First(&redemption).Error
	if err != nil {
		return err
	}
	return redemption.Delete()
}

func DeleteInvalidRedemptions() (int64, error) {
	now := common.GetTimestamp()
	result := DB.Where("status IN ? OR (status = ? AND expired_time != 0 AND expired_time < ?)", []int{common.RedemptionCodeStatusUsed, common.RedemptionCodeStatusDisabled}, common.RedemptionCodeStatusEnabled, now).Delete(&Redemption{})
	return result.RowsAffected, result.Error
}
