package models

import "time"

type OptionContractRow struct {
	Ts           time.Time
	Underlying   string
	Expiry       time.Time
	Strike       uint32
	OptionType   string
	Ltp          float64
	SpotPrice    float64
	AtmStrike    uint32
	Moneyness    string
	MoneynessLvl int16
	DaysToExpiry int16
}

type OptionContractColumnar struct {
	Ts           []time.Time `json:"ts"`
	Underlying   []string    `json:"underlying"`
	Expiry       []time.Time `json:"expiry"`
	Strike       []uint32    `json:"strike"`
	OptionType   []string    `json:"option_type"`
	Ltp          []float64   `json:"ltp"`
	SpotPrice    []float64   `json:"spot"`
	AtmStrike    []uint32    `json:"atm_strike"`
	Moneyness    []string    `json:"moneyness"`
	MoneynessLvl []int16     `json:"moneyness_lvl"`
	DaysToExpiry []int16     `json:"days_to_expiry"`
}
