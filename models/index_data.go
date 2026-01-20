package models

import "time"

type IndexDataRow struct {
	Ts         time.Time `json:"ts"`
	SpotPrice  float64   `json:"spot_price"`
	Underlying string    `json:"underlying"`
}

type IndexDataColumnar struct {
	Ts         []time.Time `json:"ts"`
	SpotPrice  []float64   `json:"spot_price"`
	Underlying string      `json:"underlying"`
}
