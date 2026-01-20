package controllers

import (
	"encoding/json"
	"net/http"
	"quant-read-api/components"
	"quant-read-api/models"
	"strconv"
	"time"
)

var ist, _ = time.LoadLocation("Asia/Kolkata")

func forceUTC(data interface{}) {
	switch v := data.(type) {

	case []models.OptionSnapshot:
		for i := range v {
			v[i].Ts = v[i].Ts.UTC()
			v[i].Expiry = v[i].Expiry.UTC()
		}

	case *[]models.OptionSnapshot:
		for i := range *v {
			(*v)[i].Ts = (*v)[i].Ts.UTC()
			(*v)[i].Expiry = (*v)[i].Expiry.UTC()
		}

	case map[string]interface{}:
		for _, val := range v {
			switch arr := val.(type) {
			case []models.OptionSnapshot:
				for i := range arr {
					arr[i].Ts = arr[i].Ts.UTC()
					arr[i].Expiry = arr[i].Expiry.UTC()
				}
			case *[]models.OptionSnapshot:
				for i := range *arr {
					(*arr)[i].Ts = (*arr)[i].Ts.UTC()
					(*arr)[i].Expiry = (*arr)[i].Expiry.UTC()
				}
			}
		}
	}
}

func GetOptionContractsByPremium(w http.ResponseWriter, r *http.Request) {

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
		http.Error(w, "from and to timestamps are required (RFC3339)", http.StatusBadRequest)
		return
	}

	from, err := time.Parse(time.RFC3339, fromStr)
	if err != nil {
		http.Error(w, "invalid from timestamp", http.StatusBadRequest)
		return
	}

	to, err := time.Parse(time.RFC3339, toStr)
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

	forceUTC(response)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
