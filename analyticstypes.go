package database

import "time"

type TripleWindow struct {
	AllTime *time.Duration
	Monthly *time.Duration
	Weekly  *time.Duration
}

type CountOnDate struct {
	Date  time.Time `json:"date"`
	Count uint64    `json:"count"`
}
