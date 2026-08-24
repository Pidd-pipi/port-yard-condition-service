package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func opsServer005() (*OpsService, http.Handler) {
	svc := newOpsService(nil)
	return svc, newOpsMux(svc)
}

func opsReq005(t *testing.T, h http.Handler, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(method, path, strings.NewReader(body)))
	return rec
}

func create005(t *testing.T, h http.Handler, id string) {
	rec := opsReq005(t, h, http.MethodPost, "/api/ops/records", `{"id":"`+id+`","subject":"s","owner":"yard","priority":"high","status":"queued","labels":{"site":"east"}}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create %s status = %d", id, rec.Code)
	}
}

func TestOpsGetIsolated(t *testing.T) {
	svc, h := opsServer005()
	create005(t, h, "ops-g-1")
	first, err := svc.Get(context.Background(), "ops-g-1")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	first.Labels["site"] = "hacked"
	second, err := svc.Get(context.Background(), "ops-g-1")
	if err != nil {
		t.Fatalf("get again: %v", err)
	}
	if second.Labels["site"] == "hacked" {
		t.Fatal("mutating the returned record changed the stored labels")
	}
}

func TestOpsListIsolated(t *testing.T) {
	svc, h := opsServer005()
	create005(t, h, "ops-l-1")
	items, err := svc.Search(context.Background(), OpsQuery{})
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(items.Items) == 0 {
		t.Fatal("no items")
	}
	items.Items[0].Labels["owner"] = "hacked"
	again, err := svc.Get(context.Background(), "ops-l-1")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if again.Labels["owner"] == "hacked" {
		t.Fatal("mutating a listed item changed the stored labels")
	}
}

func TestOpsCreateExternalMapIsolated(t *testing.T) {
	svc, _ := opsServer005()
	external := map[string]string{"site": "east"}
	record := OpsRecord{ID: "ops-x-2", Subject: "x", Owner: "yard", Priority: "high", Status: "queued", Labels: external}
	if _, err := svc.Create(context.Background(), record); err != nil {
		t.Fatalf("create: %v", err)
	}
	external["site"] = "west"
	external["extra"] = "boom"
	got, err := svc.Get(context.Background(), "ops-x-2")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Labels["site"] != "east" || got.Labels["extra"] != "" {
		t.Fatalf("stored labels changed by external mutation: %v", got.Labels)
	}
}

func TestOpsUpdateExternalMapIsolated(t *testing.T) {
	svc, h := opsServer005()
	create005(t, h, "ops-u-1")
	external := map[string]string{"site": "east"}
	record := OpsRecord{ID: "ops-u-1", Subject: "u", Owner: "yard", Priority: "high", Status: "queued", Labels: external}
	if _, err := svc.Update(context.Background(), "ops-u-1", 1, record); err != nil {
		t.Fatalf("update: %v", err)
	}
	external["site"] = "west"
	got, err := svc.Get(context.Background(), "ops-u-1")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Labels["site"] != "east" {
		t.Fatalf("stored labels changed by external update mutation: %v", got.Labels)
	}
}

func TestOpsCloneDeep(t *testing.T) {
	orig := OpsRecord{ID: "r", Labels: map[string]string{"site": "east"}}
	dup := orig.Clone()
	dup.Labels["site"] = "west"
	if orig.Labels["site"] != "east" {
		t.Fatal("Clone shares the labels map")
	}
}
