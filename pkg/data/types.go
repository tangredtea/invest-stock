package data

import "time"

type KLine struct {
	Date   time.Time `json:"date"`
	Open   float64   `json:"open"`
	High   float64   `json:"high"`
	Low    float64   `json:"low"`
	Close  float64   `json:"close"`
	Volume float64   `json:"volume"`
}

// Quote holds real-time quote data.
type Quote struct {
	Price    float64 `json:"price"`
	Open     float64 `json:"open"`
	High     float64 `json:"high"`
	Low      float64 `json:"low"`
	PreClose float64 `json:"preClose"`
}
