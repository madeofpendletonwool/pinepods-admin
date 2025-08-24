package config

import (
	"fmt"
	"os"
	"strconv"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server       ServerConfig       `yaml:"server"`
	Database     DatabaseConfig     `yaml:"database"`
	Email        EmailConfig        `yaml:"email"`
	Notifications NotificationConfig `yaml:"notifications"`
	Forms        FormsConfig        `yaml:"forms"`
	GooglePlay   GooglePlayConfig   `yaml:"google_play"`
}

type ServerConfig struct {
	Port         string `yaml:"port" env:"PORT"`
	Host         string `yaml:"host" env:"HOST"`
	Debug        bool   `yaml:"debug" env:"DEBUG"`
	CORSOrigins  []string `yaml:"cors_origins" env:"CORS_ORIGINS"`
	RateLimiting RateLimitConfig `yaml:"rate_limiting"`
}

type RateLimitConfig struct {
	Enabled     bool `yaml:"enabled"`
	RequestsPerMinute int `yaml:"requests_per_minute"`
}

type DatabaseConfig struct {
	Type     string `yaml:"type" env:"DB_TYPE"`
	Host     string `yaml:"host" env:"DB_HOST"`
	Port     string `yaml:"port" env:"DB_PORT"`
	Database string `yaml:"database" env:"DB_NAME"`
	Username string `yaml:"username" env:"DB_USER"`
	Password string `yaml:"password" env:"DB_PASSWORD"`
	SSLMode  string `yaml:"ssl_mode" env:"DB_SSL_MODE"`
}

type EmailConfig struct {
	Provider string     `yaml:"provider" env:"EMAIL_PROVIDER"`
	SMTP     SMTPConfig `yaml:"smtp"`
	SendGrid SendGridConfig `yaml:"sendgrid"`
}

type SMTPConfig struct {
	Host     string `yaml:"host" env:"SMTP_HOST"`
	Port     int    `yaml:"port" env:"SMTP_PORT"`
	Username string `yaml:"username" env:"SMTP_USERNAME"`
	Password string `yaml:"password" env:"SMTP_PASSWORD"`
	From     string `yaml:"from" env:"SMTP_FROM"`
}

type SendGridConfig struct {
	APIKey string `yaml:"api_key" env:"SENDGRID_API_KEY"`
	From   string `yaml:"from" env:"SENDGRID_FROM"`
}

type NotificationConfig struct {
	Ntfy NtfyConfig `yaml:"ntfy"`
}

type NtfyConfig struct {
	Enabled bool   `yaml:"enabled" env:"NTFY_ENABLED"`
	URL     string `yaml:"url" env:"NTFY_URL"`
	Topic   string `yaml:"topic" env:"NTFY_TOPIC"`
	Token   string `yaml:"token" env:"NTFY_TOKEN"`
}

type FormsConfig struct {
	StorageDir string                 `yaml:"storage_dir" env:"FORMS_STORAGE_DIR"`
	Forms      map[string]FormConfig  `yaml:"forms"`
}

type FormConfig struct {
	Name        string              `yaml:"name"`
	Description string              `yaml:"description"`
	Fields      []FieldConfig       `yaml:"fields"`
	Actions     []ActionConfig      `yaml:"actions"`
	Validation  ValidationConfig    `yaml:"validation"`
	Email       FormEmailConfig     `yaml:"email"`
}

type FieldConfig struct {
	Name        string `yaml:"name"`
	Type        string `yaml:"type"`
	Required    bool   `yaml:"required"`
	Validation  string `yaml:"validation"`
	Label       string `yaml:"label"`
	Placeholder string `yaml:"placeholder"`
}

type ActionConfig struct {
	Type   string                 `yaml:"type"`
	Config map[string]interface{} `yaml:"config"`
}

type ValidationConfig struct {
	MaxSubmissionsPerHour int `yaml:"max_submissions_per_hour"`
	RequireCaptcha        bool `yaml:"require_captcha"`
}

type FormEmailConfig struct {
	Enabled         bool   `yaml:"enabled"`
	Template        string `yaml:"template"`
	Subject         string `yaml:"subject"`
	SendConfirmation bool  `yaml:"send_confirmation"`
}

type GooglePlayConfig struct {
	ServiceAccountFile string `yaml:"service_account_file" env:"GOOGLE_SERVICE_ACCOUNT_FILE"`
	PackageName        string `yaml:"package_name" env:"GOOGLE_PACKAGE_NAME"`
}

// Load reads configuration from file and environment variables
func Load(configPath string) (*Config, error) {
	config := &Config{}
	
	// Set defaults
	config.setDefaults()
	
	// Read from file if it exists
	if _, err := os.Stat(configPath); err == nil {
		data, err := os.ReadFile(configPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
		
		if err := yaml.Unmarshal(data, config); err != nil {
			return nil, fmt.Errorf("failed to parse config file: %w", err)
		}
	}
	
	// Override with environment variables
	config.loadFromEnv()
	
	return config, nil
}

func (c *Config) setDefaults() {
	c.Server.Port = "8080"
	c.Server.Host = "0.0.0.0"
	c.Server.Debug = false
	c.Server.CORSOrigins = []string{"*"}
	c.Server.RateLimiting.Enabled = true
	c.Server.RateLimiting.RequestsPerMinute = 60
	
	c.Database.Type = "sqlite"
	c.Database.Database = "forms.db"
	c.Database.SSLMode = "disable"
	
	c.Email.Provider = "smtp"
	c.Email.SMTP.Port = 587
	
	c.Forms.StorageDir = "./submissions"
}

func (c *Config) loadFromEnv() {
	if port := os.Getenv("PORT"); port != "" {
		c.Server.Port = port
	}
	if host := os.Getenv("HOST"); host != "" {
		c.Server.Host = host
	}
	if debug := os.Getenv("DEBUG"); debug == "true" {
		c.Server.Debug = true
	}
	
	// Database env vars
	if dbType := os.Getenv("DB_TYPE"); dbType != "" {
		c.Database.Type = dbType
	}
	if dbHost := os.Getenv("DB_HOST"); dbHost != "" {
		c.Database.Host = dbHost
	}
	if dbPort := os.Getenv("DB_PORT"); dbPort != "" {
		c.Database.Port = dbPort
	}
	if dbName := os.Getenv("DB_NAME"); dbName != "" {
		c.Database.Database = dbName
	}
	if dbUser := os.Getenv("DB_USER"); dbUser != "" {
		c.Database.Username = dbUser
	}
	if dbPass := os.Getenv("DB_PASSWORD"); dbPass != "" {
		c.Database.Password = dbPass
	}
	
	// Email env vars
	if emailProvider := os.Getenv("EMAIL_PROVIDER"); emailProvider != "" {
		c.Email.Provider = emailProvider
	}
	if smtpHost := os.Getenv("SMTP_HOST"); smtpHost != "" {
		c.Email.SMTP.Host = smtpHost
	}
	if smtpPort := os.Getenv("SMTP_PORT"); smtpPort != "" {
		if port, err := strconv.Atoi(smtpPort); err == nil {
			c.Email.SMTP.Port = port
		}
	}
	if smtpUser := os.Getenv("SMTP_USERNAME"); smtpUser != "" {
		c.Email.SMTP.Username = smtpUser
	}
	if smtpPass := os.Getenv("SMTP_PASSWORD"); smtpPass != "" {
		c.Email.SMTP.Password = smtpPass
	}
	if smtpFrom := os.Getenv("SMTP_FROM"); smtpFrom != "" {
		c.Email.SMTP.From = smtpFrom
	}
	
	// Ntfy env vars
	if ntfyEnabled := os.Getenv("NTFY_ENABLED"); ntfyEnabled == "true" {
		c.Notifications.Ntfy.Enabled = true
	}
	if ntfyURL := os.Getenv("NTFY_URL"); ntfyURL != "" {
		c.Notifications.Ntfy.URL = ntfyURL
	}
	if ntfyTopic := os.Getenv("NTFY_TOPIC"); ntfyTopic != "" {
		c.Notifications.Ntfy.Topic = ntfyTopic
	}
	if ntfyToken := os.Getenv("NTFY_TOKEN"); ntfyToken != "" {
		c.Notifications.Ntfy.Token = ntfyToken
	}
	
	// Google Play env vars
	if serviceAccount := os.Getenv("GOOGLE_SERVICE_ACCOUNT_FILE"); serviceAccount != "" {
		c.GooglePlay.ServiceAccountFile = serviceAccount
	}
	if packageName := os.Getenv("GOOGLE_PACKAGE_NAME"); packageName != "" {
		c.GooglePlay.PackageName = packageName
	}
}