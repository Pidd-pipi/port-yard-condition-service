package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func opsServer006() http.Handler {
	return newOpsMux(newOpsService(nil))
}

func opsReq006(t *testing.T, h http.Handler, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(method, path, strings.NewReader(body)))
	return rec
}

func createActive006(t *testing.T, h http.Handler, id, owner string) string {
	rec := opsReq006(t, h, http.MethodPost, "/api/ops/records", fmt.Sprintf(`{"id":"%s","subject":"s","owner":"%s","priority":"high","status":"queued","labels":{"site":"east"}}`, id, owner))
	if rec.Code != http.StatusCreated {
		t.Fatalf("create %s status = %d", id, rec.Code)
	}
	rec = opsReq006(t, h, http.MethodPost, "/api/ops/records/"+id+"/status", `{"status":"active","expected_revision":1}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("activate %s status = %d", id, rec.Code)
	}
	var recd OpsRecord
	_ = json.Unmarshal(rec.Body.Bytes(), &recd)
	return recd.UpdatedAt
}

func getReport006(t *testing.T, h http.Handler) (OpsReport, int) {
	rec := opsReq006(t, h, http.MethodGet, "/api/ops/report", "")
	var rep OpsReport
	_ = json.Unmarshal(rec.Body.Bytes(), &rep)
	return rep, rec.Code
}

func TestOpsSearchPageOutOfRangeNoPanic(t *testing.T) {
	h := opsServer006()
	createActive006(t, h, "ops-pg-1", "alice")
	rec := opsReq006(t, h, http.MethodGet, "/api/ops/records?page=999", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("far page status = %d, want 200", rec.Code)
	}
	var page OpsPage
	_ = json.Unmarshal(rec.Body.Bytes(), &page)
	if len(page.Items) != 0 {
		t.Fatalf("far page returned %d items, want 0", len(page.Items))
	}
}

func TestOpsReportActiveOwnerFresh(t *testing.T) {
	h := opsServer006()
	createActive006(t, h, "ops-r-a1", "alice")
	createActive006(t, h, "ops-r-a2", "alice")
	createActive006(t, h, "ops-r-a3", "alice")
	createActive006(t, h, "ops-r-b1", "bob")
	rep, code := getReport006(t, h)
	if code != http.StatusOK {
		t.Fatalf("report1 status = %d", code)
	}
	if rep.ActiveOwner != "alice" {
		t.Fatalf("report1 busiest owner = %q, want alice", rep.ActiveOwner)
	}
	for _, id := range []string{"ops-r-a1", "ops-r-a2", "ops-r-a3"} {
		rec := opsReq006(t, h, http.MethodPost, "/api/ops/records/"+id+"/status", `{"status":"closed","expected_revision":2}`)
		if rec.Code != http.StatusOK {
			t.Fatalf("close %s status = %d", id, rec.Code)
		}
	}
	rep, code = getReport006(t, h)
	if code != http.StatusOK {
		t.Fatalf("report2 status = %d", code)
	}
	if rep.ActiveOwner != "bob" {
		t.Fatalf("report2 busiest owner = %q, want bob (stale counters)", rep.ActiveOwner)
	}
}

func TestOpsReportOldestActive(t *testing.T) {
	h := opsServer006()
	older := createActive006(t, h, "ops-o-1", "alice")
	newer := createActive006(t, h, "ops-o-2", "bob")
	rep, code := getReport006(t, h)
	if code != http.StatusOK {
		t.Fatalf("report status = %d", code)
	}
	if rep.OldestActive == newer {
		t.Fatalf("oldest_active = newest record %s; want the older stamp %s", rep.OldestActive, older)
	}
	if rep.OldestActive == "" || rep.OldestActive != older {
		t.Fatalf("oldest_active = %q, want %q", rep.OldestActive, older)
	}
}

func TestOpsReportFewOwnersNoPanic(t *testing.T) {
	h := opsServer006()
	createActive006(t, h, "ops-f-1", "alice")
	createActive006(t, h, "ops-f-2", "bob")
	rep, code := getReport006(t, h)
	if code != http.StatusOK {
		t.Fatalf("report status = %d, want 200", code)
	}
	if len(rep.TopOwners) != 2 {
		t.Fatalf("top owners len = %d, want 2", len(rep.TopOwners))
	}
}
