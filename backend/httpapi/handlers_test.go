package httpapi

import (
	"example.com/port-yard-condition-service/store"
	"example.com/port-yard-condition-service/web"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestYardRoutes(t *testing.T) {
	handler := NewHandler(store.New(), web.FS)
	get := httptest.NewRecorder()
	handler.ServeHTTP(get, httptest.NewRequest(http.MethodGet, "/api/yard-zones", nil))
	if get.Code != http.StatusOK || !strings.Contains(get.Body.String(), "YARD-A1") {
		t.Fatalf("collection: %d %s", get.Code, get.Body.String())
	}
	post := httptest.NewRecorder()
	handler.ServeHTTP(post, httptest.NewRequest(http.MethodPost, "/api/yard-zones/status", strings.NewReader(`{"id":"YARD-A1","status":"restricted"}`)))
	if post.Code != http.StatusOK || !strings.Contains(post.Body.String(), `"status":"restricted"`) {
		t.Fatalf("update: %d %s", post.Code, post.Body.String())
	}
}
func TestYardRejectsInvalidStatus(t *testing.T) {
	handler := NewHandler(store.New(), web.FS)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/api/yard-zones/status", strings.NewReader(`{"id":"YARD-A1","status":"busy"}`)))
	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", response.Code)
	}
}
