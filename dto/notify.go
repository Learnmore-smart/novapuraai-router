package dto

import "github.com/google/uuid"

type Notify struct {
	EventID string        `json:"-"`
	Type    string        `json:"type"`
	Title   string        `json:"title"`
	Content string        `json:"content"`
	Values  []interface{} `json:"values"`
}

const ContentValueParam = "{{value}}"

const (
	NotifyTypeQuotaExceed     = "quota_exceed"
	NotifyTypeChannelUpdate   = "channel_update"
	NotifyTypeChannelTest     = "channel_test"
	NotifyTypeDeepSeekFairUse = "deepseek_fair_use"
)

func NewNotify(t string, title string, content string, values []interface{}) Notify {
	return Notify{
		EventID: uuid.NewString(),
		Type:    t,
		Title:   title,
		Content: content,
		Values:  values,
	}
}
