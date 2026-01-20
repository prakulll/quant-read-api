package controllers

import (
	"encoding/json"
	"net/http"
	"time"

	"quant-read-api/components"
)

func GetOptionExpiries(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	q := r.URL.Query()
	underlying := q.Get("underlying")

	if underlying == "" {
		http.Error(w, "missing underlying", http.StatusBadRequest)
		return
	}

	var fromPtr *time.Time
	var toPtr *time.Time

	if fromStr := q.Get("from"); fromStr != "" {
		from, err := time.Parse("2006-01-02", fromStr)
		if err != nil {
			http.Error(w, "invalid from date", http.StatusBadRequest)
			return
		}
		fromPtr = &from
	}

	if toStr := q.Get("to"); toStr != "" {
		to, err := time.Parse("2006-01-02", toStr)
		if err != nil {
			http.Error(w, "invalid to date", http.StatusBadRequest)
			return
		}
		toPtr = &to
	}

	expiries, err := components.GetOptionExpiries(
		underlying,
		fromPtr,
		toPtr,
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]any{
		"expiries": expiries,
	})
}
