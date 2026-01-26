package controllers

import (
	"encoding/json"
	"net/http"
	"quant-read-api/components"
	"strconv"
	"strings"
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
