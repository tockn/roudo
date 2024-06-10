package view

import (
	"fmt"
	"log/slog"
	"math"
	"roudo/roudo"
	"time"

	"github.com/gdamore/tcell/v2"

	"github.com/rivo/tview"
)

func NewTUI(roudoReporter roudo.RoudoReporter, repo ViewRepository, logger *slog.Logger) Viewer {
	return &tui{
		roudoReporter: roudoReporter,
		repo:          repo,
		logger:        logger,
	}
}

type tui struct {
	roudoReporter roudo.RoudoReporter
	repo          ViewRepository

	logger *slog.Logger

	app  *tview.Application
	root *tview.Flex
}

func (t *tui) Do(yearMonth string) error {
	reports, err := t.repo.ListReports(yearMonth)
	if err != nil {
		return err
	}

	if t.app != nil {
		t.app.Stop()
	}

	t.app = tview.NewApplication()

	table, err := newRoudoReportTable(reports)
	if err != nil {
		return err
	}
	flex := tview.NewFlex().
		SetDirection(tview.FlexColumn).
		AddItem(table, 0, 1, true)

	rowOffset := 1
	table.Select(0, 0).SetFixed(1, 1).SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyEnter {
			table.SetSelectable(true, true)
		}
	}).SetSelectedFunc(func(row int, column int) {
		switch column {
		case 1:
			r := reports.Flatten()[row-rowOffset]
			form, err := t.newWorkingForm(r, func(form *tview.Form, startAt, endAt *time.Time) func() {
				return func() {
					defer func() {
						t.Do(yearMonth)
					}()

					currentReport := reports.FindByDate(r.Date)
					if startAt == nil {
						if len(currentReport) == 0 {
							return
						}
						currentReport = append(currentReport[:r.RoudoIndex], currentReport[r.RoudoIndex+1:]...)
						if err := t.roudoReporter.SaveRoudoReport(r.Date, currentReport); err != nil {
							t.logger.Error("failed to save roudo report: ", err)
						}
					} else {
						index := 0
						if len(currentReport) <= r.RoudoIndex {
							index = len(currentReport)
							currentReport = append(currentReport, roudo.Roudo{})
						}
						currentReport[index].StartAt = startAt
						currentReport[index].EndAt = endAt
						if err := t.roudoReporter.SaveRoudoReport(r.Date, currentReport); err != nil {
							t.logger.Error("failed to save roudo report: ", err)
						}
					}
				}
			}, func(form *tview.Form) func() {
				return func() {
					t.app.SetFocus(table)
					flex.RemoveItem(form)
				}
			})
			if err != nil {
				panic(err)
			}
			flex.AddItem(form, 0, 1, true)
			t.app.SetFocus(form)
		case 2:
			r := reports.Flatten()[row-rowOffset]
			form, err := newBreakingForm(r, func(form *tview.Form, startAt, endAt *time.Time) func() {
				return func() {
					defer func() {
						t.Do(yearMonth)
					}()

					currentReport := reports.FindByDate(r.Date)
					if startAt == nil {
						if len(currentReport[r.RoudoIndex].Breaks) == 0 {
							return
						}
						currentReport[r.RoudoIndex].Breaks = append(currentReport[r.RoudoIndex].Breaks[:r.BreakIndex], currentReport[r.RoudoIndex].Breaks[r.BreakIndex+1:]...)
						if err := t.roudoReporter.SaveRoudoReport(r.Date, currentReport); err != nil {
							t.logger.Error("failed to save roudo report: ", err)
						}
					} else {
						index := 0
						if len(currentReport[r.RoudoIndex].Breaks) <= r.BreakIndex {
							index = len(currentReport[r.RoudoIndex].Breaks)
							currentReport[r.RoudoIndex].Breaks = append(currentReport[r.RoudoIndex].Breaks, roudo.Break{})
						}
						currentReport[r.RoudoIndex].Breaks[index].StartAt = *startAt
						currentReport[r.RoudoIndex].Breaks[index].EndAt = endAt
						if err := t.roudoReporter.SaveRoudoReport(r.Date, currentReport); err != nil {
							t.logger.Error("failed to save roudo report: ", err)
						}
					}
				}
			}, func(form *tview.Form) func() {
				return func() {
					t.app.SetFocus(table)
					flex.RemoveItem(form)
				}
			})
			if err != nil {
				panic(err)
			}
			flex.AddItem(form, 0, 1, true)
			t.app.SetFocus(form)
		}
	})

	t.root = tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(tview.NewTextView().SetText(fmt.Sprintf("%sの勤怠", yearMonth)), 1, 1, false).
		AddItem(flex, 0, 1, true)
	return t.app.SetRoot(t.root, true).Run()
}

func newRoudoReportTable(reports roudoReportForView) (*tview.Table, error) {
	table := tview.NewTable().SetBorders(true)

	table.SetCell(0, 0, tview.NewTableCell("日付").SetAlign(tview.AlignCenter).SetSelectable(false))
	table.SetCell(0, 1, tview.NewTableCell("出退勤").SetAlign(tview.AlignCenter).SetSelectable(false))
	table.SetCell(0, 2, tview.NewTableCell("休憩").SetAlign(tview.AlignCenter).SetSelectable(false))
	table.SetCell(0, 3, tview.NewTableCell("休憩時間").SetAlign(tview.AlignCenter).SetSelectable(false))
	table.SetCell(0, 4, tview.NewTableCell("労働時間").SetAlign(tview.AlignCenter).SetSelectable(false))

	offset := 1
	totalWorkingTime := time.Duration(0)
	for repoIdx, report := range reports {
		date, err := dateToCell(report.Date)
		if err != nil {
			return nil, err
		}
		table.SetCell(repoIdx+offset, 0, date.SetSelectable(false))
		table.SetCell(repoIdx+offset, 1, newEmptyTimeCell())
		table.SetCell(repoIdx+offset, 2, newEmptyTimeCell())

		maxBreakCount := 1
		workingTime := time.Duration(0)
		breakingTime := time.Duration(0)
		for roudoIdx, r := range report.Roudos {
			workingTime += r.TotalWorkingTime()
			breakingTime += r.TotalBreakTime()

			table.SetCell(repoIdx+roudoIdx+offset, 1, newTimeCell(r.StartAt, r.EndAt).SetAlign(tview.AlignCenter))

			for breakIdx, b := range r.Breaks {
				table.SetCell(repoIdx+roudoIdx+breakIdx+offset, 2, newTimeCell(&b.StartAt, b.EndAt).SetAlign(tview.AlignCenter))
			}
			maxBreakCount = int(math.Max(float64(maxBreakCount), float64(len(r.Breaks))))
		}
		offset += int(math.Max(math.Max(0, float64(len(report.Roudos)-1)), float64(maxBreakCount-1)))

		table.SetCell(repoIdx+offset, 3, tview.NewTableCell(durationToString(breakingTime)).SetAlign(tview.AlignCenter).SetSelectable(false))
		table.SetCell(repoIdx+offset, 4, tview.NewTableCell(durationToString(workingTime)).SetAlign(tview.AlignCenter).SetSelectable(false))
		totalWorkingTime += workingTime
	}

	table.SetCell(len(reports)+offset, 3, tview.NewTableCell("総労働時間").SetAlign(tview.AlignCenter).SetSelectable(false))
	table.SetCell(len(reports)+offset, 4, tview.NewTableCell(durationToString(totalWorkingTime)).SetAlign(tview.AlignCenter).SetSelectable(false))
	return table, nil
}

func (t *tui) newWorkingForm(r flattenRoudoReportForView, handleSave func(form *tview.Form, startAt, endAt *time.Time) func(), handleCancel func(form *tview.Form) func()) (*tview.Form, error) {
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
			var s, e *time.Time
			if startAt != "" {
				ps, err := time.Parse("15:04", startAt)
				if err != nil {
					form.GetFormItem(2).(*tview.TextView).
						SetLabel("エラー").
						SetText("出勤時刻の形式が不正です")
					return
				}
				s = &ps
			}
			if endAt != "" {
				pe, err := time.Parse("15:04", endAt)
				if err != nil {
					form.GetFormItem(2).(*tview.TextView).
						SetLabel("エラー").
						SetText("退勤時刻の形式が不正です")
					return
				}
				e = &pe
			}
			handleSave(form, s, e)()
		}).
		AddButton("キャンセル", func() {
			handleCancel(form)()
		})
	form.SetBorder(true).SetTitle("勤怠入力（出退勤）").SetTitleAlign(tview.AlignLeft)
	return form, nil
}

func newBreakingForm(r flattenRoudoReportForView, handleSave func(form *tview.Form, startAt, endAt *time.Time) func(), handleCancel func(form *tview.Form) func()) (*tview.Form, error) {
	startAt := ""
	if r.Break != nil {
		startAt = timeToString(&r.Break.StartAt)
	}
	endAt := ""
	if r.Break != nil && r.Break.EndAt != nil {
		endAt = timeToString(r.Break.EndAt)
	}
	form := tview.NewForm().
		AddInputField("休憩開始時刻(HH:mm)", startAt, 0, nil, func(text string) {
			startAt = text
		}).
		AddInputField("休憩終了時刻(HH:mm)", endAt, 0, nil, func(text string) {
			endAt = text
		})
	form.AddButton("保存", func() {
		var s, e *time.Time
		if startAt != "" {
			ps, err := time.Parse("15:04", startAt)
			if err != nil {
				form.GetFormItem(2).(*tview.TextView).
					SetLabel("エラー").
					SetText("休憩開始時刻の形式が不正です")
				return
			}
			s = &ps
		}
		if endAt != "" {
			pe, err := time.Parse("15:04", endAt)
			if err != nil {
				form.GetFormItem(2).(*tview.TextView).
					SetLabel("エラー").
					SetText("休憩終了時刻の形式が不正です")
				return
			}
			e = &pe
		}
		handleSave(form, s, e)()
	}).
		AddButton("キャンセル", handleCancel(form))
	form.SetBorder(true).SetTitle("勤怠入力（休憩）").SetTitleAlign(tview.AlignLeft)
	return form, nil
}

var week = []string{"日", "月", "火", "水", "木", "金", "土"}

func dateToCell(d roudo.Date) (*tview.TableCell, error) {
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
