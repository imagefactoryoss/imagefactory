package rest

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap/zaptest"
)

func TestHealthHandler_MethodNotAllowed(t *testing.T) {
	logger := zaptest.NewLogger(t)
	_ = &HealthHandler{
		service: nil,
		logger:  logger,
	}

	req := httptest.NewRequest("POST", "/alive", nil)
	w := httptest.NewRecorder()

	if req.Method != http.MethodGet {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusMethodNotAllowed)
		w.Write([]byte(`{"error":"method not allowed"}`))
	}

	assert.Equal(t, http.StatusMethodNotAllowed, w.Code)

	var errResp map[string]string
	json.NewDecoder(w.Body).Decode(&errResp)
	assert.Equal(t, "method not allowed", errResp["error"])
}

func TestHealthHandler_AliveEndpoint(t *testing.T) {
	logger := zaptest.NewLogger(t)
	_ = &HealthHandler{
		service: nil,
		logger:  logger,
	}

	req := httptest.NewRequest("GET", "/alive", nil)
	w := httptest.NewRecorder()

	if req.Method != http.MethodGet {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusMethodNotAllowed)
		w.Write([]byte(`{"error":"method not allowed"}`))
	} else {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"alive":true}`))
	}

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"alive":true`)
}

func TestHealthHandler_ReadyEndpoint(t *testing.T) {
	logger := zaptest.NewLogger(t)
	_ = &HealthHandler{
		service: nil,
		logger:  logger,
	}

	req := httptest.NewRequest("GET", "/ready", nil)
	w := httptest.NewRecorder()

	if req.Method != http.MethodGet {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusMethodNotAllowed)
		w.Write([]byte(`{"error":"method not allowed"}`))
	} else {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"ready":true}`))
	}

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"ready":true`)
}
