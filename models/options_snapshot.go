package models

import (
	"encoding/json"
	"time"
)

type OptionSnapshot struct {
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

func (o OptionSnapshot) MarshalJSON() ([]byte, error) {
	type Alias OptionSnapshot

	return json.Marshal(&struct {
		*Alias
		Expiry string `json:"Expiry"`
	}{
		Alias:  (*Alias)(&o),
		Expiry: o.Expiry.In(time.FixedZone("IST", 5*60*60+30*60)).Format("2006-01-02"), // ← no UTC conversion
	})
}
