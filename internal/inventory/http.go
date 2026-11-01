package inventory

import (
	"embed"
	"net/http"
	"time"
)

//go:embed web/index.html
var pages embed.FS

func NewHandler(engine *Engine, mock *MockSupplier) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/mock/supplier/", mock)
	mux.HandleFunc("/api/health", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeJSON(w, 405, map[string]string{"error": "method not allowed"})
			return
		}
		writeJSON(w, 200, map[string]any{"status": "ok", "time": time.Now().UTC().Format(time.RFC3339), "mode": "local-demo"})
	})
	mux.HandleFunc("/api/items", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeJSON(w, 405, map[string]string{"error": "method not allowed"})
			return
		}
		items, _, _ := engine.Store.Snapshot()
		writeJSON(w, 200, map[string]any{"items": items, "count": len(items)})
	})
	mux.HandleFunc("/api/alerts", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeJSON(w, 405, map[string]string{"error": "method not allowed"})
			return
		}
		_, alerts, _ := engine.Store.Snapshot()
		writeJSON(w, 200, map[string]any{"alerts": alerts})
	})
	mux.HandleFunc("/api/runs", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeJSON(w, 405, map[string]string{"error": "method not allowed"})
			return
		}
		_, _, runs := engine.Store.Snapshot()
		writeJSON(w, 200, map[string]any{"runs": runs})
	})
	mux.HandleFunc("/api/sync", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSON(w, 405, map[string]string{"error": "method not allowed"})
			return
		}
		ctx := r.Context()
		// No request payload required: the configured source URL is fixed at startup.
		result, err := engine.Sync(ctx)
		if err != nil {
			writeJSON(w, http.StatusBadGateway, result)
			return
		}
		writeJSON(w, 200, result)
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" || r.Method != http.MethodGet {
			http.NotFound(w, r)
			return
		}
		b, err := pages.ReadFile("web/index.html")
		if err != nil {
			writeJSON(w, 500, map[string]string{"error": "UI unavailable"})
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; style-src 'self' 'unsafe-inline'; script-src 'self' 'unsafe-inline'; connect-src 'self'")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Write(b)
	})
	return mux
}
