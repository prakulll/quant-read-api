package models

import "time"

type FuturesDataRow struct {
	Ts           time.Time `json:"ts"`
	FuturesPrice float64   `json:"futures_price"`
	Underlying   string    `json:"underlying"`
	Series       string    `json:"series"`
}
