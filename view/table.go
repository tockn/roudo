package view

import (
	"fmt"
	"os"
	"roudo/roudo"
	"time"

	"github.com/jedib0t/go-pretty/v6/table"
)

type tableViewer struct {
	repo ViewRepository
}

func NewTableViewer(repo ViewRepository) Viewer {
	return &tableViewer{repo: repo}
}

func (t *tableViewer) Do(yearMonth string) error {
	reports, err := t.repo.ListReports(yearMonth)
	if err != nil {
		return err
	}

	tb, err := buildTableWriter(reports)
	if err != nil {
		return err
	}
	tb.Render()
	return nil
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

func calculateTotalWorkTime(rs []roudo.Roudo) time.Duration {
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

func calculateTotalBreakTime(rs []roudo.Roudo) time.Duration {
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
