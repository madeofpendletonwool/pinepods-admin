package models

import (
	"time"
)

// FormSubmission represents a form submission stored in the database
type FormSubmission struct {
	ID          string                 `json:"id" db:"id"`
	FormID      string                 `json:"form_id" db:"form_id"`
	Data        map[string]interface{} `json:"data" db:"data"`
	IPAddress   string                 `json:"ip_address" db:"ip_address"`
	UserAgent   string                 `json:"user_agent" db:"user_agent"`
	SubmittedAt time.Time              `json:"submitted_at" db:"submitted_at"`
	Processed   bool                   `json:"processed" db:"processed"`
	ProcessedAt *time.Time             `json:"processed_at,omitempty" db:"processed_at"`
	Error       string                 `json:"error,omitempty" db:"error"`
}

// SubmissionRequest represents the incoming form submission request
type SubmissionRequest struct {
	FormID string                 `json:"form_id" binding:"required"`
	Data   map[string]interface{} `json:"data" binding:"required"`
}

// SubmissionResponse represents the response sent back after form submission
type SubmissionResponse struct {
	Success   bool   `json:"success"`
	Message   string `json:"message"`
	ID        string `json:"id,omitempty"`
	Timestamp string `json:"timestamp"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error"`
	Code    int    `json:"code"`
}

// FormInfo represents basic information about a form
type FormInfo struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Enabled     bool   `json:"enabled"`
}

// ActionResult represents the result of executing an action
type ActionResult struct {
	ActionType string `json:"action_type"`
	Success    bool   `json:"success"`
	Message    string `json:"message"`
	Error      string `json:"error,omitempty"`
}

// ProcessingResult represents the overall result of processing a submission
type ProcessingResult struct {
	SubmissionID string         `json:"submission_id"`
	FormID       string         `json:"form_id"`
	Success      bool           `json:"success"`
	Actions      []ActionResult `json:"actions"`
	ProcessedAt  time.Time      `json:"processed_at"`
}

// PinepodsAnalytics represents analytics data from a Pinepods server
type PinepodsAnalytics struct {
	ID         string    `json:"id" db:"id"`
	ServerHash string    `json:"server_hash" db:"server_hash"`
	Version    string    `json:"version" db:"version"`
	FirstSeen  time.Time `json:"first_seen" db:"first_seen"`
	LastSeen   time.Time `json:"last_seen" db:"last_seen"`
	IPHash     string    `json:"-" db:"ip_hash"` // Hashed IP for abuse prevention, not returned in JSON
}

// AnalyticsRequest represents incoming analytics from a Pinepods server
type AnalyticsRequest struct {
	ServerHash string `json:"server_hash" binding:"required"`
	Version    string `json:"version" binding:"required"`
	Signature  string `json:"signature" binding:"required"` // HMAC signature for verification
}

// AnalyticsResponse represents the response sent back after analytics submission
type AnalyticsResponse struct {
	Success   bool   `json:"success"`
	Message   string `json:"message"`
	Timestamp string `json:"timestamp"`
}

// AnalyticsSummary represents aggregated analytics data for public display
type AnalyticsSummary struct {
	TotalServers     int            `json:"total_servers"`
	ActiveServers    int            `json:"active_servers"`
	VersionBreakdown map[string]int `json:"version_breakdown"`
	LastUpdated      time.Time      `json:"last_updated"`
}