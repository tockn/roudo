package main

import (
	"fmt"
	"math"
	"time"

	"github.com/tidwall/buntdb"

	"github.com/gdamore/tcell/v2"

	"github.com/rivo/tview"
)

type TUI struct {
	repo RoudoReportRepository
	db   *buntdb.DB

	app  *tview.Application
	root *tview.Flex
}

func (tui *TUI) Render(yearMonth string, reports roudoReportForView) error {
	tui.app = tview.NewApplication()

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
			form, err := tui.newWorkingForm(r, func(form *tview.Form) func() {
				return func() {
					tui.app.SetFocus(table)
					flex.RemoveItem(form)
				}
			}, func(form *tview.Form) func() {
				return func() {
					tui.app.SetFocus(table)
					flex.RemoveItem(form)
				}
			})
			if err != nil {
				panic(err)
			}
			flex.AddItem(form, 0, 1, true)
			tui.app.SetFocus(form)
		case 2:
			r := reports.Flatten()[row-1]
			form, err := newBreakingForm(r, func(form *tview.Form) func() {
				return func() {
					tui.app.SetFocus(table)
					flex.RemoveItem(form)
				}
			}, func(form *tview.Form) func() {
				return func() {
					tui.app.SetFocus(table)
					flex.RemoveItem(form)
				}
			})
			if err != nil {
				panic(err)
			}
			flex.AddItem(form, 0, 1, true)
			tui.app.SetFocus(form)
		}
	})

	tui.root = tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(tview.NewTextView().SetText(fmt.Sprintf("%sの勤怠", yearMonth)), 1, 1, false).
		AddItem(flex, 0, 1, true)
	return tui.app.SetRoot(tui.root, true).Run()
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

func (tui *TUI) newWorkingForm(r flattenRoudoReportForView, handleSave, handleCancel func(form *tview.Form) func()) (*tview.Form, error) {
	startAt := ""
	if r.Roudo.StartAt != nil {
		startAt = timeToString(r.Roudo.StartAt)
	}
	endAt := ""
	if r.Roudo.EndAt != nil {
		endAt = timeToString(r.Roudo.EndAt)
	}
	form := tview.NewForm().
		AddInputField("出勤時刻(HH:mm)", startAt, 0, nil, func(text string) {
			startAt = text
		}).
		AddInputField("退勤時刻(HH:mm)", endAt, 0, nil, func(text string) {
			endAt = text
		}).
		AddTextView("", "", 0, 0, false, false)
	form.
		AddButton("保存", func() {
			_, err := time.Parse("15:04", startAt)
			if err != nil {
				form.GetFormItem(2).(*tview.TextView).
					SetLabel("エラー").
					SetText("出勤時刻の形式が不正です")
				return
			}
			handleSave(form)()
		}).
		AddButton("キャンセル", func() {
			handleCancel(form)()
		})
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

const emptyTimeStr = "--:--"

func newEmptyTimeCell() *tview.TableCell {
	return tview.NewTableCell(fmt.Sprintf("  %s ~ %s  ", emptyTimeStr, emptyTimeStr)).SetAlign(tview.AlignCenter)
}

func newTimeCell(startAt, endAt *time.Time) *tview.TableCell {
	return tview.NewTableCell(fmt.Sprintf("  %s ~ %s  ", timeToString(startAt), timeToString(endAt))).SetAlign(tview.AlignCenter)
}

func timeToString(t *time.Time) string {
	if t == nil {
		return emptyTimeStr
	}
	return t.Format("15:04")
}
