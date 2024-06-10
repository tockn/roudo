package roudo

import "time"

type Roudo struct {
	StartAt *time.Time `json:"start_at"`
	EndAt   *time.Time `json:"end_at"`
	Breaks  []Break    `json:"breaks"`
}

func (r *Roudo) TotalWorkingTime() time.Duration {
	if r.EndAt == nil {
		return 0
	}
	total := r.EndAt.Sub(*r.StartAt)
	for _, b := range r.Breaks {
		if b.EndAt == nil {
			continue
		}
		total -= b.EndAt.Sub(b.StartAt)
	}
	return total
}

func (r *Roudo) TotalBreakTime() time.Duration {
	var total time.Duration
	for _, b := range r.Breaks {
		if b.EndAt == nil {
			continue
		}
		total += b.EndAt.Sub(b.StartAt)
	}
	return total
}
