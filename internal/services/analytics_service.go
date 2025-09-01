package services

import (
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/madeofpendletonwool/pinepods-admin/internal/config"
	"github.com/madeofpendletonwool/pinepods-admin/internal/models"
)

type AnalyticsService struct {
	config *config.Config
	db     *sql.DB
}

func NewAnalyticsService(cfg *config.Config, db *sql.DB) *AnalyticsService {
	service := &AnalyticsService{
		config: cfg,
		db:     db,
	}
	
	if err := service.createAnalyticsTables(); err != nil {
		fmt.Printf("Warning: Failed to create analytics tables: %v\n", err)
	}
	
	return service
}

func (as *AnalyticsService) createAnalyticsTables() error {
	var createTableSQL string
	
	switch as.config.Database.Type {
	case "sqlite":
		createTableSQL = `
		CREATE TABLE IF NOT EXISTS pinepods_analytics (
			id TEXT PRIMARY KEY,
			server_hash TEXT NOT NULL UNIQUE,
			version TEXT NOT NULL,
			first_seen DATETIME NOT NULL,
			last_seen DATETIME NOT NULL,
			ip_hash TEXT NOT NULL
		);
		CREATE INDEX IF NOT EXISTS idx_analytics_server_hash ON pinepods_analytics(server_hash);
		CREATE INDEX IF NOT EXISTS idx_analytics_last_seen ON pinepods_analytics(last_seen);
		CREATE INDEX IF NOT EXISTS idx_analytics_version ON pinepods_analytics(version);
		`
	case "postgres":
		createTableSQL = `
		CREATE TABLE IF NOT EXISTS pinepods_analytics (
			id TEXT PRIMARY KEY,
			server_hash TEXT NOT NULL UNIQUE,
			version TEXT NOT NULL,
			first_seen TIMESTAMP WITH TIME ZONE NOT NULL,
			last_seen TIMESTAMP WITH TIME ZONE NOT NULL,
			ip_hash TEXT NOT NULL
		);
		CREATE INDEX IF NOT EXISTS idx_analytics_server_hash ON pinepods_analytics(server_hash);
		CREATE INDEX IF NOT EXISTS idx_analytics_last_seen ON pinepods_analytics(last_seen);
		CREATE INDEX IF NOT EXISTS idx_analytics_version ON pinepods_analytics(version);
		`
	}
	
	_, err := as.db.Exec(createTableSQL)
	return err
}

func (as *AnalyticsService) VerifySignature(req *models.AnalyticsRequest, ipAddress string) bool {
	// Create expected signature: HMAC-SHA256(server_hash + version + ip_address, secret_key)
	data := req.ServerHash + req.Version + ipAddress
	secret := as.config.Analytics.SecretKey // We'll add this to config
	
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(data))
	expectedSig := hex.EncodeToString(h.Sum(nil))
	
	return hmac.Equal([]byte(req.Signature), []byte(expectedSig))
}

func (as *AnalyticsService) ProcessAnalytics(req *models.AnalyticsRequest, ipAddress string) error {
	// Hash IP address for abuse prevention (don't store raw IP)
	h := sha256.New()
	h.Write([]byte(ipAddress))
	ipHash := hex.EncodeToString(h.Sum(nil))
	
	now := time.Now().UTC()
	
	// Check if server already exists
	var existing models.PinepodsAnalytics
	query := `SELECT id, server_hash, version, first_seen, last_seen, ip_hash FROM pinepods_analytics WHERE server_hash = ?`
	if as.config.Database.Type == "postgres" {
		query = `SELECT id, server_hash, version, first_seen, last_seen, ip_hash FROM pinepods_analytics WHERE server_hash = $1`
	}
	
	err := as.db.QueryRow(query, req.ServerHash).Scan(
		&existing.ID,
		&existing.ServerHash,
		&existing.Version,
		&existing.FirstSeen,
		&existing.LastSeen,
		&existing.IPHash,
	)
	
	if err == sql.ErrNoRows {
		// New server, insert it
		newID := uuid.New().String()
		insertQuery := `INSERT INTO pinepods_analytics (id, server_hash, version, first_seen, last_seen, ip_hash) VALUES (?, ?, ?, ?, ?, ?)`
		if as.config.Database.Type == "postgres" {
			insertQuery = `INSERT INTO pinepods_analytics (id, server_hash, version, first_seen, last_seen, ip_hash) VALUES ($1, $2, $3, $4, $5, $6)`
		}
		
		_, err = as.db.Exec(insertQuery, newID, req.ServerHash, req.Version, now, now, ipHash)
		if err != nil {
			return fmt.Errorf("failed to insert new analytics record: %w", err)
		}
		
		fmt.Printf("[ANALYTICS] New Pinepods server registered: hash=%s, version=%s\n", req.ServerHash[:8], req.Version)
	} else if err != nil {
		return fmt.Errorf("failed to query existing analytics record: %w", err)
	} else {
		// Existing server, update it
		updateQuery := `UPDATE pinepods_analytics SET version = ?, last_seen = ? WHERE server_hash = ?`
		if as.config.Database.Type == "postgres" {
			updateQuery = `UPDATE pinepods_analytics SET version = $1, last_seen = $2 WHERE server_hash = $3`
		}
		
		_, err = as.db.Exec(updateQuery, req.Version, now, req.ServerHash)
		if err != nil {
			return fmt.Errorf("failed to update analytics record: %w", err)
		}
		
		fmt.Printf("[ANALYTICS] Pinepods server check-in: hash=%s, version=%s (was %s)\n", req.ServerHash[:8], req.Version, existing.Version)
	}
	
	return nil
}

func (as *AnalyticsService) GetAnalyticsSummary() (*models.AnalyticsSummary, error) {
	// Define active threshold (30 days)
	activeThreshold := time.Now().UTC().Add(-30 * 24 * time.Hour)
	
	summary := &models.AnalyticsSummary{
		VersionBreakdown: make(map[string]int),
		LastUpdated:      time.Now().UTC(),
	}
	
	// Get total server count
	totalQuery := `SELECT COUNT(*) FROM pinepods_analytics`
	err := as.db.QueryRow(totalQuery).Scan(&summary.TotalServers)
	if err != nil {
		return nil, fmt.Errorf("failed to get total server count: %w", err)
	}
	
	// Get active server count (last seen within 30 days)
	activeQuery := `SELECT COUNT(*) FROM pinepods_analytics WHERE last_seen > ?`
	if as.config.Database.Type == "postgres" {
		activeQuery = `SELECT COUNT(*) FROM pinepods_analytics WHERE last_seen > $1`
	}
	err = as.db.QueryRow(activeQuery, activeThreshold).Scan(&summary.ActiveServers)
	if err != nil {
		return nil, fmt.Errorf("failed to get active server count: %w", err)
	}
	
	// Get version breakdown (only active servers)
	versionQuery := `SELECT version, COUNT(*) FROM pinepods_analytics WHERE last_seen > ? GROUP BY version`
	if as.config.Database.Type == "postgres" {
		versionQuery = `SELECT version, COUNT(*) FROM pinepods_analytics WHERE last_seen > $1 GROUP BY version`
	}
	
	rows, err := as.db.Query(versionQuery, activeThreshold)
	if err != nil {
		return nil, fmt.Errorf("failed to get version breakdown: %w", err)
	}
	defer rows.Close()
	
	for rows.Next() {
		var version string
		var count int
		if err := rows.Scan(&version, &count); err != nil {
			return nil, fmt.Errorf("failed to scan version row: %w", err)
		}
		summary.VersionBreakdown[version] = count
	}
	
	return summary, nil
}

func (as *AnalyticsService) CleanupInactiveServers(daysThreshold int) (int, error) {
	cutoff := time.Now().UTC().Add(-time.Duration(daysThreshold) * 24 * time.Hour)
	
	deleteQuery := `DELETE FROM pinepods_analytics WHERE last_seen < ?`
	if as.config.Database.Type == "postgres" {
		deleteQuery = `DELETE FROM pinepods_analytics WHERE last_seen < $1`
	}
	
	result, err := as.db.Exec(deleteQuery, cutoff)
	if err != nil {
		return 0, fmt.Errorf("failed to cleanup inactive servers: %w", err)
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}
	
	fmt.Printf("[ANALYTICS] Cleaned up %d inactive servers (not seen for %d days)\n", rowsAffected, daysThreshold)
	return int(rowsAffected), nil
}