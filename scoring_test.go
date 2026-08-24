package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func opsServer004() http.Handler {
	return newOpsMux(newOpsService(seedOpsRecords()))
}

func opsReq004(t *testing.T, h http.Handler, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(method, path, strings.NewReader(body)))
	return rec
}

func TestOpsDuplicateCreateConflict(t *testing.T) {
	h := opsServer004()
	body := `{"id":"ops-dup-1","subject":"dup","owner":"yard","priority":"high","status":"queued","labels":{"site":"east"}}`
	rec := opsReq004(t, h, http.MethodPost, "/api/ops/records", body)
	if rec.Code != http.StatusCreated {
		t.Fatalf("first create status = %d, want 201", rec.Code)
	}
	rec = opsReq004(t, h, http.MethodPost, "/api/ops/records", body)
	if rec.Code != http.StatusConflict {
		t.Fatalf("duplicate create status = %d, want 409", rec.Code)
	}
}

func TestOpsTransitionStaleRevisionRejected(t *testing.T) {
	h := opsServer004()
	opsReq004(t, h, http.MethodPost, "/api/ops/records", `{"id":"ops-st-1","subject":"s","owner":"yard","priority":"high","status":"queued","labels":{"site":"east"}}`)
	rec := opsReq004(t, h, http.MethodPost, "/api/ops/records/ops-st-1/status", `{"status":"active","expected_revision":1}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("first transition status = %d, want 200", rec.Code)
	}
	rec = opsReq004(t, h, http.MethodPost, "/api/ops/records/ops-st-1/status", `{"status":"paused","expected_revision":1}`)
	if rec.Code != http.StatusConflict {
		t.Fatalf("stale transition status = %d, want 409", rec.Code)
	}
}

func TestOpsUpdateStaleRevisionRejected(t *testing.T) {
	h := opsServer004()
	opsReq004(t, h, http.MethodPost, "/api/ops/records", `{"id":"ops-up-1","subject":"s","owner":"yard","priority":"high","status":"queued","labels":{"site":"east"}}`)
	body := `{"subject":"v2","owner":"yard","priority":"critical","status":"queued","labels":{"site":"east"},"expected_revision":1}`
	rec := opsReq004(t, h, http.MethodPut, "/api/ops/records/ops-up-1", body)
	if rec.Code != http.StatusOK {
		t.Fatalf("first update status = %d, want 200", rec.Code)
	}
	rec = opsReq004(t, h, http.MethodPut, "/api/ops/records/ops-up-1", body)
	if rec.Code != http.StatusConflict {
		t.Fatalf("stale update status = %d, want 409", rec.Code)
	}
}

func TestOpsDeleteMissingNotFound(t *testing.T) {
	h := opsServer004()
	rec := opsReq004(t, h, http.MethodDelete, "/api/ops/records/ops-nope-1", "")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("delete missing status = %d, want 404", rec.Code)
	}
}
