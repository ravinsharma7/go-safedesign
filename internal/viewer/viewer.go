package viewer

import (
	_ "embed"
	"encoding/json"
	"net/http"

	"github.com/ravinsharma7/go-safedesign/internal/core"
	"github.com/ravinsharma7/go-safedesign/internal/source"
)

//go:embed assets/index.html
var indexHTML string

func Serve(addr string, graph core.Graph, sourceBase string) error {
	return http.ListenAndServe(addr, Handler(graph, sourceBase))
}

func Handler(graph core.Graph, sourceBase string) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(indexHTML))
	})
	mux.HandleFunc("/graph.json", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		_ = enc.Encode(graph)
	})
	mux.HandleFunc("/source", func(w http.ResponseWriter, r *http.Request) {
		block, err := source.ReadBlock(sourceBase, r.URL.Query().Get("file"), r.URL.Query().Get("range"))
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(block)
	})
	return mux
}
