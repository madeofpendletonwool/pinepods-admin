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
        .app-link { background-color: #4CAF50; color: white; padding: 15px; text-align: center; border-radius: 8px; margin: 20px 0; }
        .app-link a { color: white; text-decoration: none; font-size: 18px; font-weight: bold; }
        .links { background-color: #E3F2FD; padding: 15px; border-left: 4px solid #2196F3; margin: 20px 0; }
        .links a { color: #1976D2; text-decoration: none; font-weight: bold; }
        .links a:hover { text-decoration: underline; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>🎉 Welcome to PinePods Internal Testing!</h1>
        </div>
        <div class="content">
            <p>Hi {{index .Submission.Data "name"}},</p>
            
            <p><strong>Congratulations! You've been added to PinePods internal testing.</strong> This email is your invitation to access the beta version of PinePods.</p>
            
            {{if eq (index .Submission.Data "platform") "ios"}}
            <div class="app-link" style="background-color: #007AFF;">
                <a href="https://testflight.apple.com/join/8Aax2BcX" target="_blank" style="color: white; text-decoration: none; display: block;">
                    <p style="color: white; margin: 0; font-size: 18px; font-weight: bold;">
                        🍎 Join PinePods iOS TestFlight Beta!
                    </p>
                    <p style="color: white; margin: 8px 0 0 0; font-size: 14px;">
                        Click here to install PinePods Beta via TestFlight
                    </p>
                </a>
            </div>
            {{else}}
            <div class="app-link">
                <a href="https://play.google.com/apps/internaltest/4701694943577309896" target="_blank">
                    📱 Download PinePods Beta from Google Play
                </a>
            </div>
            {{end}}
            
            <div class="highlight">
                <h3>🚀 What you get with internal testing:</h3>
                <ul>
                    <li><strong>Early Access:</strong> Latest features before public release</li>
                    <li><strong>Direct Feedback Channel:</strong> Your input shapes development</li>
                    <li><strong>Beta Versions:</strong> Test cutting-edge functionality</li>
                    <li><strong>Community Access:</strong> Connect with other testers and developers</li>
                </ul>
            </div>
            
            <div class="links">
                <h3>🔗 Important Links:</h3>
                <ul>
                    <li><strong>Discord Community:</strong> <a href="https://discord.com/invite/bKzHRa4GNc" target="_blank">Join our Discord server</a> for real-time chat, support, and direct communication with the dev team</li>
                    <li><strong>Report Issues:</strong> Found a bug or have feedback? <a href="https://github.com/madeofpendletonwool/PinePods/issues" target="_blank">Submit an issue on GitHub</a></li>
                </ul>
            </div>
            
            <h3>📋 How to Report Mobile App Issues:</h3>
            <ol>
                <li>Visit our <a href="https://github.com/madeofpendletonwool/PinePods/issues" target="_blank">GitHub Issues page</a></li>
                <li>Click "New issue" and select the appropriate template</li>
                <li><strong>Include "[MOBILE]" in your issue title</strong> to help us identify mobile-specific problems</li>
                <li>Provide details about:
                    <ul>
                        <li>Your device model and Android version</li>
                        <li>PinePods version number (found in app settings)</li>
                        <li>Steps to reproduce the issue</li>
                        <li>Screenshots if applicable</li>
                    </ul>
                </li>
            </ol>
            
            <h3>⚠️ Important Notes:</h3>
            <ul>
                {{if eq (index .Submission.Data "platform") "ios"}}
                <li>You must use the email <strong>{{index .Submission.Data "email"}}</strong> to access TestFlight (this should be your Apple ID email)</li>
                <li>We'll send you the TestFlight invitation as soon as iOS testing is ready</li>
                {{else}}
                <li>You must use the email <strong>{{index .Submission.Data "email"}}</strong> to access the beta through Google Play</li>
                {{end}}
                <li>Beta versions may contain bugs - that's why we need your feedback!</li>
                <li>Updates are frequent, so check for new versions regularly</li>
                <li>Join our Discord for announcements about new beta releases</li>
            </ul>
            
            <p>Thank you for helping us make PinePods better! Your feedback and testing are invaluable to our development process.</p>
            
            <p>Happy podcasting!</p>
            
            <p>Best regards,<br>The PinePods Development Team</p>
        </div>
        <div class="footer">
            <p>🎧 PinePods - Your Personal Podcast Experience</p>
            <p><a href="https://discord.com/invite/bKzHRa4GNc">Discord</a> • <a href="https://github.com/madeofpendletonwool/PinePods">GitHub</a> • <a href="https://docs.pinepods.online">Documentation</a></p>
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
	
	// Check if we need authentication (for production SMTP servers)
	var auth smtp.Auth
	if es.config.Email.SMTP.Username != "" && es.config.Email.SMTP.Password != "" {
		auth = smtp.PlainAuth("",
			es.config.Email.SMTP.Username,
			es.config.Email.SMTP.Password,
			es.config.Email.SMTP.Host,
		)
	}
	// For MailHog and other test servers, auth can be nil

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

// SendWelcomeEmail sends a welcome email for internal testing after manual approval
func (es *EmailService) SendWelcomeEmail(submission *models.FormSubmission, formConfig config.FormConfig, email string) error {
	// Use the internal-testing email template
	body, err := es.renderEmailTemplate("internal-testing", EmailData{
		Submission: submission,
		FormConfig: formConfig,
	})
	if err != nil {
		return fmt.Errorf("failed to render welcome email template: %w", err)
	}

	emailData := EmailData{
		To:      email,
		Subject: "🎉 Welcome to PinePods Internal Testing - You're In!",
		Body:    body,
		IsHTML:  true,
	}

	return es.sendEmail(emailData)
}