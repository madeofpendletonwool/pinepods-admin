package services

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
	_ "modernc.org/sqlite"
	"github.com/madeofpendletonwool/pinepods-admin/internal/config"
	"github.com/madeofpendletonwool/pinepods-admin/internal/models"
)

type FormService struct {
	config *config.Config
	db     *sql.DB
}

func NewFormService(cfg *config.Config) *FormService {
	service := &FormService{
		config: cfg,
	}
	
	if err := service.initDatabase(); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	
	return service
}

func (fs *FormService) initDatabase() error {
	var err error
	var connectionString string
	
	switch fs.config.Database.Type {
	case "sqlite":
		connectionString = fs.config.Database.Database
		fs.db, err = sql.Open("sqlite", connectionString)
	case "postgres":
		connectionString = fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
			fs.config.Database.Host,
			fs.config.Database.Port,
			fs.config.Database.Username,
			fs.config.Database.Password,
			fs.config.Database.Database,
			fs.config.Database.SSLMode,
		)
		fs.db, err = sql.Open("postgres", connectionString)
	default:
		return fmt.Errorf("unsupported database type: %s", fs.config.Database.Type)
	}
	
	if err != nil {
		return fmt.Errorf("failed to open database connection: %w", err)
	}
	
	if err := fs.db.Ping(); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}
	
	return fs.createTables()
}

func (fs *FormService) createTables() error {
	var createTableSQL string
	
	switch fs.config.Database.Type {
	case "sqlite":
		createTableSQL = `
		CREATE TABLE IF NOT EXISTS form_submissions (
			id TEXT PRIMARY KEY,
			form_id TEXT NOT NULL,
			data TEXT NOT NULL,
			ip_address TEXT,
			user_agent TEXT,
			submitted_at DATETIME NOT NULL,
			processed BOOLEAN DEFAULT FALSE,
			processed_at DATETIME,
			error TEXT
		);
		CREATE INDEX IF NOT EXISTS idx_form_submissions_form_id ON form_submissions(form_id);
		CREATE INDEX IF NOT EXISTS idx_form_submissions_submitted_at ON form_submissions(submitted_at);
		`
	case "postgres":
		createTableSQL = `
		CREATE TABLE IF NOT EXISTS form_submissions (
			id TEXT PRIMARY KEY,
			form_id TEXT NOT NULL,
			data JSONB NOT NULL,
			ip_address TEXT,
			user_agent TEXT,
			submitted_at TIMESTAMP WITH TIME ZONE NOT NULL,
			processed BOOLEAN DEFAULT FALSE,
			processed_at TIMESTAMP WITH TIME ZONE,
			error TEXT
		);
		CREATE INDEX IF NOT EXISTS idx_form_submissions_form_id ON form_submissions(form_id);
		CREATE INDEX IF NOT EXISTS idx_form_submissions_submitted_at ON form_submissions(submitted_at);
		`
	}
	
	_, err := fs.db.Exec(createTableSQL)
	return err
}

func (fs *FormService) ProcessSubmission(submission *models.FormSubmission) (*models.ProcessingResult, error) {
	// Generate ID if not set
	if submission.ID == "" {
		submission.ID = uuid.New().String()
	}
	
	// Validate form exists
	formConfig, exists := fs.GetFormConfig(submission.FormID)
	if !exists {
		return nil, fmt.Errorf("form '%s' not found", submission.FormID)
	}
	
	// Validate submission data
	if err := fs.validateSubmission(submission, formConfig); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}
	
	// Store submission
	if err := fs.storeSubmission(submission); err != nil {
		return nil, fmt.Errorf("failed to store submission: %w", err)
	}
	
	// Process actions
	actionService := NewActionService(fs.config)
	result := actionService.ProcessActions(submission, formConfig)
	
	// Update submission status
	submission.Processed = result.Success
	submission.ProcessedAt = &result.ProcessedAt
	if !result.Success {
		var errorMsg string
		for _, action := range result.Actions {
			if !action.Success && action.Error != "" {
				errorMsg += action.Error + "; "
			}
		}
		submission.Error = errorMsg
	}
	
	if err := fs.updateSubmission(submission); err != nil {
		log.Printf("Failed to update submission status: %v", err)
	}
	
	// Store submission to file system for backup
	if err := fs.storeSubmissionFile(submission); err != nil {
		log.Printf("Failed to store submission file: %v", err)
	}
	
	return result, nil
}

func (fs *FormService) validateSubmission(submission *models.FormSubmission, formConfig config.FormConfig) error {
	// Check required fields
	for _, field := range formConfig.Fields {
		if field.Required {
			if value, exists := submission.Data[field.Name]; !exists || value == "" {
				return fmt.Errorf("required field '%s' is missing", field.Name)
			}
		}
	}
	
	// Add additional validation logic here (regex, length, etc.)
	
	return nil
}

func (fs *FormService) storeSubmission(submission *models.FormSubmission) error {
	dataJSON, err := json.Marshal(submission.Data)
	if err != nil {
		return err
	}
	
	query := `
		INSERT INTO form_submissions (id, form_id, data, ip_address, user_agent, submitted_at, processed, processed_at, error)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	
	if fs.config.Database.Type == "postgres" {
		query = `
			INSERT INTO form_submissions (id, form_id, data, ip_address, user_agent, submitted_at, processed, processed_at, error)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		`
	}
	
	_, err = fs.db.Exec(query,
		submission.ID,
		submission.FormID,
		string(dataJSON),
		submission.IPAddress,
		submission.UserAgent,
		submission.SubmittedAt,
		submission.Processed,
		submission.ProcessedAt,
		submission.Error,
	)
	
	return err
}

func (fs *FormService) updateSubmission(submission *models.FormSubmission) error {
	query := `
		UPDATE form_submissions 
		SET processed = ?, processed_at = ?, error = ?
		WHERE id = ?
	`
	
	if fs.config.Database.Type == "postgres" {
		query = `
			UPDATE form_submissions 
			SET processed = $1, processed_at = $2, error = $3
			WHERE id = $4
		`
	}
	
	_, err := fs.db.Exec(query,
		submission.Processed,
		submission.ProcessedAt,
		submission.Error,
		submission.ID,
	)
	
	return err
}

func (fs *FormService) storeSubmissionFile(submission *models.FormSubmission) error {
	// Create storage directory if it doesn't exist
	storageDir := fs.config.Forms.StorageDir
	if err := os.MkdirAll(storageDir, 0755); err != nil {
		return err
	}
	
	// Create date-based subdirectory
	dateDir := filepath.Join(storageDir, submission.SubmittedAt.Format("2006-01-02"))
	if err := os.MkdirAll(dateDir, 0755); err != nil {
		return err
	}
	
	// Write submission to file
	filename := fmt.Sprintf("%s_%s.json", submission.FormID, submission.ID)
	filepath := filepath.Join(dateDir, filename)
	
	data, err := json.MarshalIndent(submission, "", "  ")
	if err != nil {
		return err
	}
	
	return os.WriteFile(filepath, data, 0644)
}

func (fs *FormService) GetFormConfig(formID string) (config.FormConfig, bool) {
	if form, exists := fs.config.Forms.Forms[formID]; exists {
		return form, true
	}
	return config.FormConfig{}, false
}

func (fs *FormService) GetAvailableForms() []models.FormInfo {
	var forms []models.FormInfo
	for id, form := range fs.config.Forms.Forms {
		forms = append(forms, models.FormInfo{
			ID:          id,
			Name:        form.Name,
			Description: form.Description,
			Enabled:     true,
		})
	}
	return forms
}

func (fs *FormService) GetFormSubmissions(formID string, limit, offset int) ([]models.FormSubmission, error) {
	query := `
		SELECT id, form_id, data, ip_address, user_agent, submitted_at, processed, processed_at, error
		FROM form_submissions 
		WHERE form_id = ?
		ORDER BY submitted_at DESC
		LIMIT ? OFFSET ?
	`
	
	if fs.config.Database.Type == "postgres" {
		query = `
			SELECT id, form_id, data, ip_address, user_agent, submitted_at, processed, processed_at, error
			FROM form_submissions 
			WHERE form_id = $1
			ORDER BY submitted_at DESC
			LIMIT $2 OFFSET $3
		`
	}
	
	return fs.querySubmissions(query, formID, limit, offset)
}

func (fs *FormService) GetAllSubmissions(limit, offset int) ([]models.FormSubmission, error) {
	query := `
		SELECT id, form_id, data, ip_address, user_agent, submitted_at, processed, processed_at, error
		FROM form_submissions 
		ORDER BY submitted_at DESC
		LIMIT ? OFFSET ?
	`
	
	if fs.config.Database.Type == "postgres" {
		query = `
			SELECT id, form_id, data, ip_address, user_agent, submitted_at, processed, processed_at, error
			FROM form_submissions 
			ORDER BY submitted_at DESC
			LIMIT $1 OFFSET $2
		`
	}
	
	return fs.querySubmissions(query, limit, offset)
}

func (fs *FormService) GetSubmission(submissionID string) (*models.FormSubmission, error) {
	query := `
		SELECT id, form_id, data, ip_address, user_agent, submitted_at, processed, processed_at, error
		FROM form_submissions 
		WHERE id = ?
	`
	
	if fs.config.Database.Type == "postgres" {
		query = `
			SELECT id, form_id, data, ip_address, user_agent, submitted_at, processed, processed_at, error
			FROM form_submissions 
			WHERE id = $1
		`
	}
	
	submissions, err := fs.querySubmissions(query, submissionID)
	if err != nil {
		return nil, err
	}
	
	if len(submissions) == 0 {
		return nil, fmt.Errorf("submission not found")
	}
	
	return &submissions[0], nil
}

func (fs *FormService) DeleteSubmission(submissionID string) error {
	query := `DELETE FROM form_submissions WHERE id = ?`
	
	if fs.config.Database.Type == "postgres" {
		query = `DELETE FROM form_submissions WHERE id = $1`
	}
	
	_, err := fs.db.Exec(query, submissionID)
	return err
}

func (fs *FormService) ReprocessSubmission(submission *models.FormSubmission) (*models.ProcessingResult, error) {
	formConfig, exists := fs.GetFormConfig(submission.FormID)
	if !exists {
		return nil, fmt.Errorf("form '%s' not found", submission.FormID)
	}
	
	// Reset submission status
	submission.Processed = false
	submission.ProcessedAt = nil
	submission.Error = ""
	
	// Process actions
	actionService := NewActionService(fs.config)
	result := actionService.ProcessActions(submission, formConfig)
	
	// Update submission status
	submission.Processed = result.Success
	submission.ProcessedAt = &result.ProcessedAt
	if !result.Success {
		var errorMsg string
		for _, action := range result.Actions {
			if !action.Success && action.Error != "" {
				errorMsg += action.Error + "; "
			}
		}
		submission.Error = errorMsg
	}
	
	if err := fs.updateSubmission(submission); err != nil {
		return nil, fmt.Errorf("failed to update submission: %w", err)
	}
	
	return result, nil
}

func (fs *FormService) querySubmissions(query string, args ...interface{}) ([]models.FormSubmission, error) {
	rows, err := fs.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var submissions []models.FormSubmission
	for rows.Next() {
		var submission models.FormSubmission
		var dataJSON string
		var processedAt sql.NullTime
		var errorStr sql.NullString
		
		err := rows.Scan(
			&submission.ID,
			&submission.FormID,
			&dataJSON,
			&submission.IPAddress,
			&submission.UserAgent,
			&submission.SubmittedAt,
			&submission.Processed,
			&processedAt,
			&errorStr,
		)
		if err != nil {
			return nil, err
		}
		
		if err := json.Unmarshal([]byte(dataJSON), &submission.Data); err != nil {
			return nil, err
		}
		
		if processedAt.Valid {
			submission.ProcessedAt = &processedAt.Time
		}
		
		if errorStr.Valid {
			submission.Error = errorStr.String
		}
		
		submissions = append(submissions, submission)
	}
	
	return submissions, rows.Err()
}