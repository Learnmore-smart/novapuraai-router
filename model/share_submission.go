package model

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	ShareStatusPending  = "pending"
	ShareStatusApproved = "approved"
	ShareStatusRejected = "rejected"

	shareSubmissionActiveKey = "active"
)

// ShareSubmission is the social-share reward queue (MVP §9.5).
type ShareSubmission struct {
	Id           int     `json:"id" gorm:"primaryKey"`
	UserId       int     `json:"user_id" gorm:"index;uniqueIndex:uk_share_user_active,priority:1"`
	URL          string  `json:"url" gorm:"type:varchar(1024);not null"`
	Platform     string  `json:"platform" gorm:"type:varchar(64)"`
	Note         string  `json:"note" gorm:"type:varchar(512)"`
	Status       string  `json:"status" gorm:"type:varchar(32);index"`
	ActiveKey    *string `json:"-" gorm:"type:varchar(16);uniqueIndex:uk_share_user_active,priority:2"`
	ReviewerId   int     `json:"reviewer_id" gorm:"type:int;default:0"`
	ReviewReason string  `json:"review_reason" gorm:"type:varchar(512)"`
	Amount       int     `json:"amount" gorm:"type:int;default:0"`
	CreatedAt    int64   `json:"created_at" gorm:"autoCreateTime"`
	ReviewedAt   int64   `json:"reviewed_at" gorm:"default:0"`
}

func (ShareSubmission) TableName() string { return "share_submissions" }

func ensureShareSubmissionIndexes() error {
	if DB.Migrator().HasIndex(&ShareSubmission{}, "uk_share_user_pending") {
		if err := DB.Migrator().DropIndex(&ShareSubmission{}, "uk_share_user_pending"); err != nil {
			return err
		}
	}
	return DB.Model(&ShareSubmission{}).
		Where("active_key IS NULL AND status IN ?", []string{ShareStatusPending, ShareStatusApproved}).
		Update("active_key", shareSubmissionActiveKey).Error
}

// ValidateShareURL rejects non-http(s), oversized, and obvious private-host SSRF targets.
func ValidateShareURL(raw string) error {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return errors.New("url is required")
	}
	if len(raw) > 1024 {
		return errors.New("url too long")
	}
	u, err := url.Parse(raw)
	if err != nil {
		return errors.New("invalid url")
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return errors.New("url must be http or https")
	}
	host := u.Hostname()
	if host == "" {
		return errors.New("url host required")
	}
	// Block localhost / private IPs if host is literal IP
	if ip := net.ParseIP(host); ip != nil {
		if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsUnspecified() {
			return errors.New("url host not allowed")
		}
	}
	lower := strings.ToLower(host)
	if lower == "localhost" || strings.HasSuffix(lower, ".local") || strings.HasSuffix(lower, ".internal") {
		return errors.New("url host not allowed")
	}
	return nil
}

// CreateShareSubmission creates a pending share review item (one pending/approved per user).
func CreateShareSubmission(userId int, rawURL, platform, note string) (*ShareSubmission, error) {
	if userId <= 0 {
		return nil, errors.New("invalid user")
	}
	if err := ValidateShareURL(rawURL); err != nil {
		return nil, err
	}
	platform = strings.TrimSpace(platform)
	note = strings.TrimSpace(note)
	if len(platform) > 64 {
		return nil, errors.New("platform too long")
	}
	if len(note) > 512 {
		return nil, errors.New("note too long")
	}
	// Already has pending or approved?
	var count int64
	if err := DB.Model(&ShareSubmission{}).
		Where("user_id = ? AND status IN ?", userId, []string{ShareStatusPending, ShareStatusApproved}).
		Count(&count).Error; err != nil {
		return nil, err
	}
	if count > 0 {
		return nil, errors.New("already submitted share reward for this campaign")
	}
	amount := common.CNYYuanToQuota(common.ShareRewardCNYYuan, exchangeRateForCampaign())
	if amount <= 0 {
		return nil, errors.New("share reward campaign is unavailable")
	}
	s := &ShareSubmission{
		UserId:   userId,
		URL:      strings.TrimSpace(rawURL),
		Platform: platform,
		Note:     note,
		Status:   ShareStatusPending,
		Amount:   amount,
	}
	activeKey := shareSubmissionActiveKey
	s.ActiveKey = &activeKey
	if err := DB.Create(s).Error; err != nil {
		return nil, err
	}
	return s, nil
}

// ReviewShareSubmission approves or rejects; approve grants promo once via campaign claim.
func ReviewShareSubmission(id, reviewerId int, approve bool, reason string) error {
	if id <= 0 || reviewerId <= 0 {
		return errors.New("invalid review request")
	}
	reason = strings.TrimSpace(reason)
	if len(reason) > 512 {
		return errors.New("review reason too long")
	}
	if reason == "" {
		return errors.New("review reason is required")
	}

	var rewarded bool
	var rewardedUserId int
	var rewardedAmount int
	err := DB.Transaction(func(tx *gorm.DB) error {
		var s ShareSubmission
		if err := lockForUpdate(tx).First(&s, id).Error; err != nil {
			return err
		}
		if s.Status != ShareStatusPending {
			return errors.New("submission is not pending")
		}
		now := time.Now().Unix()
		if !approve {
			return tx.Model(&s).Updates(map[string]any{
				"status":        ShareStatusRejected,
				"active_key":    nil,
				"reviewer_id":   reviewerId,
				"review_reason": reason,
				"reviewed_at":   now,
			}).Error
		}
		// Idempotent promo grant inside same transaction
		var existing CampaignClaim
		if e := tx.Where("user_id = ? AND kind = ?", s.UserId, CampaignKindShare).First(&existing).Error; e == nil {
			// already rewarded
		} else if errors.Is(e, gorm.ErrRecordNotFound) {
			if s.Amount <= 0 {
				return errors.New("share reward amount is invalid")
			}
			claimResult := tx.Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "user_id"}, {Name: "kind"}},
				DoNothing: true,
			}).Create(&CampaignClaim{
				UserId: s.UserId,
				Kind:   CampaignKindShare,
				Amount: s.Amount,
			})
			if claimResult.Error != nil {
				return claimResult.Error
			}
			if claimResult.RowsAffected > 0 {
				if err := tx.Model(&User{}).Where("id = ?", s.UserId).Updates(map[string]any{
					"quota":       gorm.Expr("quota + ?", s.Amount),
					"promo_quota": gorm.Expr("promo_quota + ?", s.Amount),
				}).Error; err != nil {
					return err
				}
				rewarded = true
				rewardedUserId = s.UserId
				rewardedAmount = s.Amount
			}
		} else {
			return e
		}
		if err := tx.Model(&s).Updates(map[string]any{
			"status":        ShareStatusApproved,
			"reviewer_id":   reviewerId,
			"review_reason": reason,
			"reviewed_at":   now,
		}).Error; err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return err
	}
	if rewarded {
		RecordLog(rewardedUserId, LogTypeSystem, fmt.Sprintf("Share reward %s", logger.LogQuota(rewardedAmount)))
		_ = invalidateUserCache(rewardedUserId)
	}
	return nil
}

func ListShareSubmissions(status string, start, size int) ([]*ShareSubmission, int64, error) {
	q := DB.Model(&ShareSubmission{})
	if status != "" {
		q = q.Where("status = ?", status)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []*ShareSubmission
	err := q.Order("id desc").Offset(start).Limit(size).Find(&rows).Error
	return rows, total, err
}

func ListUserShareSubmissions(userId int) ([]*ShareSubmission, error) {
	var rows []*ShareSubmission
	err := DB.Where("user_id = ?", userId).Order("id desc").Find(&rows).Error
	return rows, err
}

// Admin note helper
func FormatShareReviewLog(s *ShareSubmission, approve bool) string {
	action := "reject"
	if approve {
		action = "approve"
	}
	return fmt.Sprintf("share #%d %s user=%d amount=%s", s.Id, action, s.UserId, logger.LogQuota(s.Amount))
}
