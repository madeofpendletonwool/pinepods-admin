package services

import (
	"bytes"
	"fmt"
	"html/template"
	"net/smtp"
	"strings"

	"github.com/madeofpendletonwool/pinepods-admin/internal/config"
	"github.com/madeofpendletonwool/pinepods-admin/internal/models"
)

type EmailService struct {
	config *config.Config
}

func NewEmailService(cfg *config.Config) *EmailService {
	return &EmailService{
		config: cfg,
	}
}

type EmailData struct {
	To          string
	Subject     string
	Body        string
	IsHTML      bool
	Submission  *models.FormSubmission
	FormConfig  config.FormConfig
}

func (es *EmailService) SendConfirmationEmail(submission *models.FormSubmission, formConfig config.FormConfig) error {
	if !formConfig.Email.Enabled || !formConfig.Email.SendConfirmation {
		return nil
	}

	// Get recipient email from submission data
	var recipientEmail string
	if email, exists := submission.Data["email"]; exists {
		if emailStr, ok := email.(string); ok {
			recipientEmail = emailStr
		}
	}

	if recipientEmail == "" {
		return fmt.Errorf("no email address found in submission data")
	}

	// Prepare email data
	emailData := EmailData{
		To:         recipientEmail,
		Subject:    formConfig.Email.Subject,
		Submission: submission,
		FormConfig: formConfig,
		IsHTML:     true,
	}

	// Generate email body from template
	body, err := es.renderEmailTemplate(formConfig.Email.Template, emailData)
	if err != nil {
		return fmt.Errorf("failed to render email template: %w", err)
	}
	emailData.Body = body

	return es.sendEmail(emailData)
}

func (es *EmailService) SendNotificationEmail(submission *models.FormSubmission, formConfig config.FormConfig, result *models.ProcessingResult) error {
	// This can be used to send admin notifications
	// Implementation would be similar to confirmation email
	return nil
}

func (es *EmailService) renderEmailTemplate(templateName string, data EmailData) (string, error) {
	// Default templates
	defaultTemplates := map[string]string{
		"confirmation": `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>{{.Subject}}</title>
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { background-color: #4CAF50; color: white; padding: 20px; text-align: center; }
        .content { padding: 20px; background-color: #f9f9f9; }
        .footer { padding: 20px; text-align: center; color: #666; }
        .data-table { width: 100%; border-collapse: collapse; margin: 20px 0; }
        .data-table th, .data-table td { border: 1px solid #ddd; padding: 8px; text-align: left; }
        .data-table th { background-color: #f2f2f2; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>{{.FormConfig.Name}}</h1>
            <p>Thank you for your submission!</p>
        </div>
        <div class="content">
            <p>Dear {{index .Submission.Data "name"}},</p>
            <p>We have successfully received your submission for {{.FormConfig.Name}}.</p>
            
            <h3>Submission Details:</h3>
            <table class="data-table">
                {{range $key, $value := .Submission.Data}}
                <tr>
                    <th>{{$key}}</th>
                    <td>{{$value}}</td>
                </tr>
                {{end}}
            </table>
            
            <p><strong>Submission ID:</strong> {{.Submission.ID}}</p>
            <p><strong>Submitted at:</strong> {{.Submission.SubmittedAt.Format "2006-01-02 15:04:05 UTC"}}</p>
            
            {{if .FormConfig.Description}}
            <p>{{.FormConfig.Description}}</p>
            {{end}}
        </div>
        <div class="footer">
            <p>This is an automated message. Please do not reply to this email.</p>
        </div>
    </div>
</body>
</html>`,
		"internal-testing": `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>Welcome to PinePods Internal Testing</title>
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { background-color: #2E7D32; color: white; padding: 20px; text-align: center; }
        .content { padding: 20px; background-color: #f9f9f9; }
        .footer { padding: 20px; text-align: center; color: #666; }
        .highlight { background-color: #E8F5E8; padding: 15px; border-left: 4px solid #4CAF50; margin: 20px 0; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>🎉 Welcome to PinePods Internal Testing!</h1>
        </div>
        <div class="content">
            <p>Hi {{index .Submission.Data "name"}},</p>
            
            <p>Thank you for signing up for PinePods internal testing! We're excited to have you as part of our testing community.</p>
            
            <div class="highlight">
                <h3>What happens next?</h3>
                <ul>
                    <li><strong>Google Play Console Invitation:</strong> You'll receive an email invitation from Google Play Console within the next 24 hours</li>
                    <li><strong>Early Access:</strong> Once you accept the invitation, you'll have access to beta versions of PinePods</li>
                    <li><strong>Feedback Channel:</strong> Your feedback will directly reach our development team</li>
                    <li><strong>New Features:</strong> Be the first to try new features before they're released to the public</li>
                </ul>
            </div>
            
            <h3>Important Notes:</h3>
            <ul>
                <li>Check your spam folder for the Google Play Console invitation</li>
                <li>The invitation will be sent to: <strong>{{index .Submission.Data "email"}}</strong></li>
                <li>Beta versions may have bugs - that's why we need your feedback!</li>
                <li>Join our Discord community for direct communication with the dev team</li>
            </ul>
            
            <p>If you have any questions or don't receive the invitation within 24 hours, feel free to reach out to us.</p>
            
            <p>Thanks again for helping make PinePods better!</p>
            
            <p>Best regards,<br>The PinePods Development Team</p>
        </div>
        <div class="footer">
            <p>PinePods - Your Personal Podcast Experience</p>
            <p>This is an automated message. For support, join our Discord or check our documentation.</p>
        </div>
    </div>
</body>
</html>`,
	}

	// Use default template if no custom template specified or found
	var templateContent string
	if templateName == "" {
		templateContent = defaultTemplates["confirmation"]
	} else if tmpl, exists := defaultTemplates[templateName]; exists {
		templateContent = tmpl
	} else {
		templateContent = defaultTemplates["confirmation"]
	}

	// Parse and execute template
	tmpl, err := template.New("email").Parse(templateContent)
	if err != nil {
		return "", fmt.Errorf("failed to parse email template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute email template: %w", err)
	}

	return buf.String(), nil
}

func (es *EmailService) sendEmail(emailData EmailData) error {
	switch es.config.Email.Provider {
	case "smtp":
		return es.sendSMTPEmail(emailData)
	case "sendgrid":
		return es.sendSendGridEmail(emailData)
	default:
		return fmt.Errorf("unsupported email provider: %s", es.config.Email.Provider)
	}
}

func (es *EmailService) sendSMTPEmail(emailData EmailData) error {
	// SMTP authentication
	auth := smtp.PlainAuth("",
		es.config.Email.SMTP.Username,
		es.config.Email.SMTP.Password,
		es.config.Email.SMTP.Host,
	)

	// Email headers and body
	var contentType string
	if emailData.IsHTML {
		contentType = "text/html; charset=UTF-8"
	} else {
		contentType = "text/plain; charset=UTF-8"
	}

	message := fmt.Sprintf(`To: %s
From: %s
Subject: %s
MIME-Version: 1.0
Content-Type: %s

%s`,
		emailData.To,
		es.config.Email.SMTP.From,
		emailData.Subject,
		contentType,
		emailData.Body,
	)

	// Send email
	addr := fmt.Sprintf("%s:%d", es.config.Email.SMTP.Host, es.config.Email.SMTP.Port)
	err := smtp.SendMail(addr, auth, es.config.Email.SMTP.From, []string{emailData.To}, []byte(message))
	if err != nil {
		return fmt.Errorf("failed to send SMTP email: %w", err)
	}

	return nil
}

func (es *EmailService) sendSendGridEmail(emailData EmailData) error {
	// SendGrid implementation would go here
	// For now, just return an error indicating it's not implemented
	return fmt.Errorf("SendGrid email provider not yet implemented")
}

func (es *EmailService) SendTestEmail(to string) error {
	emailData := EmailData{
		To:      to,
		Subject: "PinePods Forms - Test Email",
		Body:    "This is a test email from PinePods Forms service. If you received this, email sending is working correctly!",
		IsHTML:  false,
	}

	return es.sendEmail(emailData)
}

// GetEmailFromSubmission extracts email address from submission data
func (es *EmailService) GetEmailFromSubmission(submission *models.FormSubmission) string {
	// Try different common field names for email
	emailFields := []string{"email", "email_address", "Email", "Email_Address", "e_mail"}
	
	for _, field := range emailFields {
		if email, exists := submission.Data[field]; exists {
			if emailStr, ok := email.(string); ok && emailStr != "" {
				return strings.TrimSpace(emailStr)
			}
		}
	}
	
	return ""
}