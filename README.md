# PinePods Admin

A powerful admin and form management system for PinePods, built in Go with support for custom actions, notifications, and integrations.

## Features

🚀 **Configurable Forms**: Define forms with YAML/JSON configuration  
📧 **Email Integration**: SMTP support with HTML templates  
🔔 **Notifications**: ntfy.sh integration for real-time alerts  
🎯 **Custom Actions**: Extensible action system for form processing  
📱 **Google Play Integration**: Automatic tester management for Android apps  
🐳 **Docker Ready**: Easy deployment with Docker and docker-compose  
🗄️ **Multiple Databases**: SQLite and PostgreSQL support  
🔒 **Security**: Rate limiting, CORS, and input validation  
📊 **Monitoring**: Health checks and structured logging  

## Quick Start

### Using Docker (Recommended)

1. Clone the repository:
```bash
git clone https://github.com/madeofpendletonwool/pinepods-admin.git
cd pinepods-admin
```

2. Create your configuration:
```bash
cp configs/config.yaml configs/config.local.yaml
# Edit configs/config.local.yaml with your settings
```

3. Create data directories:
```bash
mkdir -p data submissions
```

4. Start the service:
```bash
docker-compose up -d
```

The service will be available at `http://localhost:8080`

### Manual Installation

1. Install Go 1.21 or later
2. Clone and build:
```bash
git clone https://github.com/madeofpendletonwool/pinepods-admin.git
cd pinepods-admin
go mod download
go build -o pinepods-admin cmd/server/main.go
```

3. Run:
```bash
./pinepods-admin -config configs/config.yaml
```

## Configuration

### Basic Configuration

Create a `config.yaml` file:

```yaml
server:
  port: "8080"
  host: "0.0.0.0"
  cors_origins:
    - "https://your-docs-site.com"

database:
  type: "sqlite"
  database: "./data/forms.db"

email:
  provider: "smtp"
  smtp:
    host: "smtp.gmail.com"
    port: 587
    username: "your-email@gmail.com"
    password: "your-app-password"
    from: "your-email@gmail.com"

notifications:
  ntfy:
    enabled: true
    url: "https://ntfy.sh"
    topic: "your-forms-topic"

forms:
  forms:
    contact:
      name: "Contact Form"
      fields:
        - name: "name"
          type: "text"
          required: true
        - name: "email"
          type: "email"
          required: true
        - name: "message"
          type: "textarea"
          required: true
      actions:
        - type: "send_email"
        - type: "log"
```

### Google Play Console Integration

For automatic tester management:

1. Create a service account in Google Cloud Console
2. Enable the Google Play Developer API
3. Add the service account to your Google Play Console with "Release Manager" permissions
4. Download the service account JSON file

```yaml
google_play:
  service_account_file: "/path/to/service-account.json"
  package_name: "com.your.app.package"

forms:
  forms:
    internal-testing:
      name: "Internal Testing Signup"
      fields:
        - name: "name"
          type: "text"
          required: true
        - name: "email"
          type: "email"
          required: true
      actions:
        - type: "google_play_add_tester"
          config:
            track: "internal"
        - type: "send_email"
```

## API Usage

### Submit a Form

```bash
curl -X POST http://localhost:8080/api/forms/submit \
  -H "Content-Type: application/json" \
  -d '{
    "form_id": "contact",
    "data": {
      "name": "John Doe",
      "email": "john@example.com",
      "message": "Hello world!"
    }
  }'
```

### List Available Forms

```bash
curl http://localhost:8080/api/forms/
```

### Get Form Configuration

```bash
curl http://localhost:8080/api/forms/contact
```

## Environment Variables

You can override configuration values with environment variables:

| Variable | Description | Example |
|----------|-------------|---------|
| `PORT` | Server port | `8080` |
| `HOST` | Server host | `0.0.0.0` |
| `DEBUG` | Debug mode | `false` |
| `DB_TYPE` | Database type | `sqlite` or `postgres` |
| `DB_HOST` | Database host | `localhost` |
| `DB_PORT` | Database port | `5432` |
| `DB_NAME` | Database name | `forms` |
| `DB_USER` | Database user | `forms_user` |
| `DB_PASSWORD` | Database password | `password` |
| `SMTP_HOST` | SMTP server | `smtp.gmail.com` |
| `SMTP_USERNAME` | SMTP username | `user@gmail.com` |
| `SMTP_PASSWORD` | SMTP password | `app_password` |
| `SMTP_FROM` | From email address | `forms@company.com` |
| `NTFY_ENABLED` | Enable ntfy notifications | `true` |
| `NTFY_URL` | ntfy server URL | `https://ntfy.sh` |
| `NTFY_TOPIC` | ntfy topic | `forms-notifications` |
| `NTFY_TOKEN` | ntfy auth token | `tk_...` |
| `GOOGLE_SERVICE_ACCOUNT_FILE` | Google service account | `/app/service-account.json` |
| `GOOGLE_PACKAGE_NAME` | Android package name | `com.your.app` |

## Form Configuration

### Field Types

- `text`: Single-line text input
- `email`: Email input with validation
- `textarea`: Multi-line text input
- `number`: Numeric input
- `tel`: Phone number input
- `url`: URL input

### Field Properties

```yaml
fields:
  - name: "field_name"        # Required: field identifier
    type: "text"              # Required: field type
    required: true            # Optional: validation requirement
    label: "Display Name"     # Optional: human-readable label
    placeholder: "hint text"  # Optional: placeholder text
    validation: "regex"       # Optional: validation regex
```

### Actions

#### Email Action
```yaml
actions:
  - type: "send_email"
    config:
      template: "confirmation"
```

#### Google Play Tester Action
```yaml
actions:
  - type: "google_play_add_tester"
    config:
      track: "internal"  # or "alpha", "beta"
```

#### Log Action
```yaml
actions:
  - type: "log"
    config:
      message: "Custom log message"
```

#### Webhook Action (Coming Soon)
```yaml
actions:
  - type: "webhook"
    config:
      url: "https://api.example.com/webhook"
      method: "POST"
```

## Email Templates

The system includes built-in email templates:

- `confirmation`: General confirmation email
- `internal-testing`: Specific template for app testing signups

You can customize templates by creating HTML files in the `templates/` directory.

## Deployment

### Production Docker Setup

```bash
# Copy the production compose file
cp examples/docker-compose.production.yml docker-compose.yml

# Copy environment template
cp examples/.env.example .env

# Edit .env with your settings
nano .env

# Deploy
docker-compose up -d
```

### Kubernetes

Kubernetes manifests are available in the `examples/k8s/` directory.

## Monitoring

### Health Check

```bash
curl http://localhost:8080/health
```

### Metrics

The service provides structured logging and can be integrated with monitoring systems like Prometheus.

## Development

### Running Tests

```bash
go test ./...
```

### Building

```bash
go build -o pinepods-admin cmd/server/main.go
```

### Adding Custom Actions

1. Create a new action type in `internal/services/action_service.go`
2. Add the action configuration to your form config
3. Restart the service

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests
5. Submit a pull request

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.

## Support

- 📖 Documentation: Check this README and example configurations
- 💬 Discord: Join the PinePods Discord community
- 🐛 Issues: Report bugs on GitHub Issues
- 📧 Email: Contact the development team

## Changelog

### v1.0.0
- Initial release
- Basic form submission and processing
- Email integration with SMTP
- ntfy.sh notifications
- Google Play Console integration
- Docker and docker-compose support
- SQLite and PostgreSQL support