package httpapi

import (
	"example.com/port-yard-condition-service/health"
	"example.com/port-yard-condition-service/store"
	"io/fs"
	"net/http"
)

func NewHandler(st *store.Store, staticFS fs.FS) http.Handler {
	s := &server{store: st}
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", health.Handler("port-yard-condition-service"))
	mux.HandleFunc("/api/yard-zones", s.collection)
	mux.HandleFunc("/api/yard-zones/status", s.status)
	mux.Handle("/", http.FileServer(http.FS(staticFS)))
	return mux
}
