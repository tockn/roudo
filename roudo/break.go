package roudo

import "time"

type Break struct {
	StartAt time.Time  `json:"start_at"`
	EndAt   *time.Time `json:"end_at"`
}
