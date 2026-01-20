package controllers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"quant-read-api/components"
	"quant-read-api/models"
)

func GetOptionSnapshots(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	q := r.URL.Query()

	underlying := q.Get("underlying")
	optionType := q.Get("option_type")

	moneyness := q.Get("moneyness")
	moneynessMode := q.Get("moneyness_mode")
	expiryMode := q.Get("expiry_mode")

	fromStr := q.Get("from")
	toStr := q.Get("to")

	// defaults
	if moneyness == "" {
		moneyness = "ATM"
	}
	if moneynessMode == "" {
		moneynessMode = "range"
	}
	if expiryMode == "" {
		expiryMode = "nearest"
	}

	moneynessLvl := 0
	if lvlStr := q.Get("moneyness_lvl"); lvlStr != "" {
		v, err := strconv.Atoi(lvlStr)
		if err != nil {
			http.Error(w, "invalid moneyness_lvl", http.StatusBadRequest)
			return
		}
		moneynessLvl = v
	}

	if underlying == "" || optionType == "" || fromStr == "" || toStr == "" {
		http.Error(w, "missing required params", http.StatusBadRequest)
		return
	}

	// ---- FIXED: parse timestamps in Asia/Kolkata ----
	loc, _ := time.LoadLocation("Asia/Kolkata")

	from, err := time.ParseInLocation("2006-01-02T15:04:05", fromStr, loc)
	if err != nil {
		http.Error(w, "invalid from time", http.StatusBadRequest)
		return
	}

	to, err := time.ParseInLocation("2006-01-02T15:04:05", toStr, loc)
	if err != nil {
		http.Error(w, "invalid to time", http.StatusBadRequest)
		return
	}
	// -------------------------------------------------

	rows, err := components.GetOptionSnapshots(
		underlying,
		optionType,
		moneyness,
		moneynessMode,
		moneynessLvl,
		expiryMode,
		from,
		to,
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	resp := models.OptionSnapshotColumnar{
		Ts:           make([]time.Time, 0, len(rows)),
		Underlying:   make([]string, 0, len(rows)),
		Expiry:       make([]time.Time, 0, len(rows)),
		Strike:       make([]uint32, 0, len(rows)),
		OptionType:   make([]string, 0, len(rows)),
		Ltp:          make([]float64, 0, len(rows)),
		SpotPrice:    make([]float64, 0, len(rows)),
		AtmStrike:    make([]uint32, 0, len(rows)),
		Moneyness:    make([]string, 0, len(rows)),
		MoneynessLvl: make([]int16, 0, len(rows)),
		DaysToExpiry: make([]int16, 0, len(rows)),
	}

	for _, r := range rows {
		resp.Ts = append(resp.Ts, r.Ts)
		resp.Underlying = append(resp.Underlying, r.Underlying)
		resp.Expiry = append(resp.Expiry, r.Expiry)
		resp.Strike = append(resp.Strike, r.Strike)
		resp.OptionType = append(resp.OptionType, r.OptionType)
		resp.Ltp = append(resp.Ltp, r.Ltp)
		resp.SpotPrice = append(resp.SpotPrice, r.SpotPrice)
		resp.AtmStrike = append(resp.AtmStrike, r.AtmStrike)
		resp.Moneyness = append(resp.Moneyness, r.Moneyness)
		resp.MoneynessLvl = append(resp.MoneynessLvl, r.MoneynessLvl)
		resp.DaysToExpiry = append(resp.DaysToExpiry, r.DaysToExpiry)
	}

	json.NewEncoder(w).Encode(resp)
}
