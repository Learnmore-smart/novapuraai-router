package emaildelivery

import (
	"fmt"
	"html"
	"html/template"
	"strings"
)

const notificationValuePlaceholder = "{{value}}"

var (
	verificationHTMLTemplate = template.Must(template.New("verification-email").Parse(
		"<p>您好，你正在进行{{.SystemName}}邮箱验证。</p>" +
			"<p>您的验证码为: <strong>{{.Code}}</strong></p>" +
			"<p>验证码 {{.ValidMinutes}} 分钟内有效，如果不是本人操作，请忽略。</p>",
	))
	passwordResetHTMLTemplate = template.Must(template.New("password-reset-email").Parse(
		"<p>您好，你正在进行{{.SystemName}}密码重置。</p>" +
			"<p>点击 <a href=\"{{.ResetURL}}\">此处</a> 进行密码重置。</p>" +
			"<p>如果链接无法点击，请尝试点击下面的链接或将其复制到浏览器中打开：<br> {{.ResetURL}} </p>" +
			"<p>重置链接 {{.ValidMinutes}} 分钟内有效，如果不是本人操作，请忽略。</p>",
	))
	receiptHTMLTemplate = template.Must(template.New("receipt-email").Parse(
		"<p>Thank you for your payment to {{.SystemName}}.</p>" +
			"<p>Receipt: <strong>{{.ReceiptID}}</strong></p>" +
			"<p>Amount: {{.Amount}}<br>Paid at: {{.PaidAt}}</p>",
	))
)

type VerificationTemplateData struct {
	SystemName   string
	Recipient    string
	Code         string
	ValidMinutes int
}

type PasswordResetTemplateData struct {
	SystemName   string
	Recipient    string
	ResetURL     string
	ValidMinutes int
}

type ReceiptTemplateData struct {
	SystemName string
	Recipient  string
	ReceiptID  string
	Amount     string
	PaidAt     string
}

type NotificationTemplateData struct {
	Recipient string
	Title     string
	Content   string
	Values    []any
}

func BuildVerificationMessage(data VerificationTemplateData) (Message, error) {
	htmlBody, err := renderHTML(verificationHTMLTemplate, data)
	if err != nil {
		return Message{}, err
	}
	return Message{
		Type:     MessageTypeVerification,
		To:       data.Recipient,
		Subject:  fmt.Sprintf("%s邮箱验证邮件", data.SystemName),
		HTMLBody: htmlBody,
		TextBody: fmt.Sprintf("您好，你正在进行%s邮箱验证。您的验证码为: %s。验证码 %d 分钟内有效，如果不是本人操作，请忽略。", data.SystemName, data.Code, data.ValidMinutes),
	}, nil
}

func BuildPasswordResetMessage(data PasswordResetTemplateData) (Message, error) {
	htmlBody, err := renderHTML(passwordResetHTMLTemplate, data)
	if err != nil {
		return Message{}, err
	}
	return Message{
		Type:     MessageTypePasswordReset,
		To:       data.Recipient,
		Subject:  fmt.Sprintf("%s密码重置", data.SystemName),
		HTMLBody: htmlBody,
		TextBody: fmt.Sprintf("您好，你正在进行%s密码重置。请打开以下链接：%s。重置链接 %d 分钟内有效，如果不是本人操作，请忽略。", data.SystemName, data.ResetURL, data.ValidMinutes),
	}, nil
}

func BuildReceiptMessage(data ReceiptTemplateData) (Message, error) {
	htmlBody, err := renderHTML(receiptHTMLTemplate, data)
	if err != nil {
		return Message{}, err
	}
	return Message{
		Type:     MessageTypeReceipt,
		To:       data.Recipient,
		Subject:  fmt.Sprintf("%s payment receipt", data.SystemName),
		HTMLBody: htmlBody,
		TextBody: fmt.Sprintf("Thank you for your payment to %s. Receipt: %s. Amount: %s. Paid at: %s.", data.SystemName, data.ReceiptID, data.Amount, data.PaidAt),
	}, nil
}

func BuildNotificationMessage(data NotificationTemplateData) Message {
	htmlBody := data.Content
	textBody := data.Content
	for _, value := range data.Values {
		plain := fmt.Sprintf("%v", value)
		htmlBody = strings.Replace(htmlBody, notificationValuePlaceholder, html.EscapeString(plain), 1)
		textBody = strings.Replace(textBody, notificationValuePlaceholder, plain, 1)
	}
	return Message{
		Type:     MessageTypeNotification,
		To:       data.Recipient,
		Subject:  data.Title,
		HTMLBody: htmlBody,
		TextBody: textBody,
	}
}

func renderHTML(emailTemplate *template.Template, data any) (string, error) {
	var body strings.Builder
	if err := emailTemplate.Execute(&body, data); err != nil {
		return "", err
	}
	return body.String(), nil
}
