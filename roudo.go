package main

import "time"

type Roudo struct {
	StartAt *time.Time `json:"start_at"`
	EndAt   *time.Time `json:"end_at"`
	Breaks  []Break    `json:"breaks"`
}
