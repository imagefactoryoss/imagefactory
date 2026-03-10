# Environment File Loading for Image Factory Server

## Overview

The Image Factory Server now supports loading environment variables from custom `.env` files using the `--env` command-line flag. This enhancement provides flexible configuration management for different deployment environments.

## Features

### Flexible Environment Loading
- Load environment variables from any `.env` file
- Support for absolute and relative file paths
- Command-line flag override capability
- Backward compatibility with existing environment setups

### Error Handling
- Comprehensive file existence validation
- Clear error messages for missing files
- Graceful fallback to system environment variables

### Migration Support
- `--migrate-only` flag for database-only operations
- Perfect for CI/CD pipeline integration
- Environment-specific migration execution

## Usage

### Basic Usage

```bash
# Start server with default environment
go run cmd/server/main.go

# Start server with development environment
go run cmd/server/main.go --env .env.development

# Start server with production environment
go run cmd/server/main.go --env .env.production

# Start server with custom environment file
go run cmd/server/main.go --env /path/to/custom.env
```

### Migration-Only Mode

```bash
# Run migrations only with development environment
go run cmd/server/main.go --env .env.development --migrate-only

# Run migrations only with production environment
go run cmd/server/main.go --env .env.production --migrate-only
```

### Path Resolution

```bash
# Relative paths (resolved from current working directory)
go run cmd/server/main.go --env ../.env.development
go run cmd/server/main.go --env ../../config/production.env

# Absolute paths
go run cmd/server/main.go --env /etc/image-factory/production.env
go run cmd/server/main.go --env /home/user/config/custom.env
```

## Command-Line Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--env` | string | "" | Path to environment file (optional) |
| `--migrate-only` | bool | false | Run database migrations only and exit |
| `--help` | bool | false | Show help information |

## Environment File Format

Environment files should follow the standard `.env` format:

```bash
# Image Factory Configuration
IF_SERVER_HOST=0.0.0.0
IF_SERVER_PORT=8080
IF_SERVER_ENVIRONMENT=development

# Database Configuration
IF_DATABASE_HOST=localhost
IF_DATABASE_PORT=5432
IF_DATABASE_NAME=image_factory_dev
IF_DATABASE_USER=postgres
IF_DATABASE_PASSWORD=postgres
IF_DATABASE_SSL_MODE=disable

# Authentication Configuration
IF_AUTH_JWT_SECRET=your-secret-key
IF_AUTH_JWT_EXPIRATION=24h

# Storage Configuration
IF_STORAGE_TYPE=s3
IF_STORAGE_BUCKET=my-image-factory-bucket
```

## Loading Priority

Environment variables are loaded in the following priority order (highest to lowest):

1. **Command-line specified `.env` file** (`--env` flag)
2. **System environment variables**
3. **Default configuration values**
4. **YAML configuration files** (if present)

## Integration Examples

### Development Workflow

```bash
# Frontend terminal
npm run dev

# Backend terminal (development)
cd backend
go run cmd/server/main.go --env ../.env.development

# Database migration terminal (development)
cd backend
go run cmd/server/main.go --env ../.env.development --migrate-only
```

### Production Deployment

```bash
# Production server startup
cd /opt/image-factory/backend
go run cmd/server/main.go --env /etc/image-factory/production.env

# Production migration (CI/CD)
cd /opt/image-factory/backend
go run cmd/server/main.go --env /etc/image-factory/production.env --migrate-only
```

### CI/CD Pipeline Integration

```yaml
# GitHub Actions example
- name: Run Database Migrations
  run: |
    cd backend
    go run cmd/server/main.go --env ../.env.production --migrate-only

- name: Start Application Server
  run: |
    cd backend
    go run cmd/server/main.go --env ../.env.production
```

## VS Code Tasks Integration

The VS Code tasks have been updated to support environment file loading:

```json
{
    "label": "Start Backend Server",
    "type": "shell",
    "command": "cd backend && go run cmd/server/main.go --env ../.env.development",
    "group": "build",
    "isBackground": true
}
```

## Error Handling

### File Not Found
```bash
$ go run cmd/server/main.go --env /nonexistent/file.env
2025/10/25 23:00:55 Environment file not found: /nonexistent/file.env (stat /nonexistent/file.env: no such file or directory)
exit status 1
```

### Invalid File Format
The server will continue with system environment variables if the `.env` file has parsing issues, but will log warnings about problematic lines.

## Security Considerations

### Best Practices
- Store sensitive `.env` files outside the web root
- Use appropriate file permissions (600) for production `.env` files
- Never commit `.env` files to version control
- Use absolute paths in production environments

### File Permission Example
```bash
# Set secure permissions for production environment file
chmod 600 /etc/image-factory/production.env
chown app:app /etc/image-factory/production.env
```

## Troubleshooting

### Common Issues

1. **File not found errors**
   - Verify the file path is correct
   - Check file permissions
   - Use absolute paths for production

2. **Environment variables not loaded**
   - Verify the file format (KEY=value)
   - Check for syntax errors in the file
   - Ensure no spaces around the `=` sign

3. **Migration failures**
   - Check database connectivity with loaded environment
   - Verify database credentials in the environment file
   - Ensure database exists before running migrations

### Debug Commands

```bash
# Verify environment file contents
cat .env.development

# Test environment loading (dry run)
go run cmd/server/main.go --env .env.development --migrate-only

# Check loaded environment variables
env | grep IF_
```

## Testing

Run the included test suite to verify environment file loading:

```bash
cd backend
go test ./cmd/server -v
```

Or use the demo script:

```bash
cd backend
./scripts/demo-env-loading.sh
```

## Backward Compatibility

Full backward compatibility is maintained:
- Existing deployments continue to work without changes
- System environment variables still take precedence
- YAML configuration files continue to work
- No breaking changes to existing functionality

## Future Enhancements

- **Environment validation**: Validate required environment variables
- **Hot reloading**: Reload environment files without restart
- **Encrypted environments**: Support for encrypted `.env` files
- **Multiple file support**: Load from multiple environment files

---

This guide is intended as a practical reference for running the server with explicit environment files across development and deployment environments.
