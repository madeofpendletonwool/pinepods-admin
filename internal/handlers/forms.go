package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/madeofpendletonwool/pinepods-admin/internal/models"
	"github.com/madeofpendletonwool/pinepods-admin/internal/services"
)

func (s *Server) healthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "healthy",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"version":   "1.0.0",
	})
}

func (s *Server) submitForm(c *gin.Context) {
	var req models.SubmissionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Success: false,
			Error:   "Invalid request format: " + err.Error(),
			Code:    http.StatusBadRequest,
		})
		return
	}

	// Get client info
	submission := &models.FormSubmission{
		FormID:      req.FormID,
		Data:        req.Data,
		IPAddress:   c.ClientIP(),
		UserAgent:   c.GetHeader("User-Agent"),
		SubmittedAt: time.Now().UTC(),
	}

	// Process the submission
	result, err := s.formService.ProcessSubmission(submission)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Success: false,
			Error:   "Failed to process submission: " + err.Error(),
			Code:    http.StatusInternalServerError,
		})
		return
	}

	// Send notification if enabled
	if s.config.Notifications.Ntfy.Enabled {
		go s.notificationService.SendSubmissionNotification(submission, result)
	}

	c.JSON(http.StatusOK, models.SubmissionResponse{
		Success:   true,
		Message:   "Form submitted successfully",
		ID:        submission.ID,
		Timestamp: submission.SubmittedAt.Format(time.RFC3339),
	})
}

func (s *Server) listForms(c *gin.Context) {
	forms := s.formService.GetAvailableForms()
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"forms":   forms,
	})
}

func (s *Server) getForm(c *gin.Context) {
	formID := c.Param("id")
	form, exists := s.formService.GetFormConfig(formID)
	if !exists {
		c.JSON(http.StatusNotFound, models.ErrorResponse{
			Success: false,
			Error:   "Form not found",
			Code:    http.StatusNotFound,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"form":    form,
	})
}

func (s *Server) getFormSubmissions(c *gin.Context) {
	formID := c.Param("id")
	
	// Parse query parameters
	limitStr := c.DefaultQuery("limit", "50")
	offsetStr := c.DefaultQuery("offset", "0")
	
	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		limit = 50
	}
	
	offset, err := strconv.Atoi(offsetStr)
	if err != nil {
		offset = 0
	}

	submissions, err := s.formService.GetFormSubmissions(formID, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Success: false,
			Error:   "Failed to retrieve submissions: " + err.Error(),
			Code:    http.StatusInternalServerError,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":     true,
		"submissions": submissions,
		"count":       len(submissions),
	})
}

func (s *Server) getAllSubmissions(c *gin.Context) {
	limitStr := c.DefaultQuery("limit", "50")
	offsetStr := c.DefaultQuery("offset", "0")
	
	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		limit = 50
	}
	
	offset, err := strconv.Atoi(offsetStr)
	if err != nil {
		offset = 0
	}

	submissions, err := s.formService.GetAllSubmissions(limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Success: false,
			Error:   "Failed to retrieve submissions: " + err.Error(),
			Code:    http.StatusInternalServerError,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":     true,
		"submissions": submissions,
		"count":       len(submissions),
	})
}

func (s *Server) getSubmission(c *gin.Context) {
	submissionID := c.Param("id")
	
	submission, err := s.formService.GetSubmission(submissionID)
	if err != nil {
		c.JSON(http.StatusNotFound, models.ErrorResponse{
			Success: false,
			Error:   "Submission not found",
			Code:    http.StatusNotFound,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":    true,
		"submission": submission,
	})
}

func (s *Server) deleteSubmission(c *gin.Context) {
	submissionID := c.Param("id")
	
	err := s.formService.DeleteSubmission(submissionID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Success: false,
			Error:   "Failed to delete submission: " + err.Error(),
			Code:    http.StatusInternalServerError,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Submission deleted successfully",
	})
}

func (s *Server) reprocessSubmission(c *gin.Context) {
	submissionID := c.Param("id")
	
	submission, err := s.formService.GetSubmission(submissionID)
	if err != nil {
		c.JSON(http.StatusNotFound, models.ErrorResponse{
			Success: false,
			Error:   "Submission not found",
			Code:    http.StatusNotFound,
		})
		return
	}

	result, err := s.formService.ReprocessSubmission(submission)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Success: false,
			Error:   "Failed to reprocess submission: " + err.Error(),
			Code:    http.StatusInternalServerError,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Submission reprocessed successfully",
		"result":  result,
	})
}

func (s *Server) sendWelcomeEmail(c *gin.Context) {
	var req struct {
		SubmissionID string `json:"submission_id" binding:"required"`
		Email        string `json:"email" binding:"required"`
	}
	
	fmt.Printf("[DEBUG] Received welcome email request: %+v\n", c.Request.Body)
	
	if err := c.ShouldBindJSON(&req); err != nil {
		fmt.Printf("[ERROR] Failed to bind JSON: %v\n", err)
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Success: false,
			Error:   "Invalid request format: " + err.Error(),
			Code:    http.StatusBadRequest,
		})
		return
	}

	fmt.Printf("[DEBUG] Parsed request: SubmissionID=%s, Email=%s\n", req.SubmissionID, req.Email)

	// Get the submission to verify it exists and get form config
	submission, err := s.formService.GetSubmission(req.SubmissionID)
	if err != nil {
		fmt.Printf("[ERROR] Failed to get submission %s: %v\n", req.SubmissionID, err)
		c.JSON(http.StatusNotFound, models.ErrorResponse{
			Success: false,
			Error:   "Submission not found",
			Code:    http.StatusNotFound,
		})
		return
	}

	fmt.Printf("[DEBUG] Found submission: %+v\n", submission)

	// Get form config
	formConfig, exists := s.config.Forms.Forms[submission.FormID]
	if !exists {
		fmt.Printf("[ERROR] Form config not found for FormID: %s\n", submission.FormID)
		c.JSON(http.StatusNotFound, models.ErrorResponse{
			Success: false,
			Error:   "Form configuration not found",
			Code:    http.StatusNotFound,
		})
		return
	}

	fmt.Printf("[DEBUG] Found form config: %+v\n", formConfig)
	fmt.Printf("[DEBUG] SMTP Config: Host=%s, Port=%d, From=%s\n", 
		s.config.Email.SMTP.Host, s.config.Email.SMTP.Port, s.config.Email.SMTP.From)

	// Send welcome email
	emailService := services.NewEmailService(s.config)
	err = emailService.SendWelcomeEmail(submission, formConfig, req.Email)
	if err != nil {
		fmt.Printf("[ERROR] Failed to send welcome email: %v\n", err)
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Success: false,
			Error:   "Failed to send welcome email: " + err.Error(),
			Code:    http.StatusInternalServerError,
		})
		return
	}

	fmt.Printf("[SUCCESS] Welcome email sent to %s\n", req.Email)
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": fmt.Sprintf("Welcome email sent to %s", req.Email),
	})
}

func (s *Server) indexPage(c *gin.Context) {
	forms := s.formService.GetAvailableForms()
	c.HTML(http.StatusOK, "index.html", gin.H{
		"title": "PinePods Forms",
		"forms": forms,
	})
}