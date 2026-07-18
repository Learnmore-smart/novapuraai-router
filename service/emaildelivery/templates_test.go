package emaildelivery

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVerificationTemplateIsProviderIndependentAndEscapesValues(t *testing.T) {
	message, err := BuildVerificationMessage(VerificationTemplateData{
		SystemName:   `<script>alert("x")</script>`,
		Recipient:    "recipient@example.com",
		Code:         "123456",
		ValidMinutes: 10,
	})

	require.NoError(t, err)
	assert.Equal(t, MessageTypeVerification, message.Type)
	assert.Equal(t, "recipient@example.com", message.To)
	assert.Contains(t, message.Subject, "邮箱验证邮件")
	assert.Contains(t, message.HTMLBody, "123456")
	assert.NotContains(t, message.HTMLBody, "<script>")
	assert.Contains(t, message.TextBody, "123456")
}

func TestPasswordResetTemplateContainsEscapedLinkAndPlainTextFallback(t *testing.T) {
	message, err := BuildPasswordResetMessage(PasswordResetTemplateData{
		SystemName:   "Example",
		Recipient:    "recipient@example.com",
		ResetURL:     `https://example.com/reset?a=1&b=2`,
		ValidMinutes: 10,
	})

	require.NoError(t, err)
	assert.Equal(t, MessageTypePasswordReset, message.Type)
	assert.Contains(t, message.HTMLBody, "a=1&amp;b=2")
	assert.Contains(t, message.TextBody, "https://example.com/reset?a=1&b=2")
}

func TestReceiptAndNotificationTemplatesShareProviderNeutralMessage(t *testing.T) {
	receipt, err := BuildReceiptMessage(ReceiptTemplateData{
		SystemName: "Example",
		Recipient:  "recipient@example.com",
		ReceiptID:  "receipt-17",
		Amount:     "$12.00",
		PaidAt:     "2026-07-18",
	})
	require.NoError(t, err)
	assert.Equal(t, MessageTypeReceipt, receipt.Type)
	assert.Contains(t, receipt.HTMLBody, "receipt-17")
	assert.Contains(t, receipt.TextBody, "$12.00")

	notification := BuildNotificationMessage(NotificationTemplateData{
		Recipient: "recipient@example.com",
		Title:     "Alert",
		Content:   "Quota: {{value}}",
		Values:    []any{`<script>alert("x")</script>`},
	})
	assert.Equal(t, MessageTypeNotification, notification.Type)
	assert.Contains(t, notification.HTMLBody, "&lt;script&gt;")
	assert.NotContains(t, notification.HTMLBody, "<script>")
}
