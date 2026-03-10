package logger

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"github.com/srikarm/image-factory/internal/infrastructure/config"
)

// NewLogger creates a new logger instance
func NewLogger(cfg config.LoggerConfig) (*zap.Logger, error) {
	var zapConfig zap.Config

	switch cfg.Level {
	case "debug":
		zapConfig = zap.NewDevelopmentConfig()
	case "info", "warn", "error":
		zapConfig = zap.NewProductionConfig()
	default:
		zapConfig = zap.NewProductionConfig()
	}

	// Set log level
	switch cfg.Level {
	case "debug":
		zapConfig.Level = zap.NewAtomicLevelAt(zapcore.DebugLevel)
	case "info":
		zapConfig.Level = zap.NewAtomicLevelAt(zapcore.InfoLevel)
	case "warn":
		zapConfig.Level = zap.NewAtomicLevelAt(zapcore.WarnLevel)
	case "error":
		zapConfig.Level = zap.NewAtomicLevelAt(zapcore.ErrorLevel)
	default:
		zapConfig.Level = zap.NewAtomicLevelAt(zapcore.InfoLevel)
	}

	// Set output format
	if cfg.Format == "console" {
		zapConfig.Encoding = "console"
		zapConfig.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
		zapConfig.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	} else {
		zapConfig.Encoding = "json"
		zapConfig.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
		zapConfig.EncoderConfig.EncodeLevel = zapcore.LowercaseLevelEncoder
	}

	// Set output path
	if cfg.OutputPath != "" && cfg.OutputPath != "stdout" {
		zapConfig.OutputPaths = []string{cfg.OutputPath}
		zapConfig.ErrorOutputPaths = []string{cfg.OutputPath}
	} else {
		zapConfig.OutputPaths = []string{"stdout"}
		zapConfig.ErrorOutputPaths = []string{"stderr"}
	}

	// Add caller information
	zapConfig.EncoderConfig.CallerKey = "caller"
	zapConfig.EncoderConfig.EncodeCaller = zapcore.ShortCallerEncoder

	logger, err := zapConfig.Build(zap.AddCallerSkip(1))
	if err != nil {
		return nil, err
	}

	return logger, nil
}

// NewConsoleLogger creates a console logger for development
func NewConsoleLogger() *zap.Logger {
	config := zap.NewDevelopmentConfig()
	config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	
	logger, err := config.Build()
	if err != nil {
		// Fallback to a simple logger
		logger = zap.NewExample()
	}
	
	return logger
}

// WithFields adds structured fields to the logger
func WithFields(logger *zap.Logger, fields ...zap.Field) *zap.Logger {
	return logger.With(fields...)
}

// TenantField creates a tenant field for logging
func TenantField(tenantID string) zap.Field {
	return zap.String("tenant_id", tenantID)
}

// BuildField creates a build field for logging
func BuildField(buildID string) zap.Field {
	return zap.String("build_id", buildID)
}

// ErrorField creates an error field for logging
func ErrorField(err error) zap.Field {
	return zap.Error(err)
}

// RequestIDField creates a request ID field for logging
func RequestIDField(requestID string) zap.Field {
	return zap.String("request_id", requestID)
}

// UserField creates a user field for logging
func UserField(userID string) zap.Field {
	return zap.String("user_id", userID)
}

// ComponentField creates a component field for logging
func ComponentField(component string) zap.Field {
	return zap.String("component", component)
}

// OperationField creates an operation field for logging
func OperationField(operation string) zap.Field {
	return zap.String("operation", operation)
}

// DurationField creates a duration field for logging
func DurationField(duration interface{}) zap.Field {
	return zap.Any("duration", duration)
}

// HTTPMethodField creates an HTTP method field for logging
func HTTPMethodField(method string) zap.Field {
	return zap.String("http_method", method)
}

// HTTPStatusField creates an HTTP status field for logging
func HTTPStatusField(status int) zap.Field {
	return zap.Int("http_status", status)
}

// URLField creates a URL field for logging
func URLField(url string) zap.Field {
	return zap.String("url", url)
}

// IPField creates an IP field for logging
func IPField(ip string) zap.Field {
	return zap.String("ip", ip)
}

// EnvLogger gets logger based on environment
func EnvLogger() *zap.Logger {
	env := os.Getenv("IF_SERVER_ENVIRONMENT")
	if env == "production" {
		config := zap.NewProductionConfig()
		logger, err := config.Build()
		if err != nil {
			return zap.NewExample()
		}
		return logger
	}
	return NewConsoleLogger()
}