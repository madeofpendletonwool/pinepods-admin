package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/madeofpendletonwool/pinepods-admin/internal/config"
	"github.com/madeofpendletonwool/pinepods-admin/internal/models"
)

type NotificationService struct {
	config *config.Config
}

func NewNotificationService(cfg *config.Config) *NotificationService {
	return &NotificationService{
		config: cfg,
	}
}

type NtfyMessage struct {
	Topic    string            `json:"topic"`
	Message  string            `json:"message"`
	Title    string            `json:"title"`
	Tags     []string          `json:"tags,omitempty"`
	Priority int               `json:"priority,omitempty"`
	Actions  []NtfyAction     `json:"actions,omitempty"`
	Click    string           `json:"click,omitempty"`
	Attach   string           `json:"attach,omitempty"`
	Filename string           `json:"filename,omitempty"`
	Email    string           `json:"email,omitempty"`
	Delay    string           `json:"delay,omitempty"`
}

type NtfyAction struct {
	Action string `json:"action"`
	Label  string `json:"label"`
	URL    string `json:"url,omitempty"`
	Method string `json:"method,omitempty"`
	Body   string `json:"body,omitempty"`
}

func (ns *NotificationService) SendSubmissionNotification(submission *models.FormSubmission, result *models.ProcessingResult) error {
	if !ns.config.Notifications.Ntfy.Enabled {
		return nil
	}

	// Get form config for better notification content
	formConfig, exists := ns.config.Forms.Forms[submission.FormID]
	formName := submission.FormID
	if exists {
		formName = formConfig.Name
	}

	// Create notification message
	var status string
	var tags []string
	var priority int
	
	if result.Success {
		status = "✅ Successfully processed"
		tags = []string{"white_check_mark", "forms"}
		priority = 3
	} else {
		status = "❌ Processing failed"
		tags = []string{"x", "forms", "error"}
		priority = 4
	}

	// Format submission data for display
	dataStr := ""
	for key, value := range submission.Data {
		if key == "platform" {
			platform := "Android"
			if value == "ios" {
				platform = "iOS (TestFlight)"
			}
			dataStr += fmt.Sprintf("%s: %s\n", key, platform)
		} else if key == "wantsNews" {
			wants := "No"
			if value == true || value == "true" {
				wants = "Yes"
			}
			dataStr += fmt.Sprintf("News Updates: %s\n", wants)
		} else {
			dataStr += fmt.Sprintf("%s: %v\n", key, value)
		}
	}

	message := fmt.Sprintf(`%s

Form: %s
Submission ID: %s
Submitted: %s
IP Address: %s

Data:
%s`, 
		status,
		formName,
		submission.ID[:8],
		submission.SubmittedAt.Format("2006-01-02 15:04:05"),
		submission.IPAddress,
		dataStr,
	)

	// Add action results if there were any failures
	if !result.Success {
		message += "\nAction Results:\n"
		for _, action := range result.Actions {
			statusIcon := "✅"
			if !action.Success {
				statusIcon = "❌"
			}
			message += fmt.Sprintf("%s %s: %s\n", statusIcon, action.ActionType, action.Message)
			if action.Error != "" {
				message += fmt.Sprintf("   Error: %s\n", action.Error)
			}
		}
	}

	ntfyMsg := NtfyMessage{
		Topic:    ns.config.Notifications.Ntfy.Topic,
		Title:    fmt.Sprintf("Form Submission: %s", formName),
		Message:  message,
		Tags:     tags,
		Priority: priority,
	}

	// Add action button for internal testing submissions (only for Android)
	if submission.FormID == "internal-testing-signup" && result.Success {
		// Check platform - only add action button for Android (iOS gets welcome email automatically)
		platform, platformExists := submission.Data["platform"]
		if platformExists && platform != "ios" {
			// Extract email from submission for the action button
			email := ""
			if emailVal, exists := submission.Data["email"]; exists {
				if emailStr, ok := emailVal.(string); ok {
					email = emailStr
				}
			}
			
			if email != "" {
				ntfyMsg.Actions = []NtfyAction{
					{
						Action: "http",
						Label:  "Send Welcome Email",
						URL:    "https://forms.pinepods.online/api/admin/send-welcome-email",
						Method: "POST",
						Body:   fmt.Sprintf(`{"submission_id": "%s", "email": "%s"}`, submission.ID, email),
					},
				}
			}
		}
	}

	return ns.sendNtfyMessage(ntfyMsg)
}

func (ns *NotificationService) SendCustomNotification(title, message string, tags []string, priority int) error {
	if !ns.config.Notifications.Ntfy.Enabled {
		return nil
	}

	ntfyMsg := NtfyMessage{
		Topic:    ns.config.Notifications.Ntfy.Topic,
		Title:    title,
		Message:  message,
		Tags:     tags,
		Priority: priority,
	}

	return ns.sendNtfyMessage(ntfyMsg)
}

func (ns *NotificationService) sendNtfyMessage(msg NtfyMessage) error {
	jsonData, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal ntfy message: %w", err)
	}

	req, err := http.NewRequest("POST", ns.config.Notifications.Ntfy.URL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create ntfy request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	
	// Add authentication if token is provided
	if ns.config.Notifications.Ntfy.Token != "" {
		req.Header.Set("Authorization", "Bearer "+ns.config.Notifications.Ntfy.Token)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send ntfy notification: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("ntfy server returned status %d", resp.StatusCode)
	}

	return nil
}

func (ns *NotificationService) SendTestNotification() error {
	return ns.SendCustomNotification(
		"Test Notification",
		"PinePods Forms service is running and notifications are working!",
		[]string{"gear", "test"},
		3,
	)
}