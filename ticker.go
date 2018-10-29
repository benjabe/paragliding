package main

type Ticker struct {
	TLatest    int64     `json:"t_latest"`
	TStart     int64     `json:"t_start"`
	TStop      int64     `json:"t_stop"`
	Tracks     [5]string `json:"tracks"` // TODO: make len editable somewhere convenient
	Processing int       `json:"processing"`
}
