package models

type Response[T any] struct {
	Data T    `json:"data"`
	Meta Meta `json:"meta"`
}

type Meta struct {
	Underlying string `json:"underlying,omitempty"`
	Series     string `json:"series,omitempty"`
	Expiry     string `json:"expiry,omitempty"`
	Strike     uint32 `json:"strike,omitempty"`
	OptionType string `json:"option_type,omitempty"`

	From   string `json:"from,omitempty"`
	To     string `json:"to,omitempty"`
	Tf     string `json:"tf,omitempty"`
	Offset int64  `json:"offset,omitempty"`

	FirstTs string `json:"first_ts,omitempty"`
	LastTs  string `json:"last_ts,omitempty"`
}
