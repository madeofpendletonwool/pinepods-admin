package services

import (
	"context"
	"fmt"
	"time"

	"github.com/madeofpendletonwool/pinepods-admin/internal/config"
	"github.com/madeofpendletonwool/pinepods-admin/internal/models"
	"google.golang.org/api/androidpublisher/v3"
	"google.golang.org/api/option"
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
		Success:    false,
	}

	// Check if Google Play is configured
	fmt.Printf("[DEBUG] Google Play Config - ServiceAccountFile: '%s', PackageName: '%s'\n", 
		as.config.GooglePlay.ServiceAccountFile, as.config.GooglePlay.PackageName)
	
	if as.config.GooglePlay.ServiceAccountFile == "" || as.config.GooglePlay.PackageName == "" {
		result.Error = fmt.Sprintf("Google Play Console not configured (ServiceAccountFile: '%s', PackageName: '%s')", 
			as.config.GooglePlay.ServiceAccountFile, as.config.GooglePlay.PackageName)
		result.Message = "Google Play Console configuration missing"
		return result
	}

	// Extract email from submission
	emailService := NewEmailService(as.config)
	email := emailService.GetEmailFromSubmission(submission)
	if email == "" {
		result.Error = "No email address found in submission"
		result.Message = "Email address required for Google Play testing"
		return result
	}

	// Get track from action config (default to "internal")
	track := "internal"
	if trackConfig, exists := actionConfig.Config["track"]; exists {
		if trackStr, ok := trackConfig.(string); ok {
			track = trackStr
		}
	}

	// Initialize Google Play API client
	ctx := context.Background()
	service, err := androidpublisher.NewService(ctx, option.WithCredentialsFile(as.config.GooglePlay.ServiceAccountFile))
	if err != nil {
		result.Error = fmt.Sprintf("Failed to initialize Google Play API: %v", err)
		result.Message = "Google Play API initialization failed"
		return result
	}

	packageName := as.config.GooglePlay.PackageName

	// For internal testing, we need to add the user to the tester list
	// Internal testing uses a different API endpoint - we need to use the Internal Testing API
	
	// Create a new edit for managing testers
	edit := &androidpublisher.AppEdit{}
	editCall := service.Edits.Insert(packageName, edit)
	insertedEdit, err := editCall.Do()
	if err != nil {
		result.Error = fmt.Sprintf("Failed to create edit: %v", err)
		result.Message = "Failed to create Google Play edit"
		return result
	}

	editId := insertedEdit.Id

	// Get existing testers for the internal track
	existingTesters, err := service.Edits.Testers.Get(packageName, editId, track).Do()
	if err != nil {
		// If no testers exist yet, create empty list
		existingTesters = &androidpublisher.Testers{
			GoogleGroups: []string{},
		}
	}

	// Add the new tester email to the existing list
	// For internal testing, individual emails are added directly
	newTesterList := existingTesters.GoogleGroups
	
	// Check if email is already in the list (avoid duplicates)
	emailAlreadyExists := false
	for _, existingEmail := range newTesterList {
		if existingEmail == email {
			emailAlreadyExists = true
			break
		}
	}
	
	if !emailAlreadyExists {
		newTesterList = append(newTesterList, email)
	}

	// Update the testers list
	updatedTesters := &androidpublisher.Testers{
		GoogleGroups: newTesterList,
	}

	_, err = service.Edits.Testers.Update(packageName, editId, track, updatedTesters).Do()
	if err != nil {
		result.Error = fmt.Sprintf("Failed to add tester: %v", err)
		result.Message = "Failed to add user to internal testing"
		
		// Clean up the edit
		service.Edits.Delete(packageName, editId).Do()
		return result
	}

	// Commit the edit
	_, err = service.Edits.Commit(packageName, editId).Do()
	if err != nil {
		result.Error = fmt.Sprintf("Failed to commit edit: %v", err)
		result.Message = "Failed to commit Google Play changes"
		return result
	}

	result.Success = true
	result.Message = fmt.Sprintf("Successfully added %s to %s testing track", email, track)

	// Send confirmation email if configured
	if formConfig, exists := as.config.Forms.Forms[submission.FormID]; exists {
		if formConfig.Email.Enabled && formConfig.Email.SendConfirmation {
			emailService := NewEmailService(as.config)
			if err := emailService.SendConfirmationEmail(submission, formConfig); err != nil {
				// Don't fail the action if email fails, just log it
				result.Message += fmt.Sprintf(" (Note: Confirmation email failed: %v)", err)
			}
		}
	}

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

	// Send email
	emailService := NewEmailService(as.config)
	err := emailService.SendConfirmationEmail(submission, formConfig)
	if err != nil {
		result.Error = err.Error()
		result.Message = "Failed to send email"
		return result
	}

	result.Success = true
	result.Message = "Email sent successfully"
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