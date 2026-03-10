package logger

import (
	"testing"

	"go.uber.org/zap"

	"github.com/srikarm/image-factory/internal/infrastructure/config"
)

func TestNewLoggerVariants(t *testing.T) {
	cases := []config.LoggerConfig{
		{Level: "debug", Format: "console", OutputPath: "stdout"},
		{Level: "info", Format: "json", OutputPath: "stdout"},
		{Level: "warn", Format: "json", OutputPath: "stdout"},
		{Level: "error", Format: "json", OutputPath: "stdout"},
		{Level: "unknown", Format: "unknown", OutputPath: "stdout"},
	}
	for i, cfg := range cases {
		l, err := NewLogger(cfg)
		if err != nil {
			t.Fatalf("case %d expected nil error, got %v", i, err)
		}
		if l == nil {
			t.Fatalf("case %d expected logger instance", i)
		}
		_ = l.Sync()
	}
}

func TestEnvLogger(t *testing.T) {
	t.Setenv("IF_SERVER_ENVIRONMENT", "production")
	if l := EnvLogger(); l == nil {
		t.Fatal("expected logger in production env")
	}

	t.Setenv("IF_SERVER_ENVIRONMENT", "development")
	if l := EnvLogger(); l == nil {
		t.Fatal("expected logger in non-production env")
	}
}

func TestFieldHelpersAndWithFields(t *testing.T) {
	base := zap.NewNop()
	with := WithFields(base,
		TenantField("t1"),
		BuildField("b1"),
		RequestIDField("r1"),
		UserField("u1"),
		ComponentField("api"),
		OperationField("create"),
		DurationField("1s"),
		HTTPMethodField("GET"),
		HTTPStatusField(200),
		URLField("/health"),
		IPField("127.0.0.1"),
		ErrorField(nil),
	)
	if with == nil {
		t.Fatal("expected logger with fields")
	}
}
