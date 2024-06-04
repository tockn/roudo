package main

import (
	"fmt"
	"math"
	"time"

	"github.com/gdamore/tcell/v2"

	"github.com/rivo/tview"
)

func renderTui(yearMonth string, reports roudoReportForView) error {
	app := tview.NewApplication()

	table, err := newRoudoReportTable(reports)
	if err != nil {
		return err
	}
	flex := tview.NewFlex().
		SetDirection(tview.FlexColumn).
		AddItem(table, 0, 1, true)

	table.Select(0, 0).SetFixed(1, 1).SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyEnter {
			table.SetSelectable(true, true)
		}
	}).SetSelectedFunc(func(row int, column int) {
		switch column {
		case 1:
			r := reports.Flatten()[row-1]
			form, err := newWorkingForm(r, func(form *tview.Form) func() {
				return func() {
					app.SetFocus(table)
					flex.RemoveItem(form)
				}
			}, func(form *tview.Form) func() {
				return func() {
					app.SetFocus(table)
					flex.RemoveItem(form)
				}
			})
			if err != nil {
				panic(err)
			}
			flex.AddItem(form, 0, 1, true)
			app.SetFocus(form)
		case 2:
			r := reports.Flatten()[row-1]
			form, err := newBreakingForm(r, func(form *tview.Form) func() {
				return func() {
					app.SetFocus(table)
					flex.RemoveItem(form)
				}
			}, func(form *tview.Form) func() {
				return func() {
					app.SetFocus(table)
					flex.RemoveItem(form)
				}
			})
			if err != nil {
				panic(err)
			}
			flex.AddItem(form, 0, 1, true)
			app.SetFocus(form)
		}
	})

	return app.SetRoot(flex, true).Run()
}

func newRoudoReportTable(reports roudoReportForView) (*tview.Table, error) {
	table := tview.NewTable().SetBorders(true)

	table.SetCell(0, 0, tview.NewTableCell("日付").SetAlign(tview.AlignCenter).SetSelectable(false))
	table.SetCell(0, 1, tview.NewTableCell("出退勤").SetAlign(tview.AlignCenter).SetSelectable(false))
	table.SetCell(0, 2, tview.NewTableCell("休憩").SetAlign(tview.AlignCenter).SetSelectable(false))

	offset := 1
	for repoIdx, report := range reports {
		date, err := dateToCell(report.Date)
		if err != nil {
			return nil, err
		}
		table.SetCell(repoIdx+offset, 0, date.SetSelectable(false))
		table.SetCell(repoIdx+offset, 1, newEmptyTimeCell())
		table.SetCell(repoIdx+offset, 2, newEmptyTimeCell())

		maxBreakCount := 1
		for roudoIdx, r := range report.Roudos {
			table.SetCell(repoIdx+roudoIdx+offset, 1, newTimeCell(r.StartAt, r.EndAt).SetAlign(tview.AlignCenter))

			for breakIdx, b := range r.Breaks {
				table.SetCell(repoIdx+roudoIdx+breakIdx+offset, 2, newTimeCell(&b.StartAt, b.EndAt).SetAlign(tview.AlignCenter))
			}
			maxBreakCount = int(math.Max(float64(maxBreakCount), float64(len(r.Breaks))))
		}
		offset += int(math.Max(math.Max(0, float64(len(report.Roudos)-1)), float64(maxBreakCount-1)))
	}
	return table, nil
}

func newWorkingForm(r flattenRoudoReportForView, handleSave, handleCancel func(form *tview.Form) func()) (*tview.Form, error) {
	startAtDefault := ""
	if r.Roudo.StartAt != nil {
		startAtDefault = timeToString(r.Roudo.StartAt)
	}
	endAtDefault := ""
	if r.Roudo.EndAt != nil {
		endAtDefault = timeToString(r.Roudo.EndAt)
	}
	form := tview.NewForm().
		AddInputField("出勤時刻(HH:mm)", startAtDefault, 0, nil, nil).
		AddInputField("退勤時刻(HH:mm)", endAtDefault, 0, nil, nil)
	form.AddButton("保存", handleSave(form)).
		AddButton("キャンセル", handleCancel(form))
	form.SetBorder(true).SetTitle("勤怠入力（出退勤）").SetTitleAlign(tview.AlignLeft)
	return form, nil
}

func newBreakingForm(r flattenRoudoReportForView, handleSave, handleCancel func(form *tview.Form) func()) (*tview.Form, error) {
	startAtDefault := ""
	if r.Break != nil {
		startAtDefault = timeToString(&r.Break.StartAt)
	}
	endAtDefault := ""
	if r.Break != nil && r.Break.EndAt != nil {
		endAtDefault = timeToString(r.Break.EndAt)
	}
	form := tview.NewForm().
		AddInputField("休憩開始時刻(HH:mm)", startAtDefault, 0, nil, nil).
		AddInputField("休憩終了時刻(HH:mm)", endAtDefault, 0, nil, nil)
	form.AddButton("保存", handleSave(form)).
		AddButton("キャンセル", handleCancel(form))
	form.SetBorder(true).SetTitle("勤怠入力（休憩）").SetTitleAlign(tview.AlignLeft)
	return form, nil
}

var week = []string{"日", "月", "火", "水", "木", "金", "土"}

func dateToCell(d Date) (*tview.TableCell, error) {
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

	s := fmt.Sprintf(" %s (%s) ", t.Format("01/02"), week[t.Weekday()])
	return tview.NewTableCell(s).SetTextColor(color).SetAlign(tview.AlignCenter), nil
}

func newPrimitive(text string) *tview.TextView {
	return tview.NewTextView().
		SetTextAlign(tview.AlignCenter).
		SetText(text)
}

const emptyTimeStr = "--:--"

func newEmptyTimeCell() *tview.TableCell {
	return tview.NewTableCell(fmt.Sprintf("  %s ~ %s  ", emptyTimeStr, emptyTimeStr)).SetAlign(tview.AlignCenter)
}

func newTimeCell(startAt, endAt *time.Time) *tview.TableCell {
	return tview.NewTableCell(fmt.Sprintf("  %s ~ %s  ", timeToString(startAt), timeToString(endAt))).SetAlign(tview.AlignCenter)
}

//func buildWorkingTime(rs []Roudo) (tview.Primitive, int) {
//	if len(rs) == 0 {
//		return newPrimitive(fmt.Sprintf("%s ~ %s", emptyTimeStr, emptyTimeStr)), 1
//	}
//
//	s := ""
//	for _, r := range rs {
//		startAt := emptyTimeStr
//		if r.StartAt != nil {
//			startAt = timeToString(*r.StartAt)
//		}
//		endAt := emptyTimeStr
//		if r.EndAt != nil {
//			endAt = timeToString(*r.EndAt)
//		}
//		s += fmt.Sprintf("%s ~ %s\n", startAt, endAt)
//	}
//
//	return newPrimitive(s), len(rs)
//}
//
//func buildBreakingTimePrimitive(rs []Roudo) (tview.Primitive, int) {
//	s := ""
//	row := 0
//	for _, r := range rs {
//		for _, b := range r.Breaks {
//			row++
//			startAt := timeToString(b.StartAt)
//			endAt := emptyTimeStr
//			if b.EndAt != nil {
//				endAt = timeToString(*b.EndAt)
//			}
//			s += fmt.Sprintf("%s ~ %s\n", startAt, endAt)
//		}
//	}
//	if row == 0 {
//		return newPrimitive(fmt.Sprintf("%s ~ %s", emptyTimeStr, emptyTimeStr)), 1
//	}
//
//	return newPrimitive(s), row
//}

func timeToString(t *time.Time) string {
	if t == nil {
		return emptyTimeStr
	}
	return t.Format("15:04")
}
