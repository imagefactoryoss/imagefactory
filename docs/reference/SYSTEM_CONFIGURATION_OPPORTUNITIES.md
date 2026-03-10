# System Configuration Opportunities

## Overview
This document identifies configuration opportunities that can be controlled through the system configuration system without requiring code changes. The goal is to make the system more flexible and adaptable to different deployment environments.

**Current model:** Queue-related settings map to dispatcher behavior. Builds are queued via `builds.status = 'queued'`.

## Current Configuration Infrastructure

### Existing System Configuration Types
- **LDAP**: Directory service integration settings
- **SMTP**: Email service configuration
- **General**: System-wide settings (name, description, admin/support emails, timezone, language)
- **Security**: Authentication and security policies

### Current Environment Variables (Already Configurable)
- Server settings (port, host, timeouts)
- Database connection settings
- Authentication (JWT, LDAP, SAML)
- Logging configuration
- NATS messaging
- Redis caching
- Build system settings
- SMTP email settings

## Identified Configuration Opportunities

### 1. Authentication & Security Settings
**Current State**: Hard-coded in `user/service.go`
```go
const (
    AccessTokenTTL  = 15 * time.Minute
    RefreshTokenTTL = 7 * 24 * time.Hour
)
```

**Opportunity**: Make token TTL configurable via system config
- Access token expiration (currently 15 minutes)
- Refresh token expiration (currently 7 days)
- Password policies (min length, complexity requirements)
- Account lockout settings (max attempts, lock duration)
- Session timeout settings

### 2. Build System Limits
**Current State**: Environment variables exist but could be enhanced
```yaml
IF_BUILD_DEFAULT_TIMEOUT=30m
IF_BUILD_MAX_CONCURRENT_JOBS=10
IF_BUILD_WORKER_POOL_SIZE=5
```

**Opportunity**: Make these runtime configurable
- Default build timeout per tenant
- Max concurrent builds per tenant
- Worker pool sizes
- Build queue limits
- Resource allocation limits

### 3. Rate Limiting & Throttling
**Current State**: Not implemented

**Opportunity**: Add configurable rate limits
- API request rate limits per user/IP
- Login attempt limits
- Password reset request limits
- Email sending limits

### 4. Feature Flags & Feature Toggles
**Current State**: Not implemented

**Opportunity**: Runtime feature control
- Enable/disable experimental features
- Tenant-specific feature availability
- Maintenance mode toggles
- Debug mode settings

### 5. Notification & Communication Settings
**Current State**: Basic SMTP config exists

**Opportunity**: Enhanced notification control
- Email templates (customizable)
- Notification frequency settings
- User preference overrides
- Multi-channel notifications (email, webhook, etc.)

### 6. Data Retention & Cleanup Policies
**Current State**: Not implemented

**Opportunity**: Configurable data lifecycle
- Audit log retention periods
- Build artifact cleanup policies
- User session cleanup
- Temporary file cleanup intervals

### 7. UI/UX Customization
**Current State**: Not implemented

**Opportunity**: Runtime UI customization
- System branding (logo, colors)
- Custom CSS/branding
- UI feature visibility
- Language/localization settings

### 8. Integration Settings
**Current State**: Basic external service configs

**Opportunity**: Enhanced integration control
- Webhook endpoints and secrets
- API rate limits for external services
- Retry policies and timeouts
- Circuit breaker configurations

### 9. Monitoring & Observability
**Current State**: Basic logging config

**Opportunity**: Enhanced monitoring
- Metrics collection intervals
- Alert thresholds
- Health check configurations
- Log level overrides per component

### 10. Performance Tuning
**Current State**: Some database connection pooling

**Opportunity**: Runtime performance control
- Connection pool sizes
- Cache TTL settings
- Background job concurrency
- Memory limits and thresholds

## Current Coverage

- Authentication settings (token TTLs, password policy, lockout controls)
- Build configuration (timeouts, concurrency, retention)
- Tenant-scoped system settings with admin UI

## Future Opportunities

- Rate limiting configuration
- Feature flags
- Data retention policies
- Monitoring and alerting configuration

## Configuration Schema Extensions

### Security Configuration Enhancement
```go
type SecurityConfig struct {
    // Existing fields...
    AccessTokenTTLMinutes     int  `json:"access_token_ttl_minutes"`
    RefreshTokenTTLHours      int  `json:"refresh_token_ttl_hours"`
    MaxLoginAttempts          int  `json:"max_login_attempts"`
    AccountLockDurationHours  int  `json:"account_lock_duration_hours"`
    PasswordMinLength         int  `json:"password_min_length"`
    RequireSpecialChars       bool `json:"require_special_chars"`
    RequireNumbers            bool `json:"require_numbers"`
    RequireUppercase          bool `json:"require_uppercase"`
    SessionTimeoutMinutes     int  `json:"session_timeout_minutes"`
}
```

### Build Configuration Enhancement
```go
type BuildConfig struct {
    DefaultTimeoutMinutes     int `json:"default_timeout_minutes"`
    MaxConcurrentJobs         int `json:"max_concurrent_jobs"`
    WorkerPoolSize            int `json:"worker_pool_size"`
    MaxQueueSize              int `json:"max_queue_size"`
    ArtifactRetentionDays     int `json:"artifact_retention_days"`
}
```

### New Configuration Types

#### Rate Limiting Configuration
```go
type RateLimitConfig struct {
    RequestsPerMinute    int `json:"requests_per_minute"`
    RequestsPerHour      int `json:"requests_per_hour"`
    BurstLimit           int `json:"burst_limit"`
    LoginAttemptsPerHour int `json:"login_attempts_per_hour"`
}
```

#### Feature Flags Configuration
```go
type FeatureFlagsConfig struct {
    ExperimentalFeatures bool `json:"experimental_features"`
    MaintenanceMode      bool `json:"maintenance_mode"`
    DebugMode           bool `json:"debug_mode"`
    AdvancedAnalytics   bool `json:"advanced_analytics"`
}
```

## Migration Strategy

1. Add new configuration keys to existing tenants with sensible defaults.
2. Update service constructors to accept configuration parameters where needed.
3. Implement fallback logic to environment variables for backward compatibility.
4. Add validation for configuration values, including build limits enforcement.
5. Update the admin UI to expose the relevant configuration options.

## Benefits

- Zero-downtime configuration changes through runtime configuration
- Environment-specific tuning without code deployments
- Tenant-specific customization for multi-tenant deployments
- Operational flexibility through admin-managed configuration
- Reduced deployment frequency for configuration-only changes

## Next Steps

1. **Operational Controls** - Add rate limiting, feature flags, and data retention policies
2. **Advanced Features** - UI customization, enhanced notifications, monitoring
3. **Testing & Validation** - Comprehensive testing of configuration system
4. **Documentation** - Complete admin guide for system configuration
