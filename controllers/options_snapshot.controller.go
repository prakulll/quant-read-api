package controllers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"sync"
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

/*
========================
V2 — CONCURRENT SNAPSHOT
========================
*/
func GetOptionSnapshotsV2(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	q := r.URL.Query()

	underlying := q.Get("underlying")
	optionType := q.Get("option_type")
	expiryMode := q.Get("expiry_mode")
	expiryParam := q.Get("expiry")

	moneyness := q.Get("moneyness")
	moneynessMode := q.Get("moneyness_mode")

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

	// ------------------------------------
	// EXPIRY RESOLUTION (STRING BASED)
	// ------------------------------------
	var expiries []string

	if expiryParam != "" {
		// explicit expiry list
		expiries = strings.Split(expiryParam, ",")
		for i := range expiries {
			expiries[i] = strings.TrimSpace(expiries[i])
		}
	} else {
		// component decides expiries
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

	// HARD CAP — SAFETY
	const MAX_EXPIRIES = 6
	if len(expiries) > MAX_EXPIRIES {
		expiries = expiries[:MAX_EXPIRIES]
	}

	// ------------------------------------
	// CONCURRENT FETCH
	// ------------------------------------
	out := make(map[string]models.OptionSnapshotColumnar)
	var mu sync.Mutex
	var wg sync.WaitGroup

	for _, expiry := range expiries {
		exp := expiry
		wg.Add(1)

		go func() {
			defer wg.Done()

			rows, err := components.GetOptionSnapshots(
				underlying,
				optionType,
				moneyness,
				moneynessMode,
				moneynessLvl,
				exp,
				from,
				to,
			)
			if err != nil {
				return
			}

			snap := models.OptionSnapshotColumnar{
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
				snap.Ts = append(snap.Ts, r.Ts)
				snap.Underlying = append(snap.Underlying, r.Underlying)
				snap.Expiry = append(snap.Expiry, r.Expiry)
				snap.Strike = append(snap.Strike, r.Strike)
				snap.OptionType = append(snap.OptionType, r.OptionType)
				snap.Ltp = append(snap.Ltp, r.Ltp)
				snap.SpotPrice = append(snap.SpotPrice, r.SpotPrice)
				snap.AtmStrike = append(snap.AtmStrike, r.AtmStrike)
				snap.Moneyness = append(snap.Moneyness, r.Moneyness)
				snap.MoneynessLvl = append(snap.MoneynessLvl, r.MoneynessLvl)
				snap.DaysToExpiry = append(snap.DaysToExpiry, r.DaysToExpiry)
			}

			mu.Lock()
			out[exp] = snap
			mu.Unlock()
		}()
	}

	wg.Wait()

	json.NewEncoder(w).Encode(map[string]any{
		"data": out,
		"meta": map[string]any{
			"underlying": underlying,
			"from":       from,
			"to":         to,
		},
	})
}
