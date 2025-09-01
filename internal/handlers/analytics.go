package handlers

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/madeofpendletonwool/pinepods-admin/internal/models"
)

func (s *Server) submitAnalytics(c *gin.Context) {
	if !s.config.Analytics.Enabled {
		c.JSON(http.StatusServiceUnavailable, models.ErrorResponse{
			Success: false,
			Error:   "Analytics collection is disabled",
			Code:    http.StatusServiceUnavailable,
		})
		return
	}

	var req models.AnalyticsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Success: false,
			Error:   "Invalid request format: " + err.Error(),
			Code:    http.StatusBadRequest,
		})
		return
	}

	// Verify signature to prevent abuse
	if !s.analyticsService.VerifySignature(&req, c.ClientIP()) {
		fmt.Printf("[ANALYTICS] Invalid signature from IP %s for server %s\n", c.ClientIP(), req.ServerHash[:8])
		c.JSON(http.StatusUnauthorized, models.ErrorResponse{
			Success: false,
			Error:   "Invalid signature",
			Code:    http.StatusUnauthorized,
		})
		return
	}

	// Process the analytics
	err := s.analyticsService.ProcessAnalytics(&req, c.ClientIP())
	if err != nil {
		fmt.Printf("[ANALYTICS] Failed to process analytics: %v\n", err)
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Success: false,
			Error:   "Failed to process analytics: " + err.Error(),
			Code:    http.StatusInternalServerError,
		})
		return
	}

	c.JSON(http.StatusOK, models.AnalyticsResponse{
		Success:   true,
		Message:   "Analytics processed successfully",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	})
}

func (s *Server) getAnalyticsSummary(c *gin.Context) {
	if !s.config.Analytics.Enabled {
		c.JSON(http.StatusServiceUnavailable, models.ErrorResponse{
			Success: false,
			Error:   "Analytics collection is disabled",
			Code:    http.StatusServiceUnavailable,
		})
		return
	}

	summary, err := s.analyticsService.GetAnalyticsSummary()
	if err != nil {
		fmt.Printf("[ANALYTICS] Failed to get analytics summary: %v\n", err)
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Success: false,
			Error:   "Failed to retrieve analytics summary: " + err.Error(),
			Code:    http.StatusInternalServerError,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    summary,
	})
}

func (s *Server) cleanupAnalytics(c *gin.Context) {
	// This could be protected with admin auth in the future
	daysThreshold := 90 // Default to 90 days for cleanup, but count active as 30 days

	if daysStr := c.Query("days"); daysStr != "" {
		if days, err := time.ParseDuration(daysStr + "h"); err == nil {
			daysThreshold = int(days.Hours() / 24)
		}
	}

	removed, err := s.analyticsService.CleanupInactiveServers(daysThreshold)
	if err != nil {
		fmt.Printf("[ANALYTICS] Failed to cleanup analytics: %v\n", err)
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Success: false,
			Error:   "Failed to cleanup analytics: " + err.Error(),
			Code:    http.StatusInternalServerError,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": fmt.Sprintf("Cleaned up %d inactive servers", removed),
		"removed": removed,
	})
}