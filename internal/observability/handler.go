package observability

import (
	"encoding/json"
	"net/http"
)

func JSONHandler(reg *Registry) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		snapshot := reg.Snapshot()
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(snapshot)
	}
}
