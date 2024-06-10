package roudo

import "time"

type RoudoTime struct {
	t             *time.Time
	shiftDuration time.Duration
}

func NewRoudoTime(t time.Time, shiftDuration time.Duration) RoudoTime {
	return RoudoTime{t: &t, shiftDuration: shiftDuration}
}

func (rt RoudoTime) Time() *time.Time {
	return rt.t
}

func (rt RoudoTime) ShiftedDate() Date {
	return Date(rt.t.Add(-rt.shiftDuration).Format("2006-01-02"))
}

func (rt RoudoTime) ShiftedMidnight() time.Time {
	return rt.t.Add(24 * time.Hour).Truncate(24 * time.Hour).Add(-rt.shiftDuration)
}

func (rt RoudoTime) IsOvernight(before RoudoTime) bool {
	return rt.ShiftedDate() != before.ShiftedDate()
}

type Date string
