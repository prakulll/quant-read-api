package controllers

import (
	"encoding/json"
	"net/http"
	"quant-read-api/components"
	"quant-read-api/models"
	"strconv"
	"strings"
	"sync"
	"time"
)

var ist, _ = time.LoadLocation("Asia/Kolkata")

// unified datetime parser for entire API
func parseAPITime(ts string) (time.Time, error) {
	ts = strings.ReplaceAll(ts, " ", "+")

	// RFC3339 with timezone
	if t, err := time.Parse(time.RFC3339, ts); err == nil {
		return t, nil
	}

	// fallback: assume IST
	return time.ParseInLocation("2006-01-02T15:04:05", ts, ist)
}

func GetOptionContractsByPremium(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	q := r.URL.Query()

	underlying := q.Get("underlying")
	optionType := q.Get("option_type")
	expiryMode := q.Get("expiry_mode")
	if expiryMode == "" {
		expiryMode = "nearest"
	}

	if underlying == "" || optionType == "" {
		http.Error(w, "underlying and option_type are required", http.StatusBadRequest)
		return
	}

	targetPremium, err := strconv.ParseFloat(q.Get("target_premium"), 64)
	if err != nil {
		http.Error(w, "invalid target_premium", http.StatusBadRequest)
		return
	}

	tolerance := 5.0
	if t := q.Get("tolerance"); t != "" {
		tolerance, err = strconv.ParseFloat(t, 64)
		if err != nil {
			http.Error(w, "invalid tolerance", http.StatusBadRequest)
			return
		}
	}

	fromStr := q.Get("from")
	toStr := q.Get("to")

	if fromStr == "" || toStr == "" {
		http.Error(w, "from and to timestamps are required", http.StatusBadRequest)
		return
	}

	from, err := parseAPITime(fromStr)
	if err != nil {
		http.Error(w, "invalid from timestamp", http.StatusBadRequest)
		return
	}

	to, err := parseAPITime(toStr)
	if err != nil {
		http.Error(w, "invalid to timestamp", http.StatusBadRequest)
		return
	}

	var response interface{}

	if optionType == "BOTH" {

		ce, err := components.GetOptionContractsByPremium(
			underlying,
			"CE",
			targetPremium,
			tolerance,
			expiryMode,
			from,
			to,
			50,
		)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		pe, err := components.GetOptionContractsByPremium(
			underlying,
			"PE",
			targetPremium,
			tolerance,
			expiryMode,
			from,
			to,
			50,
		)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		response = map[string]interface{}{
			"CE": ce,
			"PE": pe,
		}

	} else {

		candidates, err := components.GetOptionContractsByPremium(
			underlying,
			optionType,
			targetPremium,
			tolerance,
			expiryMode,
			from,
			to,
			50,
		)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		response = candidates
	}

	json.NewEncoder(w).Encode(response)
}

/*
=============================
V2 — CONCURRENT BY EXPIRY
=============================
GET /api/v2/options/contracts/by-premium
Supports:
- expiry=2025-11-04,2025-11-11
- expiry_mode=nearest|all (fallback)
*/
func GetOptionContractsByPremiumV2(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	q := r.URL.Query()

	underlying := q.Get("underlying")
	optionType := q.Get("option_type")

	if underlying == "" || optionType == "" {
		http.Error(w, "underlying and option_type are required", http.StatusBadRequest)
		return
	}

	targetPremium, err := strconv.ParseFloat(q.Get("target_premium"), 64)
	if err != nil {
		http.Error(w, "invalid target_premium", http.StatusBadRequest)
		return
	}

	tolerance := 5.0
	if t := q.Get("tolerance"); t != "" {
		tolerance, err = strconv.ParseFloat(t, 64)
		if err != nil {
			http.Error(w, "invalid tolerance", http.StatusBadRequest)
			return
		}
	}

	expiryMode := q.Get("expiry_mode")
	if expiryMode == "" {
		expiryMode = "nearest"
	}

	fromStr := q.Get("from")
	toStr := q.Get("to")

	if fromStr == "" || toStr == "" {
		http.Error(w, "from and to timestamps required", http.StatusBadRequest)
		return
	}

	from, err := parseAPITime(fromStr)
	if err != nil {
		http.Error(w, "invalid from timestamp", http.StatusBadRequest)
		return
	}

	to, err := parseAPITime(toStr)
	if err != nil {
		http.Error(w, "invalid to timestamp", http.StatusBadRequest)
		return
	}

	// -----------------------
	// Resolve expiries
	// -----------------------
	var expiries []string

	if expStr := q.Get("expiry"); expStr != "" {
		expiries = strings.Split(expStr, ",")
	} else {
		expiries, err = components.GetOptionExpiries(
			underlying,
			&from,
			&to,
		)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if expiryMode == "nearest" && len(expiries) > 1 {
			expiries = expiries[:1]
		}
	}

	// HARD SAFETY CAP
	const MAX_EXPIRIES = 6
	if len(expiries) > MAX_EXPIRIES {
		expiries = expiries[:MAX_EXPIRIES]
	}

	out := make(map[string][]models.OptionSnapshot)
	var wg sync.WaitGroup
	var mu sync.Mutex

	for _, exp := range expiries {
		expiry := exp
		wg.Add(1)

		go func() {
			defer wg.Done()

			rows, err := components.GetOptionContractsByPremiumForExpiry(
				underlying,
				optionType,
				targetPremium,
				tolerance,
				expiry,
				from,
				to,
				50,
			)
			if err != nil {
				return
			}

			mu.Lock()
			out[expiry] = rows
			mu.Unlock()
		}()
	}

	wg.Wait()

	json.NewEncoder(w).Encode(map[string]any{
		"data": out,
		"meta": map[string]any{
			"underlying": underlying,
			"expiries":   expiries,
			"from":       from,
			"to":         to,
		},
	})
}
