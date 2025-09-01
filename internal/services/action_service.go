package services

import (
	"fmt"
	"time"

	"github.com/madeofpendletonwool/pinepods-admin/internal/config"
	"github.com/madeofpendletonwool/pinepods-admin/internal/models"
)

type ActionService struct {
	config *config.Config
}

func NewActionService(cfg *config.Config) *ActionService {
	return &ActionService{
		config: cfg,
	}
}

func (as *ActionService) ProcessActions(submission *models.FormSubmission, formConfig config.FormConfig) *models.ProcessingResult {
	result := &models.ProcessingResult{
		SubmissionID: submission.ID,
		FormID:       submission.FormID,
		Success:      true,
		ProcessedAt:  time.Now().UTC(),
		Actions:      []models.ActionResult{},
	}

	// Process each action defined in the form config
	for _, actionConfig := range formConfig.Actions {
		actionResult := as.executeAction(submission, actionConfig)
		result.Actions = append(result.Actions, actionResult)
		
		// If any action fails, mark the overall result as failed
		if !actionResult.Success {
			result.Success = false
		}
	}

	return result
}

func (as *ActionService) executeAction(submission *models.FormSubmission, actionConfig config.ActionConfig) models.ActionResult {
	switch actionConfig.Type {
	case "google_play_add_tester":
		return as.addGooglePlayTester(submission, actionConfig)
	case "send_email":
		return as.sendEmailAction(submission, actionConfig)
	case "send_feedback_email":
		return as.sendFeedbackEmail(submission, actionConfig)
	case "webhook":
		return as.sendWebhook(submission, actionConfig)
	case "log":
		return as.logAction(submission, actionConfig)
	default:
		return models.ActionResult{
			ActionType: actionConfig.Type,
			Success:    false,
			Message:    "Unknown action type",
			Error:      fmt.Sprintf("Action type '%s' is not supported", actionConfig.Type),
		}
	}
}

func (as *ActionService) addGooglePlayTester(submission *models.FormSubmission, actionConfig config.ActionConfig) models.ActionResult {
	result := models.ActionResult{
		ActionType: "google_play_add_tester",
		Success:    true,
		Message:    "Tester details logged for manual addition to Google Play Console",
	}

	// Extract email from submission for logging
	emailService := NewEmailService(as.config)
	email := emailService.GetEmailFromSubmission(submission)
	if email == "" {
		result.Success = false
		result.Error = "No email address found in submission"
		result.Message = "Email address required for Google Play testing"
		return result
	}

	// Log the submission details - manual addition required
	fmt.Printf("[MANUAL ACTION REQUIRED] Add %s to Google Play Internal Testing\n", email)

	return result
}

func (as *ActionService) sendEmailAction(submission *models.FormSubmission, actionConfig config.ActionConfig) models.ActionResult {
	result := models.ActionResult{
		ActionType: "send_email",
		Success:    false,
	}

	// Get form config
	formConfig, exists := as.config.Forms.Forms[submission.FormID]
	if !exists {
		result.Error = "Form configuration not found"
		result.Message = "Cannot send email: form config missing"
		return result
	}

	emailService := NewEmailService(as.config)

	// Check if this is iOS internal testing - send welcome email immediately
	if submission.FormID == "internal-testing-signup" {
		// Check platform
		platform, platformExists := submission.Data["platform"]
		if platformExists && platform == "ios" {
			// For iOS, send welcome email immediately (no manual approval needed)
			email := emailService.GetEmailFromSubmission(submission)
			if email == "" {
				fmt.Printf("[ERROR] iOS welcome email failed for submission %s: No email address found\n", submission.ID[:8])
				result.Error = "No email address found in submission"
				result.Message = "Cannot send email: email address missing"
				return result
			}

			fmt.Printf("[DEBUG] Attempting to send iOS welcome email to %s for submission %s\n", email, submission.ID[:8])
			err := emailService.SendWelcomeEmail(submission, formConfig, email)
			if err != nil {
				fmt.Printf("[ERROR] iOS welcome email failed for submission %s to %s: %v\n", submission.ID[:8], email, err)
				result.Error = err.Error()
				result.Message = "Failed to send welcome email"
				return result
			}

			fmt.Printf("[SUCCESS] iOS welcome email sent automatically to %s for submission %s (external app access)\n", email, submission.ID[:8])
			result.Success = true
			result.Message = "iOS welcome email sent immediately (external app access)"
			return result
		}
	}

	// For Android and other platforms, send regular confirmation email
	email := emailService.GetEmailFromSubmission(submission)
	fmt.Printf("[DEBUG] Attempting to send confirmation email to %s for submission %s (platform: %v)\n", email, submission.ID[:8], submission.Data["platform"])
	
	err := emailService.SendConfirmationEmail(submission, formConfig)
	if err != nil {
		fmt.Printf("[ERROR] Confirmation email failed for submission %s to %s: %v\n", submission.ID[:8], email, err)
		result.Error = err.Error()
		result.Message = "Failed to send email"
		return result
	}

	fmt.Printf("[SUCCESS] Confirmation email sent to %s for submission %s\n", email, submission.ID[:8])
	result.Success = true
	result.Message = "Email sent successfully"
	return result
}

func (as *ActionService) sendFeedbackEmail(submission *models.FormSubmission, actionConfig config.ActionConfig) models.ActionResult {
	result := models.ActionResult{
		ActionType: "send_feedback_email",
		Success:    false,
	}

	if as.config.Feedback.RecipientEmail == "" {
		result.Error = "Feedback recipient email not configured"
		result.Message = "Cannot send feedback: FEEDBACK_EMAIL environment variable not set"
		return result
	}

	emailService := NewEmailService(as.config)
	err := emailService.SendFeedbackNotification(submission, as.config.Feedback.RecipientEmail)
	if err != nil {
		result.Error = err.Error()
		result.Message = "Failed to send feedback notification"
		return result
	}

	result.Success = true
	result.Message = fmt.Sprintf("Feedback notification sent to %s", as.config.Feedback.RecipientEmail)
	return result
}

func (as *ActionService) sendWebhook(submission *models.FormSubmission, actionConfig config.ActionConfig) models.ActionResult {
	result := models.ActionResult{
		ActionType: "webhook",
		Success:    false,
	}

	// Webhook implementation would go here
	// This is a placeholder for now
	result.Error = "Webhook action not yet implemented"
	result.Message = "Webhook functionality coming soon"
	
	return result
}

func (as *ActionService) logAction(submission *models.FormSubmission, actionConfig config.ActionConfig) models.ActionResult {
	result := models.ActionResult{
		ActionType: "log",
		Success:    true,
		Message:    "Action logged successfully",
	}

	// Get log message from config or use default
	logMessage := "Form submission processed"
	if msgConfig, exists := actionConfig.Config["message"]; exists {
		if msgStr, ok := msgConfig.(string); ok {
			logMessage = msgStr
		}
	}

	// Log the submission (in a real implementation, you might want to use a proper logger)
	fmt.Printf("[LOG ACTION] %s - %s - Form: %s, Submission: %s, Data: %+v\n", 
		time.Now().Format(time.RFC3339), 
		logMessage,
		submission.FormID, 
		submission.ID, 
		submission.Data)

	return result
}