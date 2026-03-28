package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap/zaptest"

	"github.com/srikarm/image-factory/internal/domain/user"
)

// MockUserService is a mock implementation of user.Service
type MockUserService struct {
	mock.Mock
}

func (m *MockUserService) Login(ctx context.Context, req user.LoginRequest) (*user.AuthResult, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*user.AuthResult), args.Error(1)
}

func (m *MockUserService) RefreshToken(ctx context.Context, refreshToken string) (*user.AuthResult, error) {
	args := m.Called(ctx, refreshToken)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*user.AuthResult), args.Error(1)
}

func (m *MockUserService) ValidateToken(ctx context.Context, accessToken string) (*user.User, error) {
	args := m.Called(ctx, accessToken)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*user.User), args.Error(1)
}

func (m *MockUserService) CreateUser(ctx context.Context, tenantID uuid.UUID, email, firstName, lastName, password string) (*user.User, error) {
	args := m.Called(ctx, tenantID, email, firstName, lastName, password)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*user.User), args.Error(1)
}

func (m *MockUserService) GetUserByID(ctx context.Context, id uuid.UUID) (*user.User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*user.User), args.Error(1)
}

func (m *MockUserService) GetUsersByTenantID(ctx context.Context, tenantID uuid.UUID) ([]*user.User, error) {
	args := m.Called(ctx, tenantID)
	return args.Get(0).([]*user.User), args.Error(1)
}

func (m *MockUserService) UpdateUser(ctx context.Context, user *user.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockUserService) DeleteUser(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockUserService) ChangePassword(ctx context.Context, userID uuid.UUID, newPassword string) error {
	args := m.Called(ctx, userID, newPassword)
	return args.Error(0)
}

func (m *MockUserService) EnableMFA(ctx context.Context, userID uuid.UUID, mfaType user.MFAType, secret string) error {
	args := m.Called(ctx, userID, mfaType, secret)
	return args.Error(0)
}

func (m *MockUserService) DisableMFA(ctx context.Context, userID uuid.UUID) error {
	args := m.Called(ctx, userID)
	return args.Error(0)
}

func (m *MockUserService) VerifyEmail(ctx context.Context, userID uuid.UUID) error {
	args := m.Called(ctx, userID)
	return args.Error(0)
}

func (m *MockUserService) UnlockAccount(ctx context.Context, userID uuid.UUID) error {
	args := m.Called(ctx, userID)
	return args.Error(0)
}

func (m *MockUserService) GetTotalUserCount(ctx context.Context) (int, error) {
	args := m.Called(ctx)
	return args.Int(0), args.Error(1)
}

func (m *MockUserService) GetActiveUserCount(ctx context.Context, days int) (int, error) {
	args := m.Called(ctx, days)
	return args.Int(0), args.Error(1)
}

func TestAuthMiddleware_Authenticate(t *testing.T) {
	logger := zaptest.NewLogger(t)

	t.Run("missing authorization header", func(t *testing.T) {
		middleware := NewAuthMiddleware(nil, nil, nil, logger, nil)

		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()

		testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		handler := middleware.Authenticate(testHandler)
		handler.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("invalid authorization header format", func(t *testing.T) {
		middleware := NewAuthMiddleware(nil, nil, nil, logger, nil)

		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Authorization", "InvalidFormat")
		w := httptest.NewRecorder()

		testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		handler := middleware.Authenticate(testHandler)
		handler.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	tests := []struct {
		name           string
		authHeader     string
		setupMock      func()
		expectedStatus int
		expectAuthCtx  bool
	}{
		{
			name:           "missing authorization header",
			authHeader:     "",
			setupMock:      func() {},
			expectedStatus: http.StatusUnauthorized,
			expectAuthCtx:  false,
		},
		{
			name:           "invalid authorization header format",
			authHeader:     "InvalidFormat",
			setupMock:      func() {},
			expectedStatus: http.StatusUnauthorized,
			expectAuthCtx:  false,
		},
		{
			name:           "invalid bearer token format",
			authHeader:     "Bearer invalid.token.here",
			setupMock:      func() {},
			expectedStatus: http.StatusUnauthorized,
			expectAuthCtx:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/test", nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}

			w := httptest.NewRecorder()

			// Create a test handler that checks for auth context
			testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if tt.expectAuthCtx {
					authCtx, ok := GetAuthContext(r)
					assert.True(t, ok, "Expected auth context to be present")
					assert.NotNil(t, authCtx, "Auth context should not be nil")
				}
				w.WriteHeader(http.StatusOK)
			})

			// If an Authorization header with Bearer token exists, provide a mock
			// auth service so ValidateToken can be called without panicking.
			var authSvc userTokenValidator
			if strings.HasPrefix(tt.authHeader, "Bearer ") {
				mockUser := &MockUserService{}
				mockUser.On("ValidateToken", mock.Anything, mock.Anything).Return(nil, errors.New("invalid token"))
				authSvc = mockUser
			}

			// Apply middleware
			authMiddleware := NewAuthMiddleware(authSvc, nil, nil, logger, nil)
			handler := authMiddleware.Authenticate(testHandler)
			handler.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}

	// New tests for tenant header behavior
	t.Run("system admin missing X-Tenant-ID returns 400", func(t *testing.T) {
		logger := zaptest.NewLogger(t)
		mockUser := &MockUserService{}
		usr, _ := user.NewUser("admin@imgfactory.com", "Admin", "User", "password")
		mockUser.On("ValidateToken", mock.Anything, "valid-token").Return(usr, nil)

		middleware := NewAuthMiddleware(mockUser, nil, nil, logger, nil)
		// mark user as system admin in cache
		middleware.cache.setAdminStatus(usr.ID(), true)

		req := httptest.NewRequest("GET", "/admin", nil)
		req.Header.Set("Authorization", "Bearer valid-token")
		w := httptest.NewRecorder()

		testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		handler := middleware.Authenticate(testHandler)
		handler.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("non-admin missing X-Tenant-ID returns 400", func(t *testing.T) {
		logger := zaptest.NewLogger(t)
		mockUser := &MockUserService{}
		usr, _ := user.NewUser("user@imgfactory.com", "Normal", "User", "password")
		mockUser.On("ValidateToken", mock.Anything, "valid-token").Return(usr, nil)

		middleware := NewAuthMiddleware(mockUser, nil, nil, logger, nil)

		req := httptest.NewRequest("GET", "/tenant", nil)
		req.Header.Set("Authorization", "Bearer valid-token")
		w := httptest.NewRecorder()

		testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		handler := middleware.Authenticate(testHandler)
		handler.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestAuthMiddleware_RequirePermission(t *testing.T) {
	logger := zaptest.NewLogger(t)

	t.Run("no auth context", func(t *testing.T) {
		middleware := NewAuthMiddleware(nil, nil, nil, logger, nil)

		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()

		testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		handler := middleware.RequirePermission(nil, "test", "read")(testHandler)
		handler.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}

func TestAuthMiddleware_OptionalAuth(t *testing.T) {
	logger := zaptest.NewLogger(t)

	t.Run("no authorization header", func(t *testing.T) {
		middleware := NewAuthMiddleware(nil, nil, nil, logger, nil)

		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()

		authCtxPresent := false
		testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if _, ok := GetAuthContext(r); ok {
				authCtxPresent = true
			}
			w.WriteHeader(http.StatusOK)
		})

		handler := middleware.OptionalAuth(testHandler)
		handler.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.False(t, authCtxPresent, "Auth context should not be present")
	})
}

func TestGetAuthContext(t *testing.T) {
	t.Run("no auth context", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		authCtx, ok := GetAuthContext(req)
		assert.False(t, ok)
		assert.Nil(t, authCtx)
	})

	t.Run("with auth context", func(t *testing.T) {
		authCtx := &AuthContext{
			UserID:   uuid.New(),
			TenantID: uuid.New(),
			Email:    "test@example.com",
		}

		req := httptest.NewRequest("GET", "/test", nil)
		ctx := context.WithValue(req.Context(), "auth", authCtx)
		req = req.WithContext(ctx)

		retrievedCtx, ok := GetAuthContext(req)
		assert.True(t, ok)
		assert.NotNil(t, retrievedCtx)
		assert.Equal(t, authCtx.UserID, retrievedCtx.UserID)
		assert.Equal(t, authCtx.TenantID, retrievedCtx.TenantID)
		assert.Equal(t, "test@example.com", retrievedCtx.Email)
	})
}
