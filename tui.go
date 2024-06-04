package main

import (
	"fmt"
	"time"

	"github.com/gdamore/tcell/v2"

	"github.com/rivo/tview"
)

func renderTui(yearMonth string, reports roudoReportForView) error {
	app := tview.NewApplication()

	grid := tview.NewGrid().
		SetBorders(true).
		SetColumns(12, 17, 17)

	grid.AddItem(newPrimitive(fmt.Sprintf("%sの勤怠", yearMonth)), 0, 0, 1, 3, 0, 0, false)
	grid.AddItem(newPrimitive("日付"), 1, 0, 1, 1, 0, 0, false)
	grid.AddItem(newPrimitive("出退勤"), 1, 1, 1, 1, 0, 0, false)
	grid.AddItem(newPrimitive("休憩"), 1, 2, 1, 1, 0, 0, false)

	rowOffset := 2
	rowSizes := []int{1}
	for i, report := range reports {
		w, wrow := buildWorkingTime(report.Roudos)
		b, brow := buildBreakingTimePrimitive(report.Roudos)

		date, err := dateToPrimitive(report.Date)
		if err != nil {
			return err
		}
		grid.
			AddItem(date, i+rowOffset, 0, 1, 1, 0, 0, false).
			AddItem(w, i+rowOffset, 1, 1, 1, 0, 0, false).
			AddItem(b, i+rowOffset, 2, 1, 1, 0, 0, false)

		rowSize := 1
		if wrow > rowSize {
			rowSize = wrow
		}
		if brow > rowSize {
			rowSize = brow
		}
		rowSizes = append(rowSizes, rowSize)
	}
	grid.SetRows(rowSizes...)
	return app.SetRoot(grid, true).SetFocus(grid).Run()
}

var week = []string{"日", "月", "火", "水", "木", "金", "土"}

func dateToPrimitive(d Date) (tview.Primitive, error) {
	t, err := time.Parse("2006-01-02", string(d))
	if err != nil {
		return nil, err
	}
	color := tcell.ColorWhite
	switch t.Weekday() {
	case time.Saturday:
		color = tcell.ColorBlue
	case time.Sunday:
		color = tcell.ColorRed
	}

	s := fmt.Sprintf("%s (%s)", t.Format("01/02"), week[t.Weekday()])
	return tview.NewTextView().
		SetTextAlign(tview.AlignCenter).
		SetText(s).SetTextColor(color), nil
}

func newPrimitive(text string) tview.Primitive {
	return tview.NewTextView().
		SetTextAlign(tview.AlignCenter).
		SetText(text)
}

const emptyTimeStr = "--:--"

func buildWorkingTime(rs []Roudo) (tview.Primitive, int) {
	if len(rs) == 0 {
		return newPrimitive(fmt.Sprintf("%s ~ %s", emptyTimeStr, emptyTimeStr)), 1
	}

	s := ""
	for _, r := range rs {
		startAt := emptyTimeStr
		if r.StartAt != nil {
			startAt = timeToString(*r.StartAt)
		}
		endAt := emptyTimeStr
		if r.EndAt != nil {
			endAt = timeToString(*r.EndAt)
		}
		s += fmt.Sprintf("%s ~ %s\n", startAt, endAt)
	}

	return newPrimitive(s), len(rs)
}

func buildBreakingTimePrimitive(rs []Roudo) (tview.Primitive, int) {
	s := ""
	row := 0
	for _, r := range rs {
		for _, b := range r.Breaks {
			row++
			startAt := timeToString(b.StartAt)
			endAt := emptyTimeStr
			if b.EndAt != nil {
				endAt = timeToString(*b.EndAt)
			}
			s += fmt.Sprintf("%s ~ %s\n", startAt, endAt)
		}
	}
	if row == 0 {
		return newPrimitive(fmt.Sprintf("%s ~ %s", emptyTimeStr, emptyTimeStr)), 1
	}

	return newPrimitive(s), row
}

func timeToString(t time.Time) string {
	return t.Format("15:04")
}
