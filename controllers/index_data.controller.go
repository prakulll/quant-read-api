package controllers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"quant-read-api/components"
	"quant-read-api/models"
)

func GetIndexData(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	loc, _ := time.LoadLocation("Asia/Kolkata")

	q := r.URL.Query()

	underlying := q.Get("underlying")
	fromStr := q.Get("from")
	toStr := q.Get("to")
	tfStr := q.Get("tf")
	offsetStr := q.Get("offset")

	if underlying == "" || fromStr == "" || toStr == "" {
		http.Error(w, "missing query params", http.StatusBadRequest)
		return
	}

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

	var tfSeconds *int64
	if tfStr != "" {
		val, err := parseTF(tfStr)
		if err != nil {
			http.Error(w, "invalid tf", http.StatusBadRequest)
			return
		}
		tfSeconds = &val
	}

	var offsetSeconds int64
	if offsetStr != "" {
		offsetSeconds, err = strconv.ParseInt(offsetStr, 10, 64)
		if err != nil {
			http.Error(w, "invalid offset", http.StatusBadRequest)
			return
		}
	}

	data, err := components.GetIndexData(
		underlying,
		from,
		to,
		tfSeconds,
		offsetSeconds,
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var firstTs, lastTs string

	switch v := data.(type) {
	case models.ColumnarOHLC:
		if len(v.Ts) > 0 {
			firstTs = v.Ts[0].Format(time.RFC3339)
			lastTs = v.Ts[len(v.Ts)-1].Format(time.RFC3339)
		}
	case []models.IndexDataRow:
		if len(v) > 0 {
			firstTs = v[0].Ts.Format(time.RFC3339)
			lastTs = v[len(v)-1].Ts.Format(time.RFC3339)
		}
	}

	resp := models.Response[any]{
		Data: data,
		Meta: models.Meta{
			Underlying: underlying,
			From:       from.Format(time.RFC3339),
			To:         to.Format(time.RFC3339),
			Tf:         tfStr,
			Offset:     offsetSeconds,
			FirstTs:    firstTs,
			LastTs:     lastTs,
		},
	}

	json.NewEncoder(w).Encode(resp)
}
