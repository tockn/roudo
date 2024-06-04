package main

import (
	"fmt"
	"os"
	"time"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/tidwall/buntdb"
)

type Viewer struct {
	db   *buntdb.DB
	repo RoudoReportRepository
}

func NewViewer(db *buntdb.DB) *Viewer {
	return &Viewer{
		db:   db,
		repo: NewRoudoReportRepository(db),
	}
}

func (v *Viewer) RenderCli(yearMonth string) error {
	reports, err := v.listReports(yearMonth)
	if err != nil {
		return err
	}

	t := TUI{
		repo: v.repo,
		db:   v.db,
	}
	return t.Render(yearMonth, reports)
	//t, err := buildTableWriter(reports)
	//if err != nil {
	//	return err
	//}
	//t.Render()
	return nil
}

type roudoReportForView []struct {
	Date   Date
	Roudos []Roudo
}

type flattenRoudoReportForView struct {
	Date  Date
	Roudo Roudo
	Break *Break
}

func (r roudoReportForView) Flatten() []flattenRoudoReportForView {
	flattenRoudos := make([]flattenRoudoReportForView, 0)
	for _, report := range r {
		if len(report.Roudos) == 0 {
			flattenRoudos = append(flattenRoudos, flattenRoudoReportForView{report.Date, Roudo{}, nil})
		}
		for _, roudo := range report.Roudos {
			if len(roudo.Breaks) == 0 {
				flattenRoudos = append(flattenRoudos, flattenRoudoReportForView{report.Date, roudo, nil})
			}

			for _, b := range roudo.Breaks {
				b := b
				flattenRoudos = append(flattenRoudos, flattenRoudoReportForView{report.Date, roudo, &b})
			}
		}
	}
	return flattenRoudos
}

func getMonthStartEnd(yearMonth string) (time.Time, time.Time, error) {
	monthStart, err := time.Parse("2006-01", yearMonth)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("月の指定が不正です ex: 2024-03")
	}
	monthEnd := monthStart.AddDate(0, 1, 0).AddDate(0, 0, -1)
	return monthStart, monthEnd, nil
}

func (v *Viewer) listReports(yearMonth string) (roudoReportForView, error) {
	monthStart, monthEnd, err := getMonthStartEnd(yearMonth)
	if err != nil {
		return nil, err
	}

	rsByDate := make(map[Date][]Roudo)
	for d := monthStart; !d.After(monthEnd); d = d.AddDate(0, 0, 1) {
		date := Date(d.Format("2006-01-02"))
		rs, err := v.repo.GetRoudoReport(date)
		if err != nil {
			return nil, err
		}
		rsByDate[date] = rs
	}

	var reports roudoReportForView
	for d := monthStart; !d.After(monthEnd); d = d.AddDate(0, 0, 1) {
		date := Date(d.Format("2006-01-02"))
		reports = append(reports, struct {
			Date   Date
			Roudos []Roudo
		}{Date: date, Roudos: rsByDate[date]})
	}

	return reports, nil
}

func buildTableWriter(reports roudoReportForView) (table.Writer, error) {
	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.AppendHeader(table.Row{"日付", "労働開始", "労働終了", "休憩開始", "休憩終了", "休憩時間", "労働時間"})

	totalWorkingTimeSum := time.Duration(0)
	for _, rp := range reports {
		date := rp.Date
		rs := rp.Roudos
		totalWorkingTime := calculateTotalWorkTime(rs)
		totalWorkingTimeSum += totalWorkingTime
		totalWorkingTimeStr := durationToString(totalWorkingTime)
		totalBreakTime := calculateTotalBreakTime(rs)
		totalBreakTimeStr := durationToString(totalBreakTime)

		if len(rs) == 0 {
			t.AppendRow(table.Row{
				date,
				"",
				"",
				"",
				"",
				totalBreakTimeStr,
				totalWorkingTimeStr,
			})
			continue
		}
		for _, r := range rs {
			if len(r.Breaks) == 0 {
				t.AppendRow(table.Row{
					date,
					r.StartAt.Format("15:04"),
					ptrTimeToString(r.EndAt),
					"",
					"",
					totalBreakTimeStr,
					totalWorkingTimeStr,
				})
			} else {
				for _, b := range r.Breaks {
					t.AppendRow(table.Row{
						date,
						r.StartAt.Format("15:04"),
						ptrTimeToString(r.EndAt),
						b.StartAt.Format("15:04"),
						ptrTimeToString(b.EndAt),
						totalBreakTimeStr,
						totalWorkingTimeStr,
					})
				}
			}
		}
	}
	t.AppendFooter(table.Row{"", "", "", "", "総労働時間", durationToString(totalWorkingTimeSum)})
	t.SetColumnConfigs([]table.ColumnConfig{
		{Number: 0, AutoMerge: true},
		{Number: 1, AutoMerge: true},
		{Number: 2, AutoMerge: true},
		{Number: 3, AutoMerge: true},
		{Number: 6, AutoMerge: true},
		{Number: 7, AutoMerge: true},
	})
	t.SetStyle(table.StyleRounded)
	return t, nil
}

func calculateTotalWorkTime(rs []Roudo) time.Duration {
	var total time.Duration
	for _, r := range rs {
		if r.EndAt == nil {
			continue
		}
		total += r.EndAt.Sub(*r.StartAt)
		for _, b := range r.Breaks {
			if b.EndAt == nil {
				continue
			}
			total -= b.EndAt.Sub(b.StartAt)
		}
	}
	return total
}

func calculateTotalBreakTime(rs []Roudo) time.Duration {
	var total time.Duration
	for _, r := range rs {
		for _, b := range r.Breaks {
			if b.EndAt == nil {
				continue
			}
			total += b.EndAt.Sub(b.StartAt)
		}
	}
	return total
}

func durationToString(d time.Duration) string {
	fl := 0.0
	if d.Seconds() > 0 {
		fl = 1
	}
	return fmt.Sprintf("%02d:%02d", int(d.Hours()), int(d.Minutes()+fl)%60)
}

func ptrTimeToString(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.Format("15:04")
}

func timePtr(t time.Time) *time.Time {
	return &t
}
