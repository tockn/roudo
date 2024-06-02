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
	rsByDate, err := v.listReports(yearMonth)
	if err != nil {
		return err
	}
	t, err := buildTableWriter(rsByDate)
	if err != nil {
		return err
	}
	t.Render()
	return nil
}

func (v *Viewer) listReports(yearMonth string) (map[Date][]Roudo, error) {
	monthStart, err := time.Parse("2006-01", yearMonth)
	if err != nil {
		return nil, fmt.Errorf("月の指定が不正です ex: 2024-03")
	}
	monthEnd := monthStart.AddDate(0, 1, 0).AddDate(0, 0, -1)

	rsByDate := make(map[Date][]Roudo, 0)
	for d := monthStart; d.Before(monthEnd); d = d.AddDate(0, 0, 1) {
		date := Date(d.Format("2006-01-02"))
		rs, err := v.repo.GetRoudoReport(date)
		if err != nil {
			return nil, err
		}
		rsByDate[date] = rs
	}
	return rsByDate, nil
}

func buildTableWriter(rsByDate map[Date][]Roudo) (table.Writer, error) {
	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.AppendHeader(table.Row{"日付", "労働開始", "労働終了", "休憩開始", "休憩終了", "総労働時間"})

	totalSum := time.Duration(0)
	for date, rs := range rsByDate {
		total := calculateTotalWorkTime(rs)
		totalSum += total
		if len(rs) == 0 {
			t.AppendRow(table.Row{
				date,
				"",
				"",
				"",
				"",
				total,
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
					total,
				})
			} else {
				for _, b := range r.Breaks {
					t.AppendRow(table.Row{
						date,
						r.StartAt.Format("15:04"),
						ptrTimeToString(r.EndAt),
						b.StartAt.Format("15:04"),
						ptrTimeToString(b.EndAt),
						total,
					})
				}
			}
		}
	}
	t.AppendFooter(table.Row{"", "", "", "", "総労働時間", totalSum})
	t.SetColumnConfigs([]table.ColumnConfig{
		{Number: 0, AutoMerge: true},
		{Number: 1, AutoMerge: true},
		{Number: 2, AutoMerge: true},
		{Number: 3, AutoMerge: true},
		{Number: 6, AutoMerge: true},
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

func ptrTimeToString(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.Format("15:04")
}

func timePtr(t time.Time) *time.Time {
	return &t
}
