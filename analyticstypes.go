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

type PanelTicketCount struct {
	PanelId    *int   `json:"panel_id"`
	PanelTitle string `json:"panel_title"`
	Count      int    `json:"count"`
}

type LabelTicketCount struct {
	LabelId int    `json:"label_id"`
	Name    string `json:"name"`
	Colour  int32  `json:"colour"`
	Count   int    `json:"count"`
}

type ThreadChannelSplit struct {
	ThreadCount  int `json:"thread_count"`
	ChannelCount int `json:"channel_count"`
}

type FeedbackResponseRate struct {
	ClosedTickets int     `json:"closed_tickets"`
	RatedTickets  int     `json:"rated_tickets"`
	Rate          float64 `json:"rate"`
}

type AutoCloseStats struct {
	AutoClosed   int `json:"auto_closed"`
	ManualClosed int `json:"manual_closed"`
}

type PeakHourEntry struct {
	DayOfWeek int `json:"day_of_week"`
	HourOfDay int `json:"hour_of_day"`
	Count     int `json:"count"`
}

type SourceBreakdown struct {
	Source TicketSource `json:"source"`
	Count int          `json:"count"`
}

type ResponseTimeByHour struct {
	HourOfDay       int            `json:"hour_of_day"`
	AvgResponseTime *time.Duration `json:"avg_response_time"`
}
