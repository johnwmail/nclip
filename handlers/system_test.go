package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestSystemHandler_Health(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Create handler
	handler := NewSystemHandler()

	// Setup request
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	// Execute handler
	handler.Health(c)

	// Check status code
	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	// Check response body
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	if err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	// Check expected fields
	expectedStatus := "ok"
	if status, ok := response["status"]; !ok {
		t.Errorf("Expected 'status' field in response")
	} else if status != expectedStatus {
		t.Errorf("Expected status '%s', got '%v'", expectedStatus, status)
	}

	expectedService := "nclip"
	if service, ok := response["service"]; !ok {
		t.Errorf("Expected 'service' field in response")
	} else if service != expectedService {
		t.Errorf("Expected service '%s', got '%v'", expectedService, service)
	}
}
