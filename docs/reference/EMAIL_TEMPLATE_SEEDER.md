# Email Template Seeder

## Overview

The Email Template Seeder provides a centralized, category-based system for managing email templates. Instead of embedding templates in database migrations, templates are organized in Go code for easy maintenance, version control, and dynamic seeding.

## Architecture

```
backend/
├── internal/
│   └── seeders/
│       ├── template_seeder.go           # Main seeder service
│       ├── template_model.go            # Shared models and constants
│       └── templates/
│           ├── templates.go             # Template aggregator
│           ├── user_management.go       # User-related templates
│           ├── project_management.go    # Project-related templates
│           ├── notifications.go         # Build/deployment notifications
│           └── invitations.go           # Invitation templates
└── cmd/
    └── seeder/
        └── main.go                      # CLI entry point
```

## Template Categories

### 1. User Management (`user_management`)
- `user_removed_from_tenant` - Notification when user is removed
- `user_added_to_tenant` - Notification when user is added
- `role_assigned` - Notification when role is assigned/changed
- `role_removed` - Notification when role is removed

### 2. Project Management (`project_management`)
- `project_created` - Notification when project is created
- `project_deleted` - Notification when project is deleted

### 3. Notifications (`notifications`)
- `build_started` - Notification when build starts
- `build_completed` - Notification when build succeeds
- `build_failed` - Notification when build fails

### 4. Invitations (`invitations`)
- `invitation_sent` - Invitation email to join tenant

### 5. Compliance (`compliance`)
*Reserved for future compliance notifications*

## Usage

### CLI Commands

#### Seed All Templates
```bash
cd backend
go run cmd/seeder/main.go \
  -action=seed \
  -host=localhost \
  -port=5432 \
  -user=postgres \
  -name=image_factory_dev
```

#### Seed Specific Category
```bash
go run cmd/seeder/main.go \
  -action=seed \
  -category=user_management \
  -host=localhost \
  -port=5432 \
  -user=postgres \
  -name=image_factory_dev
```

#### Validate Templates
Checks if all expected templates exist in the database:
```bash
go run cmd/seeder/main.go \
  -action=validate \
  -host=localhost \
  -port=5432 \
  -user=postgres \
  -name=image_factory_dev
```

#### View Statistics
```bash
go run cmd/seeder/main.go \
  -action=stats \
  -host=localhost \
  -port=5432 \
  -user=postgres \
  -name=image_factory_dev
```

#### Reset Templates
Clears and reseeds all templates (destructive):
```bash
go run cmd/seeder/main.go \
  -action=reset \
  -host=localhost \
  -port=5432 \
  -user=postgres \
  -name=image_factory_dev
```

#### Clear All Templates
Deletes all templates (destructive):
```bash
go run cmd/seeder/main.go \
  -action=clear \
  -host=localhost \
  -port=5432 \
  -user=postgres \
  -name=image_factory_dev
```

### Environment Variables

Set these instead of command-line flags:
```bash
export DB_HOST=localhost
export DB_PORT=5432
export DB_USER=postgres
export DB_PASSWORD=your_password
export DB_NAME=image_factory_dev

go run cmd/seeder/main.go -action=seed
```

### Programmatic Usage

```go
package main

import (
	"context"
	"database/sql"
	"image-factory/backend/internal/seeders"
	_ "github.com/lib/pq"
)

func main() {
	db, _ := sql.Open("postgres", "...")
	seeder := seeders.NewTemplateSeeder(db)
	ctx := context.Background()
	
	// Seed all templates
	seeder.SeedAllTemplates(ctx)
	
	// Seed specific category
	seeder.SeedTemplatesByCategory(ctx, "user_management")
	
	// Validate
	valid, missing, _ := seeder.ValidateTemplates(ctx)
	
	// Get stats
	stats, _ := seeder.TemplateStats(ctx)
	
	// Reset (clear + reseed)
	seeder.ResetTemplates(ctx)
}
```

## Adding New Templates

### Step 1: Choose or Create a Category
Look for existing category in `backend/internal/seeders/templates/`:
- `user_management.go`
- `project_management.go`
- `notifications.go`
- `invitations.go`

### Step 2: Add Template Definition
```go
// In the appropriate category file
func UserManagementTemplates() []EmailTemplateDefinition {
	return []EmailTemplateDefinition{
		{
			Name:        "template_name",
			Category:    "user_management",
			Subject:     "Email subject with {{.Variables}}",
			Description: "Short description of when this is sent",
			HTMLContent: `<!DOCTYPE html>
<!-- HTML template with {{.Variables}} -->
`,
			TextContent: `Plain text version`,
		},
		// ... other templates
	}
}
```

### Step 3: Add Template Constant (Optional but Recommended)
In `backend/internal/seeders/template_model.go`:
```go
const (
	TemplateYourTemplateName = "template_name"
)
```

### Step 4: Seed the Template
```bash
go run cmd/seeder/main.go -action=seed
```

## Template Variables

Use `{{.VariableName}}` syntax for substitutable values. Common variables:

**User Information:**
- `{{.UserName}}`
- `{{.UserEmail}}`
- `{{.UserId}}`

**Tenant Information:**
- `{{.TenantName}}`
- `{{.TenantId}}`

**Role Information:**
- `{{.Role}}`
- `{{.NewRole}}`
- `{{.OldRole}}`

**Action Metadata:**
- `{{.Timestamp}}`
- `{{.CreatedByName}}`
- `{{.RemovedByName}}`
- `{{.AssignedByName}}`

**Links:**
- `{{.DashboardUrl}}`
- `{{.ProjectUrl}}`
- `{{.BuildUrl}}`
- `{{.InviteLink}}`

**System:**
- `{{.CurrentYear}}`

## Database Schema

The seeder uses the existing `email_templates` table with columns:
- `id` (UUID, PK)
- `name` (VARCHAR, UNIQUE)
- `category` (VARCHAR)
- `subject` (VARCHAR)
- `html_content` (TEXT)
- `text_content` (TEXT)
- `description` (TEXT)
- `created_at` (TIMESTAMP)
- `updated_at` (TIMESTAMP)

## Best Practices

1. **Keep Templates Organized**: Group related templates in the same category file
2. **Use Constants**: Define template name constants to avoid typos
3. **Test Variables**: Ensure all template variables are properly documented
4. **HTML + Text**: Always provide both HTML and plain text versions
5. **Professional Styling**: Use consistent CSS styling across templates
6. **Keep DRY**: Extract common styles into a shared template if needed
7. **Document**: Use description field to document when templates are sent

## Integration with Application

### Sending Emails with Seeded Templates

```go
// In your email service
type NotificationService struct {
	db *sql.DB
	emailSvc EmailService
}

func (ns *NotificationService) NotifyUserRemoved(ctx context.Context, userID, tenantID string) error {
	// Get template from database
	template, err := ns.getTemplate(ctx, "user_removed_from_tenant")
	if err != nil {
		return err
	}
	
	// Render template with variables
	variables := map[string]interface{}{
		"UserName":      "John Doe",
		"TenantName":    "ACME Corp",
		"RemovedByName": "Admin",
		"Timestamp":     time.Now().Format("2006-01-02 15:04:05"),
		"CurrentYear":   time.Now().Year(),
	}
	
	// Send email
	return ns.emailSvc.Send(EmailMessage{
		To:           userEmail,
		Subject:      renderTemplate(template.Subject, variables),
		HTMLBody:     renderTemplate(template.HTMLContent, variables),
		TextBody:     renderTemplate(template.TextContent, variables),
	})
}
```

## Migration Path

### From Migration-Based to Seeder-Based

If you have existing templates in migrations:

1. **Extract templates** from migration files into template definition structs
2. **Remove** SQL INSERT statements from migrations
3. **Run seeder** to populate database
4. **Update application** to reference templates by name constant

Example:
```go
// Old: Template embedded in migration
// CREATE TABLE email_templates ...
// INSERT INTO email_templates VALUES (...)

// New: Template in code
func UserManagementTemplates() []EmailTemplateDefinition {
	return []EmailTemplateDefinition{
		{
			Name:        "user_removed_from_tenant",
			Category:    "user_management",
			// ...
		},
	}
}
```

## Troubleshooting

### "Missing required database configuration"
Ensure environment variables or command-line flags are set:
```bash
export DB_HOST=localhost
export DB_PORT=5432
export DB_USER=postgres
export DB_NAME=image_factory_dev
```

### "template validation failed"
Run validate action to see which templates are missing:
```bash
go run cmd/seeder/main.go -action=validate
```

Then reseed:
```bash
go run cmd/seeder/main.go -action=reset
```

### Template Not Sending
1. Check if template exists: `SELECT * FROM email_templates WHERE name = 'template_name'`
2. Verify template variables match exactly (case-sensitive)
3. Check email service logs for render errors

## Future Enhancements

- [ ] Template preview command with sample variables
- [ ] Template validation (check for unset variables, broken HTML)
- [ ] Support for template inheritance/includes
- [ ] Localization support (language-specific templates)
- [ ] Template versioning with rollback capability
- [ ] A/B testing support for subject lines

