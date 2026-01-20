package models

import "time"

type ColumnarOHLC struct {
	Ts    []time.Time `json:"ts"`
	Open  []float64   `json:"open"`
	High  []float64   `json:"high"`
	Low   []float64   `json:"low"`
	Close []float64   `json:"close"`
}
